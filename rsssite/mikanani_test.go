package rsssite

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
)

type mikanRSS struct {
	Items []struct {
		Title string `xml:"title"`
		Link  string `xml:"link"`
	} `xml:"channel>item"`
}

func TestMikananiSearchFixture(t *testing.T) {
	data, err := os.ReadFile("../test/Search.xml")
	if err != nil {
		t.Fatal(err)
	}
	checkMikananiFeed(t, string(data))
}

func TestMikananiSearchWithoutExcludeKeepsItems(t *testing.T) {
	data, err := os.ReadFile("../test/Search.xml")
	if err != nil {
		t.Fatal(err)
	}
	feed, err := gofeed.NewParser().ParseString(string(data))
	if err != nil {
		t.Fatal(err)
	}
	config := &RssConfig{
		Name: "WEB-DL", Url: "https://mikanani.me/RSS/Bangumi?bangumiId=3995&subgroupid=583",
		Cid: "fake_cid", SavePath: "ani-test", Expiration: 1,
	}
	items := magnetItemsFromFeed(feed, &Mikanani{}, config)
	if len(items) != len(feed.Items) || len(items) != 100 {
		t.Fatalf("got %d tasks from %d feed items, want 100", len(items), len(feed.Items))
	}
	for i, item := range items {
		if item.Magnet == "" {
			t.Errorf("item %d has no magnet", i)
		}
	}
}

func TestMikananiSearchLiveSnapshot(t *testing.T) {
	path := os.Getenv("RSS2CLOUD_MIKANANI_LIVE_XML")
	if path == "" {
		t.Skip("set RSS2CLOUD_MIKANANI_LIVE_XML to a downloaded live RSS file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	checkMikananiFeed(t, string(data))
}

func checkMikananiFeed(t *testing.T, raw string) {
	t.Helper()
	var source mikanRSS
	if err := xml.Unmarshal([]byte(raw), &source); err != nil {
		t.Fatal(err)
	}
	feed, err := gofeed.NewParser().ParseString(raw)
	if err != nil {
		t.Fatal(err)
	}
	checkMikananiItems(t, feed.Items, len(source.Items))
	for i, item := range feed.Items {
		if item.Title != source.Items[i].Title || item.Link != source.Items[i].Link {
			t.Errorf("item %d: parsed title or link differs from XML", i)
		}
	}
}

func checkMikananiItems(t *testing.T, items []*gofeed.Item, wantCount int) {
	t.Helper()
	if len(items) == 0 || (wantCount >= 0 && len(items) != wantCount) {
		t.Fatalf("parsed %d items, want %d", len(items), wantCount)
	}
	mikan := &Mikanani{}
	for i, item := range items {
		if item.Title == "" || item.Link == "" {
			t.Errorf("item %d: missing title or link", i)
			continue
		}
		hash, ok := strings.CutPrefix(item.Link, "https://mikanani.me/Home/Episode/")
		if !ok || len(hash) != 40 {
			t.Errorf("item %d: unexpected episode link %q", i, item.Link)
			continue
		}
		got := mikan.GetMagnetItem(item)
		if got.Title != item.Title || got.Link != item.Link || got.Magnet != "magnet:?xt=urn:btih:"+hash {
			t.Errorf("item %d: incorrect parsed magnet item: %+v", i, got)
		}
	}
	t.Logf("validated %d Mikanani items", len(items))
}
