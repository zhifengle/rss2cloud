package store

import (
	"database/sql"
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
	})
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	exists := s.HasItem("magnet:?xt=urn:btih:aa")
	if !exists {
		t.Errorf("Error: %v", err)
	}
	notExists := s.HasItem("magnet:?xt=urn:btih:bb")
	if notExists {
		t.Errorf("Error: %v", err)
	}
}
