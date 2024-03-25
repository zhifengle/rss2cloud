package request

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	urlPkg "net/url"
	"os"
	"path"
	"time"
)

var (
	ReqSiteConfig = ReadNodeSiteConfig()
	httpProxy     = "http://127.0.0.1:10809"
	ua            = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"
)

type SiteConfig struct {
	HttpsAgent string            `json:"httpsAgent,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type NodeSiteConfig = map[string]SiteConfig

func ReadNodeSiteConfig() NodeSiteConfig {
	filename := "node-site-config.json"
	config := make(NodeSiteConfig)

	if _, err := os.Stat(filename); err != nil {
		home, _ := os.UserHomeDir()
		filename = path.Join(home, filename)
		if _, err := os.Stat(filename); err != nil {
			return config
		}
	}
	file, _ := os.ReadFile(filename)
	json.Unmarshal(file, &config)
	return config
}

func Request(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ua)

	var p func(*http.Request) (*urlPkg.URL, error)

	curConfig, ok := ReqSiteConfig[req.URL.Host]
	if ok {
		if curConfig.HttpsAgent != "" {
			u, _ := http.ProxyFromEnvironment(req)
			if u == nil {
				proxy, _ := urlPkg.Parse(httpProxy)
				p = http.ProxyURL(proxy)
			} else {
				p = http.ProxyFromEnvironment
			}
		}
		if curConfig.Headers != nil {
			for k, v := range curConfig.Headers {
				req.Header.Set(k, v)
			}
		}
	}

	transport := &http.Transport{
		Proxy:               p,
		DisableCompression:  true,
		TLSHandshakeTimeout: 10 * time.Second,
		// TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		Jar:       jar,
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return client.Do(req)
}

func GetByte(url string, headers map[string]string) ([]byte, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	res, err := Request("GET", url, nil, headers)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var reader io.ReadCloser
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(res.Body)
	case "deflate":
		reader = flate.NewReader(res.Body)
	default:
		reader = res.Body
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return body, nil
}

func Get(url string, headers map[string]string) (string, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	body, err := GetByte(url, headers)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func PostJson(url string, body []byte, headers map[string]string) (string, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = "application/json; charset=UTF-8"
	res, err := Request("POST", url, bytes.NewBuffer(body), headers)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil && err != io.EOF {
		return "", err
	}
	return string(body), nil
}

func PostForm(url string, data urlPkg.Values, headers map[string]string) (string, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	res, err := Request("POST", url, bytes.NewBufferString(data.Encode()), headers)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil && err != io.EOF {
		return "", err
	}
	return string(body), nil
}
