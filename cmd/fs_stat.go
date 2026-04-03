package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var fsStatCmd = &cobra.Command{
	Use:   "stat <path>",
	Short: "Show object metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)

		entry, err := session.Stat(ctx, args[0])
		if err != nil {
			printFsError(err)
			os.Exit(1)
		}
		printEntry(entry)
	},
}
