package rsssite

import (
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
