package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deadblue/elevengo"
	"github.com/rivo/uniseg"
)

func TestFormatPercentUsesAPIPercentage(t *testing.T) {
	for _, tc := range []struct {
		input float64
		want  string
	}{
		{input: 0, want: "0%"},
		{input: 42.4, want: "42%"},
		{input: 100, want: "100%"},
	} {
		if got := formatPercent(tc.input); got != tc.want {
			t.Errorf("formatPercent(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFilterOfflineTasks(t *testing.T) {
	tasks := []elevengo.OfflineTask{
		{Name: "done", Status: 2},
		{Name: "failed", Status: -1},
		{Name: "running", Status: 1},
	}
	got := filterOfflineTasks(tasks, "running")
	if len(got) != 1 || got[0].Name != "running" {
		t.Fatalf("filterOfflineTasks returned %#v", got)
	}
}

func TestCompactTaskNameRemovesOnlyKnownReleaseTags(t *testing.T) {
	input := "[字幕组] 作品名 [12] [1080P][Baha][HEVC]"
	want := "[字幕组] 作品名 [12]"
	if got := compactTaskName(input); got != want {
		t.Fatalf("compactTaskName() = %q, want %q", got, want)
	}
}

func TestTruncateAndPadDisplayWidth(t *testing.T) {
	got := truncateDisplayWidth("中文标题 English title", 12)
	if width := uniseg.StringWidth(got); width > 12 {
		t.Fatalf("truncated width = %d, want <= 12: %q", width, got)
	}
	if width := uniseg.StringWidth(padDisplayWidth(got, 12)); width != 12 {
		t.Fatalf("padded width = %d, want 12", width)
	}
}

func TestPrintOfflineTasksAlignsMixedWidthNames(t *testing.T) {
	tasks := []elevengo.OfflineTask{
		{Name: "中文 [1080P]", Size: 1024, Status: 1, Percent: 42},
		{Name: "English [Baha]", Size: 2048, Status: 2, Percent: 100},
	}
	var out bytes.Buffer
	printOfflineTasks(&out, tasks, false)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines: %q", len(lines), out.String())
	}
	for i, token := range []string{"SIZE", "1.0 KB", "2.0 KB"} {
		column := strings.Index(lines[i], token)
		if column < 0 {
			t.Fatalf("line %d does not contain %q: %q", i, token, lines[i])
		}
		wantWidth := taskNameWidth + 2 + 10 - uniseg.StringWidth(token)
		if width := uniseg.StringWidth(lines[i][:column]); width != wantWidth {
			t.Fatalf("line %d size column starts at width %d, want %d", i, width, wantWidth)
		}
	}
	if strings.Contains(out.String(), "1080P") || strings.Contains(out.String(), "Baha") {
		t.Fatalf("compact output still contains release tags: %q", out.String())
	}
}
