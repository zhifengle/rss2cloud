package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/reeflective/readline"

	"github.com/zhifengle/rss2cloud/cloudfs"
)

var errNonInteractiveShell = errors.New("fs shell requires an interactive terminal")

const shellHelp = `Available commands:
  pwd                        print working directory
  ls [path]                  list directory
  cd <dir>                   change directory
  stat <path>                show object metadata
  mkdir <path>               create directory
  rename <path> <new-name>   rename object
  mv <src...> <target-dir>   move objects
  cp <src...> <target-dir>   copy objects
	flatten <dir>              flatten descendant files into dir
	search-mv <root> <keyword> <target-dir>
	search_mv <root> <keyword> <target-dir>
	                             search files and move matches
  rm <path...>               delete objects
  refresh                    clear session cache
  help                       show this help
  exit / quit                leave the shell

Line editing, history navigation, and search are provided by reeflective/readline.`

func runShellLoop(ctx context.Context, session *cloudfs.Session, out io.Writer, historyFile string) error {
	if runtime.GOOS == "windows" {
		return runBasicShellLoop(ctx, session, out, historyFile)
	}

	return runReadlineShellLoop(ctx, session, out, historyFile)
}

func runReadlineShellLoop(ctx context.Context, session *cloudfs.Session, out io.Writer, historyFile string) error {
	rl, err := newShellReadline(ctx, session, historyFile)
	if err != nil {
		return err
	}

	for {
		line, err := rl.Readline()
		if err != nil {
			switch {
			case errors.Is(err, io.EOF):
				fmt.Fprintln(out)
				return nil
			case errors.Is(err, readline.ErrInterrupt):
				continue
			default:
				return err
			}
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var buf bytes.Buffer
		done := dispatchShellCommand(ctx, session, &buf, line)
		if err := printShellOutput(out, buf.String()); err != nil {
			return err
		}
		if done {
			return nil
		}
	}
}

func runBasicShellLoop(ctx context.Context, session *cloudfs.Session, out io.Writer, historyFile string) error {
	history, err := initShellHistory(historyFile)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		if err := printShellPrompt(out, session); err != nil {
			return err
		}

		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		if errors.Is(readErr, io.EOF) && line == "" {
			if err := printShellOutput(out, ""); err != nil {
				return err
			}
			return nil
		}

		line = strings.TrimRight(line, "\r\n")
		line = strings.TrimSpace(line)
		if line == "" {
			if errors.Is(readErr, io.EOF) {
				return nil
			}
			continue
		}

		if history != nil {
			_, _ = history.Write(line)
		}

		var buf bytes.Buffer
		done := dispatchShellCommand(ctx, session, &buf, line)
		if err := printShellOutput(out, buf.String()); err != nil {
			return err
		}
		if done || errors.Is(readErr, io.EOF) {
			return nil
		}
	}
}

func newShellReadline(ctx context.Context, session *cloudfs.Session, historyFile string) (*readline.Shell, error) {
	history, err := initShellHistory(historyFile)
	if err != nil {
		return nil, err
	}

	rl := readline.NewShell()
	rl.Prompt.Primary(func() string {
		return fmt.Sprintf("%s:%s> ", session.Provider(), session.Pwd())
	})
	rl.History.Delete()
	rl.History.Add("rss2cloud fs shell", history)
	rl.Completer = func(line []rune, cursor int) readline.Completions {
		return shellReadlineCompletions(ctx, session, string(line), cursor)
	}

	return rl, nil
}

func printShellOutput(out io.Writer, msg string) error {
	msg = strings.TrimRight(msg, "\r\n")
	if msg == "" {
		if runtime.GOOS == "windows" {
			_, err := io.WriteString(out, "\r\n")
			return err
		}
		_, err := io.WriteString(out, "\n")
		return err
	}

	if runtime.GOOS == "windows" {
		msg = strings.ReplaceAll(msg, "\r\n", "\n")
		msg = strings.ReplaceAll(msg, "\n", "\r\n")
	}

	if _, err := io.WriteString(out, msg); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_, err := io.WriteString(out, "\r\n")
		return err
	}
	if _, err := io.WriteString(out, "\n"); err != nil {
		return err
	}
	return nil
}

func printShellPrompt(out io.Writer, session *cloudfs.Session) error {
	_, err := io.WriteString(out, fmt.Sprintf("%s:%s> ", session.Provider(), session.Pwd()))
	return err
}

