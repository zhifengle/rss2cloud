package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var fsMvCmd = &cobra.Command{
	Use:   "mv <src...> <target-dir>",
	Short: "Move objects into a target directory",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx, cmd)

		targetDir := args[len(args)-1]
		sources := args[:len(args)-1]

		entries, err := session.Mv(ctx, targetDir, sources...)
		if err != nil {
			printFsError(err)
			os.Exit(1)
		}
		if fsJSON {
			printEntries(entries)
		} else {
			for _, e := range entries {
				fmt.Printf("moved: %s\n", e.Name)
			}
		}
	},
}
