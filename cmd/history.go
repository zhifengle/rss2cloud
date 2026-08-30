package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/rivo/uniseg"
	"github.com/spf13/cobra"
	"github.com/zhifengle/rss2cloud/config"
	"github.com/zhifengle/rss2cloud/store"
)

var (
	historyLimit    int
	historyName     string
	historyFullName bool
)

const historyFeedWidth = 16

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "List offline submission history from the local database",
	Run: func(cmd *cobra.Command, args []string) {
		cliParams := buildCLIParams(cmd)
		cfg, _, err := config.LoadWithOptions(cliParams, config.LoadOptions{})
		if err != nil {
			log.Fatalln(err)
		}
		st, err := store.NewWithPath(cfg.Database.Path)
		if err != nil {
			log.Fatalln(err)
		}
		defer st.Close()

		items, err := st.ListHistory(historyLimit, historyName)
		if err != nil {
			log.Fatalln(err)
		}
		if len(items) == 0 {
			fmt.Println("no history records")
			return
		}
		printHistoryItems(os.Stdout, items, historyFullName)
	},
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "number of records (most recent first)")
	historyCmd.Flags().StringVar(&historyName, "name", "", "filter by RSS feed config name")
	historyCmd.Flags().BoolVar(&historyFullName, "full-name", false, "show the original title without tag removal or truncation")
	rootCmd.AddCommand(historyCmd)
}

func printHistoryItems(out io.Writer, items []store.HistoryItem, fullName bool) {
	idWidth := len("ID")
	titleWidth := taskNameWidth
	titles := make([]string, len(items))
	feeds := make([]string, len(items))
	for i := range items {
		if width := len(strconv.FormatInt(items[i].ID, 10)); width > idWidth {
			idWidth = width
		}
		title := items[i].Title
		if !fullName {
			title = compactTaskName(title)
			title = truncateDisplayWidth(title, taskNameWidth)
		} else if width := uniseg.StringWidth(title); width > titleWidth {
			titleWidth = width
		}
		titles[i] = title
		feeds[i] = truncateDisplayWidth(items[i].ConfigName, historyFeedWidth)
	}

	fmt.Fprintf(out, "%*s  %s  %s  %-19s  %s\n",
		idWidth, "ID", padDisplayWidth("TITLE", titleWidth),
		padDisplayWidth("FEED", historyFeedWidth), "CREATED", "HASH")
	for i := range items {
		fmt.Fprintf(out, "%*d  %s  %s  %-19s  %s\n",
			idWidth, items[i].ID, padDisplayWidth(titles[i], titleWidth),
			padDisplayWidth(feeds[i], historyFeedWidth),
			items[i].CreatedAt.Format("2006-01-02 15:04:05"), items[i].TaskHash)
	}
}
