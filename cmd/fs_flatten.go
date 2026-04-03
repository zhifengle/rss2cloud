package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zhifengle/rss2cloud/cloudfs"
)

var (
	fsFlattenDryRun        bool
	fsFlattenKeepEmptyDirs bool
)

var fsFlattenCmd = &cobra.Command{
	Use:   "flatten <dir>",
	Short: "Flatten descendant files into the target directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)

		result, err := session.Flatten(ctx, args[0], cloudfs.FlattenOptions{
			DryRun:        fsFlattenDryRun,
			KeepEmptyDirs: fsFlattenKeepEmptyDirs,
		})
		if err != nil {
			printFsError(err)
			os.Exit(1)
		}

		if fsJSON {
			printJSON(map[string]any{
				"target":           toEntryJSON(result.Target),
				"planned_moves":    len(result.PlannedMoves),
				"planned_removals": len(result.PlannedRemovals),
				"moved":            len(result.Moved),
				"removed_dirs":     len(result.RemovedDirs),
				"dry_run":          fsFlattenDryRun,
				"keep_empty_dirs":  fsFlattenKeepEmptyDirs,
			})
			return
		}

		if fsFlattenDryRun {
			fmt.Printf(
				"flatten plan for %s: %d move(s), %d directory removal(s)\n",
				args[0], len(result.PlannedMoves), len(result.PlannedRemovals),
			)
			return
		}

		fmt.Printf(
			"flattened %s: moved %d file(s), removed %d directory(s)\n",
			args[0], len(result.Moved), len(result.RemovedDirs),
		)
	},
}

func init() {
	fsFlattenCmd.Flags().BoolVar(&fsFlattenDryRun, "dry-run", false, "plan flatten actions without applying them")
	fsFlattenCmd.Flags().BoolVar(&fsFlattenKeepEmptyDirs, "keep-empty-dirs", false, "do not remove emptied descendant directories")
}
