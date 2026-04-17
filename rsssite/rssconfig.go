package rsssite

import (
	urlPkg "net/url"
	"sort"
	"strings"

	"github.com/zhifengle/rss2cloud/config"
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
	// Use config.Load() with RSSPath from SetRssJsonPath()
	cfg, _, err := config.LoadWithOptions(config.CLIParams{RSSPath: rssJsonPath}, config.LoadOptions{RSS: true})
	if err != nil {
		return nil
	}

	// Convert config.RssConfig to rsssite.RssConfig
	result := make(map[string][]RssConfig)
	for site, configs := range cfg.RSS {
		siteConfigs := make([]RssConfig, len(configs))
		for i, c := range configs {
			siteConfigs[i] = RssConfig{
				Name:       c.Name,
				Url:        c.Url,
				Cid:        c.Cid,
				SavePath:   c.SavePath,
				Filter:     c.Filter,
				Expiration: c.Expiration,
			}
		}
		result[site] = siteConfigs
	}

	RssConfigDict = result
	return &result
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
