package request

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type cookiesResponse struct {
	Cookies map[string]string `json:"cookies"`
}

type postResponse struct {
	Data    string            `json:"data"`
	Form    map[string]string `json:"form"`
	Headers map[string]string `json:"headers"`
	JSON    map[string]string `json:"json"`
	Method  string            `json:"method"`
}

func withIsolatedGlobals(t *testing.T) {
	t.Helper()

	originalConfig := ReqSiteConfig
	originalProxy := httpProxy
	originalUA := ua

	clientMapMu.Lock()
	originalClientMap := clientMap
	clientMap = make(map[string]*http.Client)
	clientMapMu.Unlock()

	ReqSiteConfig = make(NodeSiteConfig)
	httpProxy = "http://127.0.0.1:10809"
	ua = "test-agent"

	t.Cleanup(func() {
		ReqSiteConfig = originalConfig
		httpProxy = originalProxy
		ua = originalUA

		clientMapMu.Lock()
		clientMap = originalClientMap
		clientMapMu.Unlock()
	})
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
		})
	})

	mux.HandleFunc("/gzip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		_, _ = gz.Write([]byte("compressed response"))
	})

	mux.HandleFunc("/cookies", func(w http.ResponseWriter, r *http.Request) {
		resp := cookiesResponse{Cookies: map[string]string{}}
		for _, cookie := range r.Cookies() {
			resp.Cookies[cookie.Name] = cookie.Value
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/cookies/set", func(w http.ResponseWriter, r *http.Request) {
		for key, values := range r.URL.Query() {
			if len(values) == 0 {
				continue
			}
			http.SetCookie(w, &http.Cookie{Name: key, Value: values[0], Path: "/"})
		}
		http.Redirect(w, r, "/cookies", http.StatusFound)
	})

	mux.HandleFunc("/post-form", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form := map[string]string{}
		for key := range r.PostForm {
			form[key] = r.PostForm.Get(key)
		}
		resp := postResponse{
			Form:    form,
			Headers: map[string]string{"Content-Type": r.Header.Get("Content-Type")},
			Method:  r.Method,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/post-json", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		payload := map[string]string{}
		_ = json.Unmarshal(body, &payload)

		resp := postResponse{
			Data:    string(body),
			JSON:    payload,
			Headers: map[string]string{"Content-Type": r.Header.Get("Content-Type")},
			Method:  r.Method,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, strings.Repeat("download-data-", 64))
	})

	return httptest.NewServer(mux)
}

func TestReadNodeSiteConfig(t *testing.T) {
	withIsolatedGlobals(t)

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "node-site-config.json")
	configContent := `{"example.com":{"httpsAgent":"yes","headers":{"X-Test":"1"}}}`
	if err := os.WriteFile(configFile, []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	config := ReadNodeSiteConfig()
	siteConfig, ok := config["example.com"]
	if !ok {
		t.Fatalf("expected example.com config to exist")
	}
	if siteConfig.HttpsAgent != "yes" {
		t.Fatalf("expected httpsAgent to be yes, got %q", siteConfig.HttpsAgent)
	}
	if siteConfig.Headers["X-Test"] != "1" {
		t.Fatalf("expected X-Test header to be 1, got %q", siteConfig.Headers["X-Test"])
	}
}

