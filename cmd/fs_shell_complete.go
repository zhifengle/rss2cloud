package cmd

import (
	"context"
	"strings"

	"github.com/reeflective/readline"

	"github.com/zhifengle/rss2cloud/cloudfs"
)

// shellCommands is the fixed set of commands the shell understands.
var shellCommands = []string{
	"pwd", "ls", "cd", "stat", "mkdir", "rename", "mv", "cp", "flatten", "search-mv", "search_mv", "rm",
	"refresh", "help", "exit", "quit",
}

func readlineCompletionPrefix(line string) string {
	trimmed := line
	if idx := strings.LastIndexAny(trimmed, " \t"); idx >= 0 {
		return trimmed[idx+1:]
	}
	return trimmed
}

func readlineCompletionCandidates(candidates []string, tag string) []readline.Completion {
	completions := make([]readline.Completion, 0, len(candidates))
	for _, candidate := range candidates {
		completions = append(completions, readline.Completion{
			Value:   candidate,
			Display: candidate,
			Tag:     tag,
		})
	}
	return completions
}

func shellReadlineCompletions(ctx context.Context, session *cloudfs.Session, line string, cursor int) readline.Completions {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(line) {
		cursor = len(line)
	}

	current := line[:cursor]
	prefix := readlineCompletionPrefix(current)
	tokens := parseShellLine(current)
	tag := "paths"
	if len(tokens) == 0 || (len(tokens) == 1 && !strings.HasSuffix(current, " ")) {
		tag = "commands"
	}

	comps := readline.CompleteRaw(readlineCompletionCandidates(completeInput(ctx, session, current), tag))
	comps.PREFIX = prefix
	if tag == "paths" {
		comps = comps.NoSpace('/')
	}
	return comps
}

// completeInput returns completion candidates for the current input line.
// It is called when the user presses Tab.
func completeInput(ctx context.Context, session *cloudfs.Session, line string) []string {
	tokens := parseShellLine(line)

	// No tokens yet — complete command names.
	if len(tokens) == 0 || (len(tokens) == 1 && !strings.HasSuffix(line, " ")) {
		prefix := ""
		if len(tokens) == 1 {
			prefix = tokens[0]
		}
		return filterPrefix(shellCommands, prefix)
	}

	// At least one token and a trailing space — complete path argument.
	// Use the last token as the path prefix to complete.
	var pathPrefix string
	if !strings.HasSuffix(line, " ") && len(tokens) > 1 {
		pathPrefix = tokens[len(tokens)-1]
	}
	return completePath(ctx, session, pathPrefix)
}

// completePath lists entries in the parent directory of pathPrefix and returns
// names that match the basename prefix. Directories are returned first.
// Candidates are shell-escaped so spaces and quote characters remain usable.
func completePath(ctx context.Context, session *cloudfs.Session, pathPrefix string) []string {
	// Strip surrounding quotes from pathPrefix before matching.
	rawPrefix := stripLeadingQuote(pathPrefix)

	parentPath := "."
	namePrefix := rawPrefix
	if idx := strings.LastIndex(rawPrefix, "/"); idx >= 0 {
		parentPath = rawPrefix[:idx+1]
		namePrefix = rawPrefix[idx+1:]
	}

	entries, err := session.Ls(ctx, parentPath)
	if err != nil {
		return nil
	}

	var dirs, files []string
	for _, e := range entries {
		if !strings.HasPrefix(e.Name, namePrefix) {
			continue
		}
		rawCandidate := e.Name
		if parentPath != "." {
			rawCandidate = parentPath + e.Name
		}
		if e.IsDir() {
			rawCandidate += "/"
		}
		candidate := shellEscapeToken(rawCandidate)
		if e.IsDir() {
			dirs = append(dirs, candidate)
		} else {
			files = append(files, candidate)
		}
	}
	return append(dirs, files...)
}

// shellEscapeToken returns a representation that parseShellLine can round-trip.
// Prefer simple quoting when possible; otherwise fall back to backslash escapes.
func shellEscapeToken(s string) string {
	if !strings.ContainsAny(s, " \t'\"\\") {
		return s
	}
	if !strings.ContainsRune(s, '\'') {
		return "'" + s + "'"
	}
	if !strings.ContainsRune(s, '"') {
		return `"` + s + `"`
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\\', '\'', '"':
			b.WriteByte('\\')
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// stripLeadingQuote removes a leading ' or " from s (used when the user has
// already typed an opening quote before pressing Tab).
func stripLeadingQuote(s string) string {
	if len(s) > 0 && (s[0] == '\'' || s[0] == '"') {
		return s[1:]
	}
	return s
}

func filterPrefix(list []string, prefix string) []string {
	var out []string
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			out = append(out, s)
		}
	}
	return out
}
