package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var fsRenameCmd = &cobra.Command{
	Use:   "rename <path> <new-name>",
	Short: "Rename an object (basename only)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)

		entry, err := session.Rename(ctx, args[0], args[1])
		if err != nil {
			printFsError(err)
			os.Exit(1)
		}
		printEntry(entry)
	},
}
