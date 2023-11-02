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

func TestGetRssConfigByURL(t *testing.T) {
	rssConfig := GetRssConfigByURL("http://share.dmhy.org/topics/rss/rss.xml")
	t.Log(rssConfig)
}
