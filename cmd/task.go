package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/deadblue/elevengo"
	"github.com/rivo/uniseg"
	"github.com/spf13/cobra"
)

var (
	taskLimit    int
	taskStatus   string
	taskFullName bool
)

const taskNameWidth = 48

var taskNameTagPattern = regexp.MustCompile(`[\[【]([^\]】]+)[\]】]`)
var taskResolutionTagPattern = regexp.MustCompile(`^(360|480|540|720|1080|1440|2160|4320)p$`)

var compactTaskNameTags = map[string]bool{
	"4k": true, "8k": true, "uhd": true, "fhd": true, "hd": true,
	"baha": true, "bahamut": true, "bglobal": true, "bilibili": true,
	"cr": true, "crunchyroll": true, "netflix": true, "nf": true,
	"web": true, "webdl": true, "webrip": true, "bluray": true,
	"bdrip": true, "bd": true, "hdtv": true,
	"avc": true, "hevc": true, "h264": true, "h265": true,
	"x264": true, "x265": true, "aac": true, "flac": true,
	"10bit": true, "8bit": true,
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "List live offline download tasks from 115",
	Run: func(cmd *cobra.Command, args []string) {
		initAgent(cmd)
		tasks, err := pAgent.ListOfflineTasks(taskLimit, func(t *elevengo.OfflineTask) bool {
			return offlineTaskMatchesStatus(t, taskStatus)
		})
		if err != nil {
			log.Fatalln(err)
		}
		if len(tasks) == 0 {
			fmt.Println("no offline tasks")
			return
		}
		printOfflineTasks(os.Stdout, tasks, taskFullName)
	},
}

func init() {
	taskCmd.Flags().IntVarP(&taskLimit, "limit", "n", 50, "number of tasks")
	taskCmd.Flags().StringVarP(&taskStatus, "status", "s", "all", "filter: all|done|failed|running")
	taskCmd.Flags().BoolVar(&taskFullName, "full-name", false, "show the original task name without tag removal or truncation")
	rootCmd.AddCommand(taskCmd)
}

func printOfflineTasks(out io.Writer, tasks []elevengo.OfflineTask, fullName bool) {
	nameWidth := taskNameWidth
	names := make([]string, len(tasks))
	for i := range tasks {
		name := tasks[i].Name
		if !fullName {
			name = compactTaskName(name)
			name = truncateDisplayWidth(name, taskNameWidth)
		} else if width := uniseg.StringWidth(name); width > nameWidth {
			nameWidth = width
		}
		names[i] = name
	}

	fmt.Fprintf(out, "%s  %10s  %-7s  %7s\n", padDisplayWidth("NAME", nameWidth), "SIZE", "STATUS", "PERCENT")
	for i := range tasks {
		t := &tasks[i]
		fmt.Fprintf(out, "%s  %10s  %-7s  %7s\n",
			padDisplayWidth(names[i], nameWidth), formatSize(t.Size), offlineStatusText(t), formatPercent(t.Percent))
	}
}

func compactTaskName(name string) string {
	name = taskNameTagPattern.ReplaceAllStringFunc(name, func(tag string) string {
		parts := taskNameTagPattern.FindStringSubmatch(tag)
		if len(parts) != 2 || !isCompactTaskNameTag(parts[1]) {
			return tag
		}
		return ""
	})
	return strings.Join(strings.Fields(name), " ")
}

func isCompactTaskNameTag(tag string) bool {
	normalized := strings.ToLower(tag)
	normalized = strings.NewReplacer(" ", "", "-", "", "_", "", ".", "").Replace(normalized)
	if compactTaskNameTags[normalized] {
		return true
	}
	return taskResolutionTagPattern.MatchString(normalized)
}

func truncateDisplayWidth(s string, maxWidth int) string {
	if maxWidth <= 0 || uniseg.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}

	remainingWidth := maxWidth - 3
	var result strings.Builder
	state := -1
	for len(s) > 0 {
		cluster, rest, width, newState := uniseg.FirstGraphemeClusterInString(s, state)
		if width > remainingWidth {
			break
		}
		result.WriteString(cluster)
		remainingWidth -= width
		s = rest
		state = newState
	}
	return result.String() + "..."
}

func padDisplayWidth(s string, width int) string {
	padding := width - uniseg.StringWidth(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

// filterOfflineTasks keeps tasks matching the requested status.
func filterOfflineTasks(tasks []elevengo.OfflineTask, status string) []elevengo.OfflineTask {
	var out []elevengo.OfflineTask
	for i := range tasks {
		t := tasks[i]
		if offlineTaskMatchesStatus(&t, status) {
			out = append(out, t)
		}
	}
	return out
}

func offlineTaskMatchesStatus(t *elevengo.OfflineTask, status string) bool {
	switch status {
	case "", "all":
		return true
	case "done":
		return t.IsDone()
	case "failed":
		return t.IsFailed()
	case "running":
		return t.IsRunning()
	default:
		return false
	}
}

func offlineStatusText(t *elevengo.OfflineTask) string {
	switch {
	case t.IsDone():
		return "Done"
	case t.IsFailed():
		return "Failed"
	case t.IsRunning():
		return "Running"
	default:
		return "Unknown"
	}
}

// formatSize renders a byte count as a human-readable string (e.g. 1.2 GB).
func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatPercent(percent float64) string {
	return fmt.Sprintf("%.0f%%", percent)
}
