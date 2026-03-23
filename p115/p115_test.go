package p115

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bytedance/mockey"
	"github.com/deadblue/elevengo"
	"github.com/deadblue/elevengo/option"
	"github.com/zhifengle/rss2cloud/rsssite"
	"github.com/zhifengle/rss2cloud/store"
)

func TestAddCloudTasksSavesOnlySuccessfulChunk(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New failed: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE if not exists `rss_items`").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE if not exists `sites_status`").WillReturnResult(sqlmock.NewResult(0, 0))

	for range 3 {
		mock.ExpectQuery("SELECT count\\(\\*\\) AS num FROM rss_items WHERE magnet = \\?").
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"num"}).AddRow(0))
	}

	for range 2 {
		mock.ExpectExec("INSERT INTO rss_items").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 0, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	callCount := 0
	patch := mockey.Mock((*elevengo.Agent).OfflineAddUrl).To(
		func(_ *elevengo.Agent, _ []string, _ ...*option.OfflineAddOptions) ([]string, error) {
			callCount++
			if callCount == 2 {
				return nil, errors.New("second chunk failed")
			}
			return []string{"ok"}, nil
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
