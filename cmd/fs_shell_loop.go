package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/zhifengle/rss2cloud/cloudfs"
)

type lineReadState struct {
	skipLeadingLF bool
}

const shellHelp = `Available commands:
  pwd                        print working directory
  ls [path]                  list directory
  cd <dir>                   change directory
  stat <path>                show object metadata
  mkdir <path>               create directory
  rename <path> <new-name>   rename object
  mv <src...> <target-dir>   move objects
  cp <src...> <target-dir>   copy objects
  rm <path...>               delete objects
  history                    show command history
  !N                         re-run history entry N
  help                       show this help
  exit / quit                leave the shell`

// runShellLoop runs the interactive REPL until the user exits or EOF.
// When stdin is a real terminal it switches to raw mode so Tab completion
// and backspace work character-by-character. When stdin is a pipe or
// redirected file it falls back to the same byte-by-byte reader without
// raw mode (used by tests and scripted input).
func runShellLoop(
	ctx context.Context,
	session *cloudfs.Session,
	history *shellHistory,
	in io.Reader,
	out io.Writer,
) {
	readState := &lineReadState{}

	// Attempt raw mode only when in is os.Stdin and it is a real terminal.
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		oldState, err := term.MakeRaw(int(f.Fd()))
		if err == nil {
			defer term.Restore(int(f.Fd()), oldState)
		}
		// In raw mode \r is sent instead of \n on Enter; we handle both below.
	}

	for {
		fmt.Fprintf(out, "%s:%s> ", session.Provider(), session.Pwd())

		line, eof := readLineWithCompletion(ctx, session, in, out, readState)
		line = strings.TrimSpace(line)

		if line != "" {
			// Handle !N history recall before adding to history.
			if strings.HasPrefix(line, "!") {
				recalled, ok := recallHistory(history, line[1:], out)
				if !ok {
					if eof {
						break
					}
					continue
				}
				line = recalled
				fmt.Fprintln(out, line)
			}

			history.add(line)

			if done := dispatchShellCommand(ctx, session, history, out, line); done || eof {
				break
			}
		} else if eof {
			fmt.Fprintln(out)
			break
		}
	}
}

// readLineWithCompletion reads one line from in, handling Tab for completion.
// Returns the line (without trailing newline) and whether EOF was reached.
// Both \n (Unix) and \r (raw-mode Enter / Windows CR) are treated as line
// terminators. When the previous line ended with \r we skip a single leading
// \n on the next call so CRLF input does not create a spurious empty line,
// while raw-mode terminals that only send \r remain non-blocking.
func readLineWithCompletion(
	ctx context.Context,
	session *cloudfs.Session,
	in io.Reader,
	out io.Writer,
	state *lineReadState,
) (string, bool) {
	var buf []byte
	b := make([]byte, 1)
	haveByte := false
	var ch byte

	for {
		if !haveByte {
			n, err := in.Read(b)
			if n == 0 || err != nil {
				// EOF — return whatever is buffered (may be non-empty for unterminated last line).
				return string(buf), true
			}
			ch = b[0]
		} else {
			haveByte = false
		}

		if state != nil && state.skipLeadingLF {
			state.skipLeadingLF = false
			if ch == '\n' {
				continue
			}
		}

		switch ch {
		case '\n':
			// Both \n and \r end the line. In cooked mode the OS converts Enter
			// to \n. In raw mode Enter sends \r. We never peek for a following
			// \n because that would block on a real terminal.
			fmt.Fprintln(out)
			return string(buf), false
		case '\r':
			if state != nil {
				state.skipLeadingLF = true
			}
			fmt.Fprintln(out)
			return string(buf), false
		case '\t':
			line := string(buf)
			candidates := completeInput(ctx, session, line)
			if len(candidates) == 0 {
				fmt.Fprint(out, "\a")
			} else if len(candidates) == 1 {
				completed := completeSuffix(line, candidates[0])
				if strings.HasPrefix(completed, "\x00") {
					// Full token replacement: erase current token and write new one.
					lastSpace := strings.LastIndex(line, " ")
					tokenLen := len(line)
					if lastSpace >= 0 {
						tokenLen = len(line) - lastSpace - 1
					}
					// Erase tokenLen chars from terminal.
					for i := 0; i < tokenLen; i++ {
						fmt.Fprint(out, "\b \b")
					}
					replacement := completed[1:] // strip \x00
					fmt.Fprint(out, replacement)
					if lastSpace >= 0 {
						buf = append(buf[:lastSpace+1], []byte(replacement)...)
					} else {
						buf = []byte(replacement)
					}
				} else {
					fmt.Fprint(out, completed)
					buf = append(buf, []byte(completed)...)
				}
			} else {
				fmt.Fprintln(out)
				for _, c := range candidates {
					fmt.Fprintf(out, "  %s\n", c)
				}
				fmt.Fprintf(out, "%s:%s> %s", session.Provider(), session.Pwd(), string(buf))
			}
		case 127, '\b':
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Fprint(out, "\b \b")
			}
		case 3:
			// Ctrl-C — clear line.
			fmt.Fprintln(out)
			return "", false
		default:
			if ch >= 32 {
				buf = append(buf, ch)
				out.Write([]byte{ch}) //nolint:errcheck
			}
		}
	}
}

