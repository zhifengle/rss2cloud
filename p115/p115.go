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

var disableCache = false
var defaultChunkSize = 200
var chunkDelay = 2
var cooldownMinMs uint
var cooldownMaxMs uint

type Option struct {
	DisableCache  bool
	ChunkDelay    int
	ChunkSize     int
	CooldownMinMs int
	CooldownMaxMs int
}

func SetOption(opt Option) {
	disableCache = opt.DisableCache
	chunkDelay = 2
	if opt.ChunkDelay > 0 {
		chunkDelay = opt.ChunkDelay
	}
	defaultChunkSize = 200
	if opt.ChunkSize > 0 {
		defaultChunkSize = opt.ChunkSize
	}
	cooldownMinMs = 0
	cooldownMaxMs = 0
	if opt.CooldownMinMs > 0 {
		cooldownMinMs = uint(opt.CooldownMinMs)
	}
	if opt.CooldownMaxMs > 0 {
		cooldownMaxMs = uint(opt.CooldownMaxMs)
	}
	if cooldownMinMs > 0 && cooldownMaxMs == 0 {
		cooldownMaxMs = cooldownMinMs
	}
	if cooldownMaxMs > 0 && cooldownMaxMs < cooldownMinMs {
		cooldownMaxMs = cooldownMinMs
	}
}

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
	agent := newElevengoAgent()
	cookiesMap := parseCookies(cookies)
	err := agent.CredentialImport(&elevengo.Credential{
		UID: cookiesMap["UID"], CID: cookiesMap["CID"], SEID: cookiesMap["SEID"],
		KID: cookiesMap["KID"],
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
		if disableCache || !ag.StoreInstance.HasItem(item.Magnet) {
			filterdItems = append(filterdItems, item)
		}
	}
	if emptyNum != 0 {
		log.Printf("[warning] [%s] has %d empty task\n", config.Name, emptyNum)
	}
	if len(filterdItems) == 0 {
		log.Printf("[%s] has 0 task\n", config.Name)
		return
	}
	for i, items := range chunkBy(filterdItems, defaultChunkSize) {
		urls := make([]string, 0)
		for _, item := range items {
			urls = append(urls, item.Magnet)
		}
		_, err := ag.Agent.OfflineAddUrl(urls, &option.OfflineAddOptions{SaveDirId: config.Cid, SavePath: config.SavePath})
		if err != nil {
			log.Printf("Add offline error: %s\n", err)
			return
		}
		log.Printf("[%s] [%s] add %d tasks\n", config.Name, config.Url, len(urls))
		ag.StoreInstance.SaveMagnetItems(filterdItems)
		if i != len(filterdItems)/defaultChunkSize {
			time.Sleep(time.Second * time.Duration(chunkDelay))
		}
	}
}

func (ag *Agent) AddRssUrlTask(url string) {
	config := rsssite.GetRssConfigByURL(url)
	if config == nil {
		pwd := os.Getenv("PWD")
		log.Printf("config not found: %s for url: %s\n", pwd, url)
		return
	}
	magnetItems := rsssite.GetMagnetItemList(config)
	ag.addCloudTasks(magnetItems, config)
}

func (ag *Agent) ExecuteAllRssTask() {
	rssDict := rsssite.ReadRssConfigDict()
	if rssDict == nil {
		pwd := os.Getenv("PWD")
		log.Printf("rss config not found: %s\n", pwd)
		return
	}
	for _, configs := range *rssDict {
		for i, config := range configs {
			magnetItems := rsssite.GetMagnetItemList(&config)
			ag.addCloudTasks(magnetItems, &config)
			if i != len(configs)-1 {
				time.Sleep(time.Second * time.Duration(chunkDelay))
			}
		}
	}
}

func (ag *Agent) AddMagnetTask(magnets []string, cid string, savepath string) {
	for i, urls := range chunkBy(magnets, defaultChunkSize) {
		_, err := ag.Agent.OfflineAddUrl(urls, &option.OfflineAddOptions{SaveDirId: cid, SavePath: savepath})
		if err != nil {
			log.Printf("Add offline error: %s\n", err)
			return
		}
		log.Printf("[magnet] add %d tasks\n", len(urls))
		if i != len(magnets)/defaultChunkSize {
			time.Sleep(time.Second * time.Duration(chunkDelay))
		}
	}
}
func (ag *Agent) OfflineClear(num int) (err error) {
	flag := elevengo.OfflineClearFlag(num)
	return ag.Agent.OfflineClear(flag)
}

func SaveCookies(agent *elevengo.Agent) {
	cr := &elevengo.Credential{}
	agent.CredentialExport(cr)
	cookies := fmt.Sprintf("UID=%s; CID=%s; SEID=%s; KID=%s", cr.UID, cr.CID, cr.SEID, cr.KID)
	os.WriteFile(".cookies", []byte(cookies), 0600)
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
	agent := newElevengoAgent()
	session := &elevengo.QrcodeSession{}
	// @TODO: add option; default is tv
	err := agent.QrcodeStart(session, option.Qrcode().LoginTv())
	if err != nil {
		return nil, err
	}
	err = DisplayQrcode(session.Image)
	if err != nil {
		return nil, err
	}
	after := time.Now().Add(2 * time.Minute)
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
		if time.Now().After(after) {
			return nil, errors.New("login timed out")
		}
	}
}

func newElevengoAgent() *elevengo.Agent {
	opts := option.Agent()
	if cooldownMinMs > 0 || cooldownMaxMs > 0 {
		minMs := cooldownMinMs
		maxMs := cooldownMaxMs
		if maxMs == 0 {
			maxMs = minMs
		}
		if maxMs < minMs {
			maxMs = minMs
		}
		opts.WithCooldown(minMs, maxMs)
	}
	return elevengo.New(opts)
}
