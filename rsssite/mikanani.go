package rsssite

import (
	"fmt"
	"strings"

	"github.com/mmcdole/gofeed"
)

type Mikanani struct{}

func (m *Mikanani) GetMagnet(item *gofeed.Item) string {
	lst := strings.Split(item.Link, "Episode/")
	if len(lst) != 2 {
		return ""
	}
	return fmt.Sprintf("magnet:?xt=urn:btih:%s", lst[1])
}

func (m *Mikanani) GetMagnetItem(item *gofeed.Item) MagnetItem {
	return MagnetItem{
		Title:       item.Title,
		Link:        item.Link,
		Magnet:      m.GetMagnet(item),
		Description: item.Description,
		Content:     item.Content,
	}
}
