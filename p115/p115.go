package p115

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

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
	cookies := LoadCookies()
	if cookies != "" {
		agent, err := NewAgent(cookies)
		// cookies is invalid
		if err != nil {
			return nil, err
		}
		return agent, nil
	}
	return nil, errors.New(".cookies is empty or not exist")
}

func NewAgentByQrcode() (*Agent, error) {
	cookies := LoadCookies()
	if cookies != "" {
		agent, err := NewAgent(cookies)
		// cookies is invalid
		if err != nil {
			return QrcodeLogin()
		}
		return agent, nil
	}
	return QrcodeLogin()
}
func NewAgentByConfig() (*Agent, error) {
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
	emptyNum := 0
	filterdItems := make([]rsssite.MagnetItem, 0)
	for _, item := range magnetItems {
		if item.Magnet == "" {
			emptyNum += 1
			continue
		}
		if !ag.StoreInstance.HasItem(item.Magnet) {
			filterdItems = append(filterdItems, item)
		}
	}
	if emptyNum != 0 {
		log.Printf("[warning] [%s] has %d empty task", config.Name, emptyNum)
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
	rssDict := rsssite.ReadRssConfigDict()
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

func SaveCookies(agent *elevengo.Agent) {
	cr := &elevengo.Credential{}
	agent.CredentialExport(cr)
	cookies := fmt.Sprintf("UID=%s; CID=%s; SEID=%s", cr.UID, cr.CID, cr.SEID)
	os.WriteFile(".cookies", []byte(cookies), 0644)
}

func LoadCookies() string {
	// check if .cookies exists
	if _, err := os.Stat(".cookies"); err != nil {
		return ""
	}
	cookies, err := os.ReadFile(".cookies")
	if err != nil {
		return ""
	}
	return string(cookies)
}

func QrcodeLogin() (*Agent, error) {
	agent := elevengo.Default()
	session := &elevengo.QrcodeSession{}
	err := agent.QrcodeStart(session)
	if err != nil {
		return nil, err
	}
	err = DisplayQrcode(session.Image)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	after := now.Add(2 * time.Minute)
	for {
		time.Sleep(200 * time.Millisecond)
		success, err := agent.QrcodePoll(session)
		if success {
			SaveCookies(agent)
			DisposeQrcode()

			return &Agent{
				Agent:         agent,
				StoreInstance: store.New(nil),
			}, nil
		}
		if err != nil && err == elevengo.ErrQrcodeCancelled {
			return nil, errors.New("login cancelled")
		}
		if now.After(after) {
			return nil, errors.New("login timed out")
		}
	}
}
