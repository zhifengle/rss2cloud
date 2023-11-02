package rsssite

import (
	"fmt"

	"github.com/mmcdole/gofeed"
)

type Nyaa struct {
}

func (n *Nyaa) GetMagnet(item *gofeed.Item) string {
	if item.Extensions["nyaa"] == nil {
		return ""
	}
	if item.Extensions["nyaa"]["infoHash"] == nil {
		return ""
	}
	if len(item.Extensions["nyaa"]["infoHash"]) == 0 {
		return ""
	}
	return fmt.Sprintf("magnet:?xt=urn:btih:%s", item.Extensions["nyaa"]["infoHash"][0].Value)
}

func (n *Nyaa) GetMagnetItem(item *gofeed.Item) MagnetItem {
	return MagnetItem{
		Title:       item.Title,
		Link:        item.Link,
		Magnet:      n.GetMagnet(item),
		Description: item.Description,
		Content:     item.Content,
	}
}
