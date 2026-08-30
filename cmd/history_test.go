package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/rivo/uniseg"
	"github.com/zhifengle/rss2cloud/store"
)

func TestPrintHistoryItemsCompactsAndAlignsMixedWidthText(t *testing.T) {
	items := []store.HistoryItem{
		{ID: 9, Title: "中文标题 [1080P][Baha]", ConfigName: "动画订阅", CreatedAt: time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local), TaskHash: "hash1"},
		{ID: 100, Title: "English title [HEVC]", ConfigName: "English feed", CreatedAt: time.Date(2026, 8, 30, 11, 0, 0, 0, time.Local), TaskHash: "hash2"},
	}
	var out bytes.Buffer
	printHistoryItems(&out, items, false)

	text := out.String()
	if strings.Contains(text, "1080P") || strings.Contains(text, "Baha") || strings.Contains(text, "HEVC") {
		t.Fatalf("compact output still contains release tags: %q", text)
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines: %q", len(lines), text)
	}
	for i, token := range []string{"TITLE", "中文标题", "English title"} {
		column := strings.Index(lines[i], token)
		if column < 0 {
			t.Fatalf("line %d does not contain %q: %q", i, token, lines[i])
		}
		if width := uniseg.StringWidth(lines[i][:column]); width != 5 {
			t.Fatalf("line %d title starts at width %d, want 5", i, width)
		}
	}
}

func TestPrintHistoryItemsFullNamePreservesOriginalTitle(t *testing.T) {
	item := store.HistoryItem{ID: 1, Title: "作品 [1080P][Baha]"}
	var out bytes.Buffer
	printHistoryItems(&out, []store.HistoryItem{item}, true)
	if !strings.Contains(out.String(), item.Title) {
		t.Fatalf("full-name output lost original title: %q", out.String())
	}
}