func TestReadNodeSiteConfigFromUserConfigDir(t *testing.T) {
	withIsolatedGlobals(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")

	configDir := filepath.Join(homeDir, ".config", "rss2cloud")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configFile := filepath.Join(configDir, "node-site-config.json")
	configContent := `{"example.org":{"httpsAgent":"yes","headers":{"X-Config":"home"}}}`
	if err := os.WriteFile(configFile, []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	config := ReadNodeSiteConfig()
	siteConfig, ok := config["example.org"]
	if !ok {
		t.Fatalf("expected example.org config to exist")
	}
	if siteConfig.Headers["X-Config"] != "home" {
		t.Fatalf("expected X-Config header to be home, got %q", siteConfig.Headers["X-Config"])
	}
}

func TestGet(t *testing.T) {
	withIsolatedGlobals(t)
	server := newTestServer(t)
	defer server.Close()

	res, err := Get(server.URL+"/get", nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !strings.Contains(res, `"method":"GET"`) {
		t.Fatalf("expected GET response, got %s", res)
	}
}

func TestGetByteWithGzip(t *testing.T) {
	withIsolatedGlobals(t)
	server := newTestServer(t)
	defer server.Close()

	res, err := GetByte(server.URL+"/gzip", nil)
	if err != nil {
		t.Fatalf("GetByte failed: %v", err)
	}
	if string(res) != "compressed response" {
		t.Fatalf("expected decompressed response, got %q", string(res))
	}
}

func TestSetCookie(t *testing.T) {
	withIsolatedGlobals(t)
	server := newTestServer(t)
	defer server.Close()

	if _, err := Get(server.URL+"/cookies/set?foo=bar", nil); err != nil {
		t.Fatalf("failed to set first cookie: %v", err)
	}

	res, err := Get(server.URL+"/cookies/set?baz=qux", nil)
	if err != nil {
		t.Fatalf("failed to set second cookie: %v", err)
	}

	var result cookiesResponse
	if err := json.Unmarshal([]byte(res), &result); err != nil {
		t.Fatalf("failed to unmarshal cookie response: %v", err)
	}
	if result.Cookies["foo"] != "bar" || result.Cookies["baz"] != "qux" {
		t.Fatalf("expected both cookies to persist, got %#v", result.Cookies)
	}
}

func TestGetWithProxy(t *testing.T) {
	withIsolatedGlobals(t)

	t.Setenv("HTTP_PROXY", "http://127.0.0.1:7890")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:7890")

	req, err := http.NewRequest(http.MethodGet, "https://example.com/resource", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	ReqSiteConfig["example.com"] = SiteConfig{HttpsAgent: "yes"}

	client := getClientByReq(req)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	if transport.Proxy == nil {
		t.Fatalf("expected proxy function to be configured")
	}

	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("proxy function returned error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("expected env proxy to be used, got %v", proxyURL)
	}
}

func TestGetClientByReqConcurrent(t *testing.T) {
	withIsolatedGlobals(t)

	req, err := http.NewRequest(http.MethodGet, "https://example.com/resource", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	ReqSiteConfig["example.com"] = SiteConfig{HttpsAgent: "yes"}

	const goroutines = 32
	clients := make(chan *http.Client, goroutines)
	var wg sync.WaitGroup

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			clients <- getClientByReq(req)
		}()
	}

	wg.Wait()
	close(clients)

	var first *http.Client
	for client := range clients {
		if client == nil {
			t.Fatalf("expected non-nil client")
		}
		if first == nil {
			first = client
			continue
		}
		if client != first {
			t.Fatalf("expected all goroutines to share the same client instance")
		}
	}
}

func TestPostForm(t *testing.T) {
	withIsolatedGlobals(t)
	server := newTestServer(t)
	defer server.Close()

	values := url.Values{}
	values.Add("custname", "testpost")

	res, err := PostForm(server.URL+"/post-form", values, nil)
	if err != nil {
		t.Fatalf("PostForm failed: %v", err)
	}

	var result postResponse
	if err := json.Unmarshal(res, &result); err != nil {
		t.Fatalf("failed to unmarshal form response: %v", err)
	}
	if result.Method != http.MethodPost {
		t.Fatalf("expected POST method, got %s", result.Method)
	}
	if result.Form["custname"] != "testpost" {
		t.Fatalf("expected custname=testpost, got %#v", result.Form)
	}
	if result.Headers["Content-Type"] != "application/x-www-form-urlencoded" {
		t.Fatalf("unexpected content-type: %q", result.Headers["Content-Type"])
	}
}

func TestPostJson(t *testing.T) {
	withIsolatedGlobals(t)
	server := newTestServer(t)
	defer server.Close()

	payload, err := json.Marshal(map[string]string{"custname": "testpost"})
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	res, err := PostJson(server.URL+"/post-json", payload, nil)
	if err != nil {
		t.Fatalf("PostJson failed: %v", err)
	}

	var result postResponse
	if err := json.Unmarshal(res, &result); err != nil {
		t.Fatalf("failed to unmarshal json response: %v", err)
	}
	if result.Method != http.MethodPost {
		t.Fatalf("expected POST method, got %s", result.Method)
	}
	if result.JSON["custname"] != "testpost" {
		t.Fatalf("expected custname=testpost, got %#v", result.JSON)
	}
	if result.Headers["Content-Type"] != "application/json; charset=UTF-8" {
		t.Fatalf("unexpected content-type: %q", result.Headers["Content-Type"])
	}
}

func TestDownloadFile(t *testing.T) {
	withIsolatedGlobals(t)
	server := newTestServer(t)
	defer server.Close()

	res, err := Request(http.MethodGet, server.URL+"/download", nil, map[string]string{})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer res.Body.Close()

	filePath := filepath.Join(t.TempDir(), "file.test")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	defer file.Close()

	written, err := io.Copy(file, res.Body)
	if err != nil {
		t.Fatalf("failed to write download to file: %v", err)
	}
	if written == 0 {
		t.Fatalf("expected downloaded data to be written")
	}
}
