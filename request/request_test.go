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
	ReqSiteConfig["httpbin.org"] = SiteConfig{
		HttpsAgent: "yes",
	}
	res, err := Get(url, nil)
	if err != nil {
		t.Error()
	}
	t.Log(res)
}

type CookiesResponse struct {
	Cookies map[string]string `json:"cookies"`
}

func TestSetCookie(t *testing.T) {
	url := "https://httpbin.org/cookies/set?foo=bar"
	ReqSiteConfig["httpbin.org"] = SiteConfig{
		HttpsAgent: "yes",
	}
	res, err := Get(url, nil)
	if err != nil {
		t.Error()
	}
	var result CookiesResponse
	err = json.Unmarshal([]byte(res), &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if result.Cookies["foo"] != "bar" {
		t.Errorf("Expected 'foo' to be 'bar', got %v", result.Cookies["foo"])
	}
	// Test setting the second cookie
	url = "https://httpbin.org/cookies/set?baz=qux"
	res, err = Get(url, nil)
	if err != nil {
		t.Fatalf("Failed to set cookie 'baz': %v", err)
	}

	result = CookiesResponse{}
	err = json.Unmarshal([]byte(res), &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if result.Cookies["baz"] != "qux" || result.Cookies["foo"] != "bar" {
		t.Errorf("Expected 'baz' to be 'qux' and 'foo' to be 'bar', got %v and %v", result.Cookies["baz"], result.Cookies["foo"])
	}
}

func TestGetWithProxy(t *testing.T) {
	url := "https://httpbin.org/ip"
	ReqSiteConfig["httpbin.org"] = SiteConfig{
		HttpsAgent: "yes",
	}
	os.Setenv("http_proxy", "socks5://127.0.0.1:7890")
	os.Setenv("https_proxy", "socks5://127.0.0.1:7890")
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
