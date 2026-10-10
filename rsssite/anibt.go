package rsssite

import (
	"encoding/hex"
	"encoding/xml"
	"strings"

	"github.com/mmcdole/gofeed"
)

type Anibt struct {
}

func (r *Anibt) GetMagnet(item *gofeed.Item) string {
	if magnet := parseAnibtTorrentCustom(item.Custom["torrent"]); magnet != "" {
		return trimMagnetDN(magnet)
	}
	// A torrent download URL would create an HTTP download of the .torrent
	// file in 115, rather than an offline task for its contents.
	for _, enclosure := range item.Enclosures {
		magnet := strings.TrimSpace(enclosure.URL)
		if strings.HasPrefix(magnet, "magnet:?") {
			return trimMagnetDN(magnet)
		}
	}
	return ""
}

func (r *Anibt) GetMagnetItem(item *gofeed.Item) MagnetItem {
	return MagnetItem{
		Title:       item.Title,
		Link:        item.Link,
		Magnet:      r.GetMagnet(item),
		Description: item.Description,
		Content:     item.Content,
	}
}

func parseAnibtTorrentCustom(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var torrent struct {
		InfoHash  string `xml:"infohash"`
		MagnetURI string `xml:"magneturi"`
	}
	value = strings.ReplaceAll(value, "&", "&amp;")
	if err := xml.Unmarshal([]byte("<torrent>"+value+"</torrent>"), &torrent); err != nil {
		return ""
	}

	magnet := strings.TrimSpace(torrent.MagnetURI)
	if strings.HasPrefix(magnet, "magnet:?") {
		return magnet
	}

	hash := strings.TrimSpace(torrent.InfoHash)
	if len(hash) != 40 {
		return ""
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return ""
	}
	return "magnet:?xt=urn:btih:" + strings.ToLower(hash)
}

func trimMagnetDN(magnet string) string {
	magnet = strings.TrimSpace(magnet)
	if before, _, ok := strings.Cut(magnet, "&dn="); ok {
		return before
	}
	return magnet
}
