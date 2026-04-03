package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var fsLsCmd = &cobra.Command{
	Use:   "ls [path]",
	Short: "List directory contents",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)

		target := ""
		if len(args) > 0 {
			target = args[0]
		}

		entries, err := session.Ls(ctx, target)
		if err != nil {
			printFsError(err)
			os.Exit(1)
		}
		printEntries(entries)
	},
}
