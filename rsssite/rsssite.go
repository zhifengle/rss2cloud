package rsssite

import (
	"log"
	urlPkg "net/url"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/mmcdole/gofeed"
	"github.com/zhifengle/rss2cloud/request"
)

type MagnetSite interface {
	GetMagnet(item *gofeed.Item) string
	GetMagnetItem(item *gofeed.Item) MagnetItem
}

type MagnetItem struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Magnet      string `json:"magnet"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

func getSite(url string) MagnetSite {
	name := url
	if strings.HasPrefix(url, "http") {
		urlObj, _ := urlPkg.Parse(url)
		name = urlObj.Host
	}
	switch name {
	case "mikanani.me", "mikanime.tv":
		return &Mikanani{}
	case "nyaa.si", "sukebei.nyaa.si":
		return &Nyaa{}
	case "share.dmhy.org":
		return &Dmhy{}
	case "share.acgnx.se", "share.acgnx.net", "www.acgnx.se":
		return &Acgnx{}
	case "rsshub.app":
		return &Rsshub{}
	default:
		log.Printf("[error] not support site: [%s]. rss URL: %s\n", name, url)
		return nil
	}
}

func GetFeed(url string) *gofeed.Feed {
	res, err := request.Get(url, nil)
	if err != nil {
		log.Printf("[error] get rss from %s error: %s\n", url, err)
		return nil
	}
	feed, err := gofeed.NewParser().ParseString(res)
	if err != nil {
		log.Printf("[error] parse rss error: %s\n", err)
		return nil
	}
	return feed
}

func GetMagnetItemList(config *RssConfig) []MagnetItem {
	site := getSite(config.Url)
	if site == nil {
		return nil
	}
	feed := GetFeed(config.Url)
	if feed == nil {
		return nil
	}
	var itemList []MagnetItem
	var re *regexp2.Regexp
	if strings.HasPrefix(config.Filter, "/") && strings.HasSuffix(config.Filter, "/") {
		re = regexp2.MustCompile(config.Filter[1:len(config.Filter)-1], 0)
	}
	for _, item := range feed.Items {
		flag := true
		if config.Filter != "" {
			if re != nil {
				flag, _ = re.MatchString(item.Title)
			} else {
				flag = strings.Contains(item.Title, config.Filter)
			}
		}
		if !flag {
			continue
		}
		itemList = append(itemList, site.GetMagnetItem(item))
	}
	return itemList
}

func GetMagnetByEnclosure(item *gofeed.Item) string {
	if len(item.Enclosures) == 0 {
		return ""
	}
	// find enclosure by type == "application/x-bittorrent" or url has prefix magnet:?
	for _, enclosure := range item.Enclosures {
		if enclosure.Type == "application/x-bittorrent" || strings.HasPrefix(enclosure.URL, "magnet:?") {
			lst := strings.Split(item.Enclosures[0].URL, "&dn=")
			if len(lst) != 2 {
				return enclosure.URL
			}
			return lst[0]
		}
	}
	return ""
}
