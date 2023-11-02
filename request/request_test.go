package request

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"testing"
)

func TestReadNodeSiteConfig(t *testing.T) {
	config := ReadNodeSiteConfig()
	site := "share.dmhy.org"
	siteConfig, ok := config[site]
	if !ok {
		return
	}
	t.Log(siteConfig.HttpsAgent)
}

func TestGet(t *testing.T) {
	url := "https://httpbin.org/ip"
	res, err := Get(url, nil)
	if err != nil {
		t.Error()
	}
	t.Log(res)
}

func TestPostForm(t *testing.T) {
	targetUrl := "https://httpbin.org/post"
	values := url.Values{}
	values.Add("custname", "testpost")
	res, err := PostForm(targetUrl, values, nil)
	if err != nil {
		t.Error()
	}
	t.Log(res)
}

func TestPostJson(t *testing.T) {
	targetUrl := "https://httpbin.org/post"
	post_body_struct := struct {
		Custname string `json:"custname"`
	}{
		Custname: "testpost",
	}
	// convert struct to json bytes
	values, _ := json.Marshal(post_body_struct)
	res, err := PostJson(targetUrl, values, nil)
	if err != nil {
		t.Error()
	}
	t.Log(res)
}

func TestDownloadFile(t *testing.T) {
	targetUrl := "https://cachefly.cachefly.net/10mb.test"
	res, _ := Request("GET", targetUrl, nil, make(map[string]string))
	defer res.Body.Close()
	file, _ := os.Create("file.test")
	_, err := io.Copy(file, res.Body)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Close the file.
	file.Close()
}
