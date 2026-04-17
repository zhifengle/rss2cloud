package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zhifengle/rss2cloud/rsssite"
)

func TestStore(t *testing.T) {
	db, _ := sql.Open("sqlite3", ":memory:")
	s := New(db)
	err := s.SaveMagnetItems([]rsssite.MagnetItem{
		{
			Title:       "test",
			Link:        "test",
			Magnet:      "magnet:?xt=urn:btih:aa",
			Description: "test",
			Content:     "test",
		},
		{
			Title:       "test",
			Link:        "test",
			Magnet:      "magnet:?xt=urn:btih:uniquehash&dn=test",
			Description: "test",
			Content:     "test",
		},
	})
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	exists := s.HasItem("magnet:?xt=urn:btih:aa")
	if !exists {
		t.Errorf("Error: %v", err)
	}
	exists = s.HasMagnetByXt("magnet:?xt=urn:btih:uniquehash&dn=test2")
	if !exists {
		t.Errorf("Error: %v", err)
	}
	notExists := s.HasItem("magnet:?xt=urn:btih:bb")
	if notExists {
		t.Errorf("Error: %v", err)
	}
}

func TestNewWithPathCreatesParentDir(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "db.sqlite")
	s, err := NewWithPath(dbPath)
	if err != nil {
		t.Fatalf("NewWithPath failed: %v", err)
	}
	defer s.Close()

	if s.DBInstance == nil {
		t.Fatal("expected DBInstance to be set")
	}
	if err := s.SaveMagnetItems([]rsssite.MagnetItem{{
		Title:  "test",
		Link:   "test",
		Magnet: "magnet:?xt=urn:btih:path",
	}}); err != nil {
		t.Fatalf("SaveMagnetItems failed: %v", err)
	}
}
