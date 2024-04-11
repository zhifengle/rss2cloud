package rsssite

import (
	"log"
	urlPkg "net/url"
	"regexp"
	"strings"

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
	default:
		log.Fatalln("unknown site: ", name)
		return nil
	}
}

func GetFeed(url string) *gofeed.Feed {
	res, err := request.Get(url, nil)
	if err != nil {
		return nil
	}
	feed, err := gofeed.NewParser().ParseString(res)
	if err != nil {
		return nil
	}
	return feed
}

func GetMagnetItemList(config *RssConfig) []MagnetItem {
	feed := GetFeed(config.Url)
	site := getSite(config.Url)
	if feed == nil {
		return nil
	}
	var itemList []MagnetItem
	var re *regexp.Regexp
	if strings.HasPrefix(config.Filter, "/") && strings.HasSuffix(config.Filter, "/") {
		re = regexp.MustCompile(config.Filter[1 : len(config.Filter)-1])
	}
	for _, item := range feed.Items {
		flag := true
		if config.Filter != "" {
			if re != nil {
				flag = re.MatchString(item.Title)
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
