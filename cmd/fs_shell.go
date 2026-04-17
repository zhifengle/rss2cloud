package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var fsHistoryFile string
var initShellSession = initFsSession

var fsShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start an interactive filesystem shell",
	Long: `Start an interactive filesystem shell.

Directory listings and path completion reuse the shared fs session cache.
Use --list-cache-ttl on the parent fs command to tune freshness, for example:
  rss2cloud fs --list-cache-ttl 5s shell
  rss2cloud fs --list-cache-ttl 0 shell`,
	Example: "rss2cloud fs shell\n" +
		"rss2cloud fs --list-cache-ttl 5s shell\n" +
		"rss2cloud fs --list-cache-ttl 0 shell",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireInteractiveTerminal(os.Stdin); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}

		ctx := context.Background()
		session := initShellSession(ctx, cmd)

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

		if err := printShellOutput(os.Stdout, fmt.Sprintf("%s shell - type 'help' for commands, 'exit' to quit", session.Provider())); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}

		if err := runShellLoop(ctx, session, os.Stdout, fsHistoryFile); err != nil {
			if !errors.Is(err, errNonInteractiveShell) {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			return
		}

		// Persist state on exit.
		if err := saveShellState(shellStateFile, shellPersistedState{
			LastCwd: session.Pwd(),
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save shell state to %s: %v\n", shellStateFile, err)
		}
	},
}

func requireInteractiveTerminal(f *os.File) error {
	if f == nil || !term.IsTerminal(int(f.Fd())) {
		return errNonInteractiveShell
	}
	return nil
}

func init() {
	fsShellCmd.Flags().StringVar(&fsHistoryFile, "history-file", defaultHistoryFile, "path to history file")
	fsCmd.AddCommand(fsShellCmd)
}
