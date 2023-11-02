package p115

import (
	"errors"
	"log"
	"strings"

	"github.com/deadblue/elevengo"
	"github.com/deadblue/elevengo/option"
	"github.com/zhifengle/rss2cloud/request"
	"github.com/zhifengle/rss2cloud/rsssite"
	"github.com/zhifengle/rss2cloud/store"
)

type Agent struct {
	Agent         *elevengo.Agent
	StoreInstance *store.Store
}

func parseCookies(cookiesString string) map[string]string {
	cookies := make(map[string]string)

	// Split the cookies string into individual cookies
	cookiePairs := strings.Split(cookiesString, ";")

	// Parse each cookie into key-value pair
	for _, cookiePair := range cookiePairs {
		cookie := strings.TrimSpace(cookiePair)
		cookieParts := strings.SplitN(cookie, "=", 2)
		if len(cookieParts) == 2 {
			key := cookieParts[0]
			value := cookieParts[1]
			cookies[key] = value
		}
	}

	return cookies
}
func New() (*Agent, error) {
	config := request.ReadNodeSiteConfig()
	if p115Config, ok := config["115.com"]; ok {
		cookies, ok := p115Config.Headers["cookie"]
		if !ok {
			cookies = p115Config.Headers["Cookie"]
		}
		if cookies == "" {
			return nil, errors.New("115 cookie is empty")
		}
		return NewAgent(cookies)
	}
	return nil, errors.New("no 115.com config in node-site-config.json")
}

func NewAgent(cookies string) (*Agent, error) {
	agent := elevengo.Default()
	cookiesMap := parseCookies(cookies)
	err := agent.CredentialImport(&elevengo.Credential{
		UID: cookiesMap["UID"], CID: cookiesMap["CID"], SEID: cookiesMap["SEID"],
	})
	if err != nil {
		return nil, err
	}
	return &Agent{
		Agent:         agent,
		StoreInstance: store.New(nil),
	}, nil
}

func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}

func (ag *Agent) addCloudTasks(magnetItems []rsssite.MagnetItem, config *rsssite.RssConfig) {
	filterdItems := make([]rsssite.MagnetItem, 0)
	for _, item := range magnetItems {
		if !ag.StoreInstance.HasItem(item.Magnet) {
			filterdItems = append(filterdItems, item)
		}
	}
	if len(filterdItems) == 0 {
		log.Printf("[%s] has 0 task", config.Name)
		return
	}
	for _, items := range chunkBy(filterdItems, 200) {
		urls := make([]string, 0)
		for _, item := range items {
			urls = append(urls, item.Magnet)
		}
		_, err := ag.Agent.OfflineAddUrl(urls, option.OfflineSaveDownloadedFileTo(config.Cid))
		if err != nil {
			log.Printf("Add offline error: %s\n", err)
			return
		}
		log.Printf("[%s] [%s] add %d tasks\n", config.Name, config.Url, len(urls))
		ag.StoreInstance.SaveMagnetItems(filterdItems)
	}
}

func (ag *Agent) AddRssUrlTask(url string) {
	config := rsssite.GetRssConfigByURL(url)
	if config == nil {
		return
	}
	magnetItems := rsssite.GetMagnetItemList(config)
	ag.addCloudTasks(magnetItems, config)
}

func (ag *Agent) ExecuteAllRssTask() {
	rssDict := rsssite.ReadRssConfigDict("")
	for _, configs := range *rssDict {
		for _, config := range configs {
			magnetItems := rsssite.GetMagnetItemList(&config)
			ag.addCloudTasks(magnetItems, &config)
		}
	}
}

func (ag *Agent) AddMagnetTask(magnets []string, cid string) {
	for _, urls := range chunkBy(magnets, 200) {
		_, err := ag.Agent.OfflineAddUrl(urls, option.OfflineSaveDownloadedFileTo(cid))
		if err != nil {
			log.Printf("Add offline error: %s\n", err)
			return
		}
		log.Printf("[magnet] add %d tasks\n", len(urls))
	}
}
