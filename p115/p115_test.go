package p115

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zhifengle/rss2cloud/rsssite"
)

func TestSuccessfulMagnetItemsPreservesHashAlignment(t *testing.T) {
	items := []rsssite.MagnetItem{{Title: "one"}, {Title: "two"}, {Title: "three"}}
	var rejected []string
	gotItems, gotHashes := successfulMagnetItems(items, []string{"hash-one", "", "hash-three"}, func(item rsssite.MagnetItem) {
		rejected = append(rejected, item.Title)
	})
	if len(gotItems) != 2 || gotItems[0].Title != "one" || gotItems[1].Title != "three" {
		t.Fatalf("unexpected successful items: %#v", gotItems)
	}
	if len(gotHashes) != 2 || gotHashes[0] != "hash-one" || gotHashes[1] != "hash-three" {
		t.Fatalf("unexpected successful hashes: %#v", gotHashes)
	}
	if len(rejected) != 1 || rejected[0] != "two" {
		t.Fatalf("unexpected rejected items: %#v", rejected)
	}
}

func TestLoadCookiesFromUserConfigDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")

	configDir := filepath.Join(homeDir, ".config", "rss2cloud")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ".cookies"), []byte("UID=1; CID=2; SEID=3; KID=4"), 0o600); err != nil {
		t.Fatalf("failed to write cookies: %v", err)
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

	cookies := LoadCookies()
	if cookies != "UID=1; CID=2; SEID=3; KID=4" {
		t.Fatalf("unexpected cookies: %q", cookies)
	}
}
