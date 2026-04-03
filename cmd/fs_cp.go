package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var fsCpCmd = &cobra.Command{
	Use:   "cp <src...> <target-dir>",
	Short: "Copy objects into a target directory",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)

		targetDir := args[len(args)-1]
		sources := args[:len(args)-1]

		if err := session.Cp(ctx, targetDir, sources...); err != nil {
			printFsError(err)
			os.Exit(1)
		}
		if fsJSON {
			printJSON(map[string]any{"copied": len(sources), "target": targetDir})
		} else {
			fmt.Printf("copied %d object(s) to %s\n", len(sources), targetDir)
		}
	},
}
