package rsssite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestDmhy(t *testing.T) {
	dmhy := &Dmhy{}
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL("https://share.dmhy.org/topics/rss/rss.xml")
	t.Log(feed)
	for _, item := range feed.Items {
		t.Log(dmhy.GetMagnetItem(item))
	}
}

func TestAcgnx(t *testing.T) {
	acgnx := &Acgnx{}
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL("https://share.acgnx.net/rss.xml")
	for _, item := range feed.Items[:1] {
		t.Log(acgnx.GetMagnetItem(item))
	}
}

func TestAnibt(t *testing.T) {
	anibt := &Anibt{}
	data, err := os.ReadFile("../test/anibt.rss")
	if err != nil {
		t.Fatal(err)
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseString(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) == 0 {
		t.Fatal("expected at least one anibt item")
	}

	got := anibt.GetMagnet(feed.Items[0])
	want := "magnet:?xt=urn:btih:8b8c2f0a461a212b0b7417289376ff243284edc6"
	if got != want {
		t.Fatalf("expected first magnet %q, got %q", want, got)
	}

	for _, item := range feed.Items {
		magnet := anibt.GetMagnet(item)
		if !strings.HasPrefix(magnet, "magnet:?xt=urn:btih:") {
			t.Fatalf("expected magnet URI for %q, got %q", item.Title, magnet)
		}
	}
}

func TestGetRssConfigByURL(t *testing.T) {
	rssConfig := GetRssConfigByURL("http://share.dmhy.org/topics/rss/rss.xml")
	t.Log(rssConfig)
}

func TestGetRssConfigByURLMatchesSchemeAndQueryOrder(t *testing.T) {
	SetRssJsonPath("../rss.json")
	t.Cleanup(func() {
		SetRssJsonPath("")
	})

	rssConfig := GetRssConfigByURL("https://share.dmhy.org/topics/rss/rss.xml?sort_id=2&team_id=0&order=date-desc&keyword=%E6%B0%B4%E6%98%9F%E7%9A%84%E9%AD%94%E5%A5%B3")
	if rssConfig == nil {
		t.Fatal("expected config, got nil")
	}
	if rssConfig.SavePath != "文件夹名称" {
		t.Fatalf("expected savepath 文件夹名称, got %q", rssConfig.SavePath)
	}
	if rssConfig.Cid != "" {
		t.Fatalf("expected empty cid from sample config, got %q", rssConfig.Cid)
	}
}

func TestReadRssConfigDictFromUserConfigDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")

	configDir := filepath.Join(homeDir, ".config", "rss2cloud")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configFile := filepath.Join(configDir, "rss.json")
	configContent := `{"example.com":[{"name":"from-config-dir","url":"https://example.com/rss"}]}`
	if err := os.WriteFile(configFile, []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to create rss config: %v", err)
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
		SetRssJsonPath("")
		RssConfigDict = nil
	})
	SetRssJsonPath("")
	RssConfigDict = nil

	configs := ReadRssConfigDict()
	if configs == nil {
		t.Fatalf("expected rss config to be read")
	}
	got := (*configs)["example.com"]
	if len(got) != 1 || got[0].Name != "from-config-dir" {
		t.Fatalf("unexpected rss config: %#v", got)
	}
}

func TestCompileFilter(t *testing.T) {
	t.Parallel()
	// plain substring rules are not compiled into a regex
	if re := compileFilter("简体内嵌"); re != nil {
		t.Errorf("compileFilter(plain) = %v, want nil", re)
	}
	// /.../-wrapped rules compile into a regex
	if re := compileFilter("/简体|1080p/"); re == nil {
		t.Errorf("compileFilter(/regex/) = nil, want non-nil")
	}
}

func TestMatchRule(t *testing.T) {
	t.Parallel()
	// empty rule matches everything (no whitelist/blacklist restriction)
	if !matchRule("anything", "", nil) {
		t.Errorf("empty rule should match all titles")
	}
	// plain substring
	if !matchRule("简日双语 1080p", "简日双语", nil) {
		t.Errorf("substring present should match")
	}
	if matchRule("简体内嵌 720p", "简日双语", nil) {
		t.Errorf("substring absent should not match")
	}
	// regex (whitelist: match to include; blacklist: match to exclude)
	re := compileFilter("/简体|1080p/")
	if !matchRule("简体内嵌 1080p", "/简体|1080p/", re) {
		t.Errorf("regex match should return true")
	}
	if matchRule("繁體 720p", "/简体|1080p/", re) {
		t.Errorf("regex non-match should return false")
	}
}
