package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zhifengle/rss2cloud/cloudfs"
)

var (
	fsSearchMvType int
	fsSearchMvExt  string
)

var fsSearchMvCmd = &cobra.Command{
	Use:   "search-mv <search-root> <keyword> <target-dir>",
	Aliases: []string{"search_mv"},
	Short: "Search files under a directory and move matches into a target directory",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)

		searchRoot := args[0]
		keyword := strings.TrimSpace(args[1])
		targetDir := args[2]
		if keyword == "" {
			printFsError(fmt.Errorf("keyword must not be empty"))
			os.Exit(1)
		}

		entries, err := session.SearchMove(ctx, searchRoot, keyword, targetDir, cloudfs.SearchOptions{
			FileType: fsSearchMvType,
			ExtName:  fsSearchMvExt,
		})
		if err != nil {
			printFsError(err)
			os.Exit(1)
		}

		if fsJSON {
			printEntries(entries)
			return
		}
		if len(entries) == 0 {
			fmt.Printf("moved 0 matched file(s) to %s\n", targetDir)
			return
		}
		for _, entry := range entries {
			fmt.Printf("moved: %s\n", entry.Name)
		}
	},
}

func init() {
	fsSearchMvCmd.Flags().IntVar(&fsSearchMvType, "type", 0, "file type filter: 0 all, 1 document, 2 image, 3 audio, 4 video, 5 archive, 6 software")
	fsSearchMvCmd.Flags().StringVar(&fsSearchMvExt, "ext", "", "file extension filter (for example mkv)")
}