// dispatchShellCommand parses and executes one shell line.
// Returns true when the shell should exit.
func dispatchShellCommand(ctx context.Context, session *cloudfs.Session, out io.Writer, line string) bool {
	tokens := parseShellLine(line)
	if len(tokens) == 0 {
		return false
	}
	cmd, args := tokens[0], tokens[1:]

	switch cmd {
	case "exit", "quit":
		return true

	case "help":
		fmt.Fprintln(out, shellHelp)

	case "refresh":
		if len(args) != 0 {
			fmt.Fprintln(out, "usage: refresh")
			break
		}
		session.Refresh()
		fmt.Fprintln(out, "cache cleared")

	case "pwd":
		if len(args) != 0 {
			fmt.Fprintln(out, "usage: pwd")
			break
		}
		fmt.Fprintln(out, session.Pwd())

	case "ls":
		if len(args) > 1 {
			fmt.Fprintln(out, "usage: ls [path]")
			break
		}
		path := ""
		if len(args) == 1 {
			path = args[0]
		}
		entries, err := session.Ls(ctx, path)
		if err != nil {
			shellError(out, err)
			break
		}
		fprintEntries(out, entries)

	case "cd":
		if len(args) != 1 {
			fmt.Fprintln(out, "usage: cd <dir>")
			break
		}
		if _, err := session.Cd(ctx, args[0]); err != nil {
			shellError(out, err)
		}

	case "stat":
		if len(args) != 1 {
			fmt.Fprintln(out, "usage: stat <path>")
			break
		}
		entry, err := session.Stat(ctx, args[0])
		if err != nil {
			shellError(out, err)
			break
		}
		fprintEntry(out, entry)

	case "mkdir":
		if len(args) != 1 {
			fmt.Fprintln(out, "usage: mkdir <path>")
			break
		}
		entry, err := session.Mkdir(ctx, args[0])
		if err != nil {
			shellError(out, err)
			break
		}
		fprintEntry(out, entry)

	case "rename":
		if len(args) != 2 {
			fmt.Fprintln(out, "usage: rename <path> <new-name>")
			break
		}
		entry, err := session.Rename(ctx, args[0], args[1])
		if err != nil {
			shellError(out, err)
			break
		}
		fprintEntry(out, entry)

	case "mv":
		if len(args) < 2 {
			fmt.Fprintln(out, "usage: mv <src...> <target-dir>")
			break
		}
		targetDir := args[len(args)-1]
		sources := args[:len(args)-1]
		entries, err := session.Mv(ctx, targetDir, sources...)
		if err != nil {
			shellError(out, err)
			break
		}
		for _, e := range entries {
			fmt.Fprintf(out, "moved: %s\n", e.Name)
		}

	case "cp":
		if len(args) < 2 {
			fmt.Fprintln(out, "usage: cp <src...> <target-dir>")
			break
		}
		targetDir := args[len(args)-1]
		sources := args[:len(args)-1]
		if err := session.Cp(ctx, targetDir, sources...); err != nil {
			shellError(out, err)
			break
		}
		fmt.Fprintf(out, "copied %d object(s) to %s\n", len(sources), targetDir)

	case "flatten":
		if len(args) != 1 {
			fmt.Fprintln(out, "usage: flatten <dir>")
			break
		}
		result, err := session.Flatten(ctx, args[0], cloudfs.FlattenOptions{})
		if err != nil {
			shellError(out, err)
			break
		}
		fmt.Fprintf(
			out,
			"flattened %s: moved %d file(s), removed %d directory(s)\n",
			args[0], len(result.Moved), len(result.RemovedDirs),
		)

	case "search-mv", "search_mv":
		if len(args) != 3 {
			fmt.Fprintln(out, "usage: search-mv <root> <keyword> <target-dir>")
			break
		}
		entries, err := session.SearchMove(ctx, args[0], args[1], args[2], cloudfs.SearchOptions{})
		if err != nil {
			shellError(out, err)
			break
		}
		if len(entries) == 0 {
			fmt.Fprintf(out, "moved 0 matched file(s) to %s\n", args[2])
			break
		}
		for _, e := range entries {
			fmt.Fprintf(out, "moved: %s\n", e.Name)
		}

	case "rm":
		if len(args) == 0 {
			fmt.Fprintln(out, "usage: rm <path...>")
			break
		}
		if err := session.Rm(ctx, args...); err != nil {
			shellError(out, err)
		}

	default:
		fmt.Fprintf(out, "unknown command: %s (type 'help' for list)\n", cmd)
	}
	return false
}

// shellError prints an error in shell-friendly format.
func shellError(out io.Writer, err error) {
	fmt.Fprintf(out, "error: %v\n", err)
}

