package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var fsRmForce bool

var fsRmCmd = &cobra.Command{
	Use:   "rm <path...>",
	Short: "Delete objects",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)

		if err := session.Rm(ctx, args...); err != nil {
			printFsError(err)
			os.Exit(1)
		}
	},
}

func init() {
	fsRmCmd.Flags().BoolVar(&fsRmForce, "force", false, "skip confirmation (reserved)")
}
