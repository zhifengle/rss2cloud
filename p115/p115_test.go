package p115

import (
	"errors"
	"iter"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bytedance/mockey"
	"github.com/deadblue/elevengo"
	"github.com/deadblue/elevengo/option"
	"github.com/zhifengle/rss2cloud/rsssite"
	"github.com/zhifengle/rss2cloud/store"
)

type offlineIteratorStub struct {
	tasks   []elevengo.OfflineTask
	count   int
	yielded int
}

func (it *offlineIteratorStub) Count() int {
	return it.count
}

func (it *offlineIteratorStub) Items() iter.Seq2[int, *elevengo.OfflineTask] {
	return func(yield func(int, *elevengo.OfflineTask) bool) {
		for i := range it.tasks {
			it.yielded++
			if !yield(i, &it.tasks[i]) {
				return
			}
		}
	}
}

func TestListOfflineTasksStopsAtMatchingLimit(t *testing.T) {
	it := &offlineIteratorStub{
		count: 4,
		tasks: []elevengo.OfflineTask{
			{Name: "failed", Status: -1},
			{Name: "running-1", Status: 1},
			{Name: "running-2", Status: 1},
			{Name: "unread", Status: 1},
		},
	}
	patch := mockey.Mock((*elevengo.Agent).OfflineIterate).To(
		func(*elevengo.Agent) (elevengo.Iterator[elevengo.OfflineTask], error) {
			return it, nil
		},
	).Build()
	defer patch.UnPatch()

	ag := &Agent{Agent: &elevengo.Agent{}}
	tasks, err := ag.ListOfflineTasks(2, func(task *elevengo.OfflineTask) bool {
		return task.IsRunning()
	})
	if err != nil {
		t.Fatalf("ListOfflineTasks failed: %v", err)
	}
	if len(tasks) != 2 || tasks[0].Name != "running-1" || tasks[1].Name != "running-2" {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}
	if it.yielded != 3 {
		t.Fatalf("yielded %d tasks, want 3", it.yielded)
	}
}

func TestListOfflineTasksReportsIncompleteIteration(t *testing.T) {
	it := &offlineIteratorStub{
		count: 3,
		tasks: []elevengo.OfflineTask{{Name: "first"}, {Name: "second"}},
	}
	patch := mockey.Mock((*elevengo.Agent).OfflineIterate).To(
		func(*elevengo.Agent) (elevengo.Iterator[elevengo.OfflineTask], error) {
			return it, nil
		},
	).Build()
	defer patch.UnPatch()

	ag := &Agent{Agent: &elevengo.Agent{}}
	if _, err := ag.ListOfflineTasks(20, nil); err == nil {
		t.Fatal("expected incomplete iteration error")
	}
}

func TestAddCloudTasksSavesOnlySuccessfulChunk(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE if not exists `rss_items`").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE if not exists `sites_status`").WillReturnResult(sqlmock.NewResult(0, 0))
	// migrateSchema probes existing columns; report the history columns as
	// present so no ALTER statements are issued.
	mock.ExpectQuery("PRAGMA table_info").
		WillReturnRows(sqlmock.NewRows([]string{"cid", "name", "type", "notnull", "dflt_value", "pk"}).
			AddRow(0, "configName", "VARCHAR(255)", 0, nil, 0).
			AddRow(1, "configUrl", "VARCHAR(255)", 0, nil, 0).
			AddRow(2, "taskHash", "VARCHAR(255)", 0, nil, 0))

	for range 3 {
		mock.ExpectQuery("SELECT count\\(\\*\\) AS num FROM rss_items WHERE magnet = \\?").
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"num"}).AddRow(0))
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO rss_items").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "ok", 0, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	callCount := 0
	patch := mockey.Mock((*elevengo.Agent).OfflineAddUrl).To(
		func(_ *elevengo.Agent, _ []string, _ ...*option.OfflineAddOptions) ([]string, error) {
			callCount++
			if callCount == 2 {
				return nil, errors.New("second chunk failed")
			}
			return []string{"ok", ""}, nil
		},
	).Build()
	defer patch.UnPatch()

	ag := &Agent{
		Agent:         &elevengo.Agent{},
		StoreInstance: store.New(db),
	}

	defaultChunkSize = 2
	t.Cleanup(func() {
		defaultChunkSize = 200
	})

	ag.addCloudTasks([]rsssite.MagnetItem{
		{Magnet: "magnet:?xt=urn:btih:1", Title: "1"},
		{Magnet: "magnet:?xt=urn:btih:2", Title: "2"},
		{Magnet: "magnet:?xt=urn:btih:3", Title: "3"},
	}, &rsssite.RssConfig{Name: "test", Url: "https://example.com/rss", Cid: "cid"})

	if callCount != 2 {
		t.Fatalf("expected 2 add attempts, got %d", callCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected db interactions: %v", err)
	}
}