// completeSuffix returns the text that should be appended to line to reach
// the candidate. It accounts for quoted candidates: if the candidate is
// quoted (contains spaces), the entire last token on the line is replaced
// by the quoted candidate.
func completeSuffix(line, candidate string) string {
	lastSpace := strings.LastIndex(line, " ")
	var currentToken string
	if lastSpace >= 0 {
		currentToken = line[lastSpace+1:]
	} else {
		currentToken = line
	}

	// If the candidate is quoted, replace the whole current token.
	if quote := leadingQuote(candidate); quote != 0 {
		// currentToken may start with a quote already (user typed 'my).
		// Return the full quoted candidate minus what the user already typed.
		rawToken := stripLeadingQuote(currentToken)
		rawCandidate := candidate[1 : len(candidate)-1] // strip surrounding quotes
		if strings.HasPrefix(rawCandidate, rawToken) {
			// Suffix = rest of raw candidate + closing quote, minus the opening
			// quote the user may or may not have typed.
			suffix := rawCandidate[len(rawToken):] + string(quote)
			if len(currentToken) == 0 || currentToken[0] != quote {
				suffix = string(quote) + rawCandidate + string(quote)
				// We need to replace the whole token, not just append.
				// Signal this by returning the full replacement prefixed with \x00.
				return "\x00" + suffix
			}
			return suffix
		}
		return ""
	}

	// Unquoted candidate — simple prefix match.
	if strings.HasPrefix(candidate, currentToken) {
		return candidate[len(currentToken):]
	}
	return ""
}

func leadingQuote(s string) byte {
	if len(s) < 2 {
		return 0
	}
	if (s[0] == '\'' || s[0] == '"') && s[len(s)-1] == s[0] {
		return s[0]
	}
	return 0
}

// recallHistory looks up history entry by 1-based index string.
// Returns the recalled line and true on success.
func recallHistory(history *shellHistory, indexStr string, out io.Writer) (string, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(indexStr))
	if err != nil || n < 1 || n > len(history.all()) {
		fmt.Fprintf(out, "error: !%s: no such history entry\n", indexStr)
		return "", false
	}
	return history.all()[n-1], true
}

// dispatchShellCommand parses and executes one shell line.
// Returns true when the shell should exit.
func dispatchShellCommand(ctx context.Context, session *cloudfs.Session, history *shellHistory, out io.Writer, line string) bool {
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

	case "history":
		if len(args) != 0 {
			fmt.Fprintln(out, "usage: history")
			break
		}
		for i, e := range history.all() {
			fmt.Fprintf(out, "  %3d  %s\n", i+1, e)
		}

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

// fprintEntry writes a single entry to out (shared with one-shot commands).
func fprintEntry(out io.Writer, e cloudfs.Entry) {
	fmt.Fprintf(out, "id:        %s\n", e.ID)
	if e.ParentID != "" {
		fmt.Fprintf(out, "parent_id: %s\n", e.ParentID)
	}
	fmt.Fprintf(out, "name:      %s\n", e.Name)
	fmt.Fprintf(out, "type:      %s\n", e.Type)
	fmt.Fprintf(out, "size:      %d\n", e.Size)
	if e.PickCode != "" {
		fmt.Fprintf(out, "pick_code: %s\n", e.PickCode)
	}
}

// fprintEntries writes a list of entries to out.
func fprintEntries(out io.Writer, entries []cloudfs.Entry) {
	for _, e := range entries {
		typeChar := "-"
		if e.IsDir() {
			typeChar = "d"
		}
		fmt.Fprintf(out, "%s  %-12s  %s\n", typeChar, e.ID, e.Name)
	}
}
