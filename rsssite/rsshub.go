package rsssite

import (
	"github.com/mmcdole/gofeed"
)

type Rsshub struct {
}

func (r *Rsshub) GetMagnet(item *gofeed.Item) string {
	return GetMagnetByEnclosure(item)
}

func (r *Rsshub) GetMagnetItem(item *gofeed.Item) MagnetItem {
	return MagnetItem{
		Title:       item.Title,
		Link:        item.Link,
		Magnet:      r.GetMagnet(item),
		Description: item.Description,
		Content:     item.Content,
	}
}
