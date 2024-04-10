package rsssite

import "github.com/mmcdole/gofeed"

type Acgnx struct {
}

func (a *Acgnx) GetMagnet(item *gofeed.Item) string {
	if item.Enclosures == nil || len(item.Enclosures) == 0 {
		return ""
	}
	return item.Enclosures[0].URL
}

func (a *Acgnx) GetMagnetItem(item *gofeed.Item) MagnetItem {
	return MagnetItem{
		Title:       item.Title,
		Link:        item.Link,
		Magnet:      a.GetMagnet(item),
		Description: item.Description,
		Content:     item.Content,
	}
}
