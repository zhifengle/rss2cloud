package rsssite

import (
	"encoding/json"
	urlPkg "net/url"
	"os"
	"sort"
	"strings"

	"github.com/zhifengle/rss2cloud/configfile"
)

var (
	RssConfigDict map[string][]RssConfig
	rssJsonPath   string
)

type RssConfig struct {
	Name       string `json:"name"`
	Url        string `json:"url"`
	Cid        string `json:"cid,omitempty"`
	SavePath   string `json:"savepath,omitempty"`
	Filter     string `json:"filter,omitempty"`
	Expiration uint   `json:"expiration,omitempty"`
}

func SetRssJsonPath(p string) {
	rssJsonPath = p
}

func ReadRssConfigDict() *map[string][]RssConfig {
	var (
		file []byte
		err  error
	)
	if rssJsonPath != "" {
		file, err = os.ReadFile(rssJsonPath)
		if err != nil {
			return nil
		}
	} else {
		file, _, err = configfile.ReadFile("rss.json", false)
		if err != nil {
			return nil
		}
	}
	config := make(map[string][]RssConfig)
	json.Unmarshal(file, &config)
	RssConfigDict = config
	return &config
}

func GetRssConfigByURL(url string) *RssConfig {
	urlObj, err := urlPkg.Parse(url)
	if err != nil {
		return nil
	}
	ReadRssConfigDict()
	configs, ok := RssConfigDict[urlObj.Host]
	if !ok {
		return &RssConfig{
			Url: url,
		}
	}
	for _, config := range configs {
		if isSameRSSURL(config.Url, url) {
			return &config
		}
	}
	return &RssConfig{
		Url: url,
	}
}

func isSameRSSURL(left string, right string) bool {
	if left == right {
		return true
	}
	return normalizeRSSURL(left) == normalizeRSSURL(right)
}

func normalizeRSSURL(raw string) string {
	u, err := urlPkg.Parse(raw)
	if err != nil {
		return raw
	}

	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port != "" {
		host = host + ":" + port
	}

	path := strings.TrimSuffix(u.EscapedPath(), "/")
	query := u.Query()
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	queryParts := make([]string, 0)
	for _, key := range keys {
		values := query[key]
		sort.Strings(values)
		escapedKey := urlPkg.QueryEscape(key)
		for _, value := range values {
			queryParts = append(queryParts, escapedKey+"="+urlPkg.QueryEscape(value))
		}
	}

	normalized := host + path
	if len(queryParts) > 0 {
		normalized += "?" + strings.Join(queryParts, "&")
	}
	return normalized
}
