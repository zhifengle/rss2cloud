package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var fsMkdirCmd = &cobra.Command{
	Use:   "mkdir <path>",
	Short: "Create a directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)

		entry, err := session.Mkdir(ctx, args[0])
		if err != nil {
			printFsError(err)
			os.Exit(1)
		}
		printEntry(entry)
	},
}
