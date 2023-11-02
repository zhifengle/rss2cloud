package rsssite

import (
	"encoding/json"
	urlPkg "net/url"
	"os"
)

var (
	RssConfigDict map[string][]RssConfig
	rssJsonPath   string
)

type RssConfig struct {
	Name       string `json:"name"`
	Url        string `json:"url"`
	Cid        string `json:"cid,omitempty"`
	Filter     string `json:"filter,omitempty"`
	Expiration uint   `json:"expiration,omitempty"`
}

func SetRssJsonPath(p string) {
	rssJsonPath = p
}

func ReadRssConfigDict() *map[string][]RssConfig {
	if rssJsonPath == "" {
		rssJsonPath = "rss.json"
	}
	// read config
	file, err := os.ReadFile(rssJsonPath)
	if err != nil {
		return nil
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
		if config.Url == url {
			return &config
		}
	}
	return &RssConfig{
		Url: url,
	}
}