func TestSuccessfulMagnetItemsPreservesHashAlignment(t *testing.T) {
	items := []rsssite.MagnetItem{{Title: "one"}, {Title: "two"}, {Title: "three"}}
	var rejected []string
	gotItems, gotHashes := successfulMagnetItems(items, []string{"hash-one", "", "hash-three"}, func(item rsssite.MagnetItem) {
		rejected = append(rejected, item.Title)
	})
	if len(gotItems) != 2 || gotItems[0].Title != "one" || gotItems[1].Title != "three" {
		t.Fatalf("unexpected successful items: %#v", gotItems)
	}
	if len(gotHashes) != 2 || gotHashes[0] != "hash-one" || gotHashes[1] != "hash-three" {
		t.Fatalf("unexpected successful hashes: %#v", gotHashes)
	}
	if len(rejected) != 1 || rejected[0] != "two" {
		t.Fatalf("unexpected rejected items: %#v", rejected)
	}
}

func TestQrcodeLoginTimeout(t *testing.T) {
	fakeNow := time.Unix(0, 0)

	patchDefault := mockey.Mock(elevengo.Default).To(func() *elevengo.Agent {
		return &elevengo.Agent{}
	}).Build()
	defer patchDefault.UnPatch()

	patchNew := mockey.Mock(elevengo.New).To(func(_ ...*option.AgentOptions) *elevengo.Agent {
		return &elevengo.Agent{}
	}).Build()
	defer patchNew.UnPatch()

	patchNow := mockey.Mock(time.Now).To(func() time.Time {
		return fakeNow
	}).Build()
	defer patchNow.UnPatch()

	patchSleep := mockey.Mock(time.Sleep).To(func(d time.Duration) {
		fakeNow = fakeNow.Add(d)
	}).Build()
	defer patchSleep.UnPatch()

	patchStart := mockey.Mock((*elevengo.Agent).QrcodeStart).To(
		func(_ *elevengo.Agent, session *elevengo.QrcodeSession, _ ...*option.QrcodeOptions) error {
			session.Image = []byte("fake")
			return nil
		},
	).Build()
	defer patchStart.UnPatch()

	patchPoll := mockey.Mock((*elevengo.Agent).QrcodePoll).To(
		func(_ *elevengo.Agent, _ *elevengo.QrcodeSession) (bool, error) {
			return false, nil
		},
	).Build()
	defer patchPoll.UnPatch()

	patchDisplay := mockey.Mock(DisplayQrcode).To(func([]byte) error {
		return nil
	}).Build()
	defer patchDisplay.UnPatch()

	patchDispose := mockey.Mock(DisposeQrcode).To(func() {}).Build()
	defer patchDispose.UnPatch()

	agent, err := QrcodeLogin()
	if agent != nil {
		t.Fatalf("expected nil agent on timeout, got %#v", agent)
	}
	if err == nil || err.Error() != "login timed out" {
		t.Fatalf("expected login timed out error, got %v", err)
	}
}

func TestLoadCookiesFromUserConfigDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")

	configDir := filepath.Join(homeDir, ".config", "rss2cloud")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ".cookies"), []byte("UID=1; CID=2; SEID=3; KID=4"), 0o600); err != nil {
		t.Fatalf("failed to write cookies: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	cookies := LoadCookies()
	if cookies != "UID=1; CID=2; SEID=3; KID=4" {
		t.Fatalf("unexpected cookies: %q", cookies)
	}
}
