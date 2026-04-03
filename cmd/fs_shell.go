package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var fsHistoryFile string

var fsShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start an interactive filesystem shell",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		session := initFsSession(ctx)

		// Restore last cwd from persisted state (only if --cwd was not given).
		if fsCwd == "" {
			state := loadShellState(shellStateFile)
			if state.LastCwd != "" {
				if _, err := session.Cd(ctx, state.LastCwd); err != nil {
					// Silently fall back to root if the saved path no longer exists.
					_ = err
				}
			}
		}

		history := newShellHistory(fsHistoryFile)

		fmt.Fprintf(os.Stdout, "%s shell — type 'help' for commands, 'exit' to quit\n", session.Provider())

		runShellLoop(ctx, session, history, os.Stdin, os.Stdout)

		// Persist state on exit.
		if err := saveShellState(shellStateFile, shellPersistedState{
			LastCwd: session.Pwd(),
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save shell state to %s: %v\n", shellStateFile, err)
		}
		if err := history.save(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save history to %s: %v\n", fsHistoryFile, err)
		}
	},
}

func init() {
	fsShellCmd.Flags().StringVar(&fsHistoryFile, "history-file", defaultHistoryFile, "path to history file")
	fsCmd.AddCommand(fsShellCmd)
}
