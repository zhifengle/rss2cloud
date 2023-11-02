package rsssite

import (
	"strings"

	"github.com/mmcdole/gofeed"
)

type Dmhy struct {
}

func (d *Dmhy) GetMagnet(item *gofeed.Item) string {
	if item.Enclosures == nil || len(item.Enclosures) == 0 {
		return ""
	}
	lst := strings.Split(item.Enclosures[0].URL, "&dn=")
	if len(lst) != 2 {
		return item.Enclosures[0].URL
	}
	return lst[0]
}

func (d *Dmhy) GetMagnetItem(item *gofeed.Item) MagnetItem {
	return MagnetItem{
		Title:       item.Title,
		Link:        item.Link,
		Magnet:      d.GetMagnet(item),
		Description: item.Description,
		Content:     item.Content,
	}
}
