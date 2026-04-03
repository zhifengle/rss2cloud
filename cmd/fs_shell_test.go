package cmd

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/zhifengle/rss2cloud/cloudfs"
)

// --- parseShellLine ---

func TestParseShellLine_Basic(t *testing.T) {
	tokens := parseShellLine("mv /anime/sub /")
	if len(tokens) != 3 || tokens[0] != "mv" || tokens[1] != "/anime/sub" || tokens[2] != "/" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestParseShellLine_SingleQuotes(t *testing.T) {
	tokens := parseShellLine("rename '/anime/my show' 'new show'")
	if len(tokens) != 3 || tokens[1] != "/anime/my show" || tokens[2] != "new show" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestParseShellLine_DoubleQuotes(t *testing.T) {
	tokens := parseShellLine(`rename "/anime/my show" "new show"`)
	if len(tokens) != 3 || tokens[1] != "/anime/my show" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestParseShellLine_Empty(t *testing.T) {
	if tokens := parseShellLine("   "); len(tokens) != 0 {
		t.Fatalf("expected empty, got %v", tokens)
	}
}

// --- shell loop dispatch ---

func runShellCmd(t *testing.T, line string) string {
	t.Helper()
	ctx := context.Background()
	session := newTestSession(t)
	history := &shellHistory{}
	var buf bytes.Buffer
	dispatchShellCommand(ctx, session, history, &buf, line)
	return buf.String()
}

func TestShellPwd(t *testing.T) {
	out := runShellCmd(t, "pwd")
	if strings.TrimSpace(out) != "/" {
		t.Fatalf("expected /, got %q", out)
	}
}

func TestShellHelp(t *testing.T) {
	out := runShellCmd(t, "help")
	if !strings.Contains(out, "pwd") || !strings.Contains(out, "ls") {
		t.Fatalf("help output missing commands: %q", out)
	}
}

func TestShellUnknownCommand(t *testing.T) {
	out := runShellCmd(t, "frobnicate")
	if !strings.Contains(out, "unknown command") {
		t.Fatalf("expected unknown command message, got %q", out)
	}
}

func TestShellCdChangesPwd(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	history := &shellHistory{}
	var buf bytes.Buffer

	dispatchShellCommand(ctx, session, history, &buf, "cd /anime")
	if session.Pwd() != "/anime" {
		t.Fatalf("expected /anime after cd, got %q", session.Pwd())
	}
}

func TestShellLs(t *testing.T) {
	out := runShellCmd(t, "ls /")
	if !strings.Contains(out, "anime") {
		t.Fatalf("expected 'anime' in ls output, got %q", out)
	}
}

func TestShellMkdir(t *testing.T) {
	out := runShellCmd(t, "mkdir /anime/newdir")
	if !strings.Contains(out, "newdir") {
		t.Fatalf("expected 'newdir' in mkdir output, got %q", out)
	}
}

func TestShellRename(t *testing.T) {
	out := runShellCmd(t, "rename /anime shows")
	if !strings.Contains(out, "shows") {
		t.Fatalf("expected 'shows' in rename output, got %q", out)
	}
}

func TestShellExitReturnsTrue(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	history := &shellHistory{}
	var buf bytes.Buffer
	done := dispatchShellCommand(ctx, session, history, &buf, "exit")
	if !done {
		t.Fatal("expected exit to return true")
	}
	done = dispatchShellCommand(ctx, session, history, &buf, "quit")
	if !done {
		t.Fatal("expected quit to return true")
	}
}

func TestShellMvOutput(t *testing.T) {
	out := runShellCmd(t, "mv /notes.txt /anime")
	if !strings.Contains(out, "moved") {
		t.Fatalf("expected 'moved' in mv output, got %q", out)
	}
}

func TestShellCpOutput(t *testing.T) {
	out := runShellCmd(t, "cp /notes.txt /anime")
	if !strings.Contains(out, "copied") {
		t.Fatalf("expected 'copied' in cp output, got %q", out)
	}
}

func TestShellSearchMvOutput(t *testing.T) {
	out := runShellCmd(t, "search-mv /anime episode /")
	if !strings.Contains(out, "moved") {
		t.Fatalf("expected 'moved' in search-mv output, got %q", out)
	}
}

func TestShellFlattenShowsUnsupported(t *testing.T) {
	out := runShellCmd(t, "flatten /anime")
	if !strings.Contains(out, "unsupported") {
		t.Fatalf("expected unsupported error in flatten output, got %q", out)
	}
}

func TestShellRm(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	history := &shellHistory{}
	var buf bytes.Buffer
	dispatchShellCommand(ctx, session, history, &buf, "rm /notes.txt")
	// After rm, stat should fail.
	dispatchShellCommand(ctx, session, history, &buf, "stat /notes.txt")
	if !strings.Contains(buf.String(), "error") {
		t.Fatalf("expected error after rm, got %q", buf.String())
	}
}

// --- history ---

func TestShellHistoryAdd(t *testing.T) {
	h := &shellHistory{}
	h.add("ls /")
	h.add("cd /anime")
	h.add("cd /anime") // duplicate — should be skipped
	if len(h.all()) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(h.all()))
	}
}

func TestShellHistorySaveLoad(t *testing.T) {
	f, err := os.CreateTemp("", "hist_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	h := newShellHistory(f.Name())
	h.add("ls /")
	h.add("cd /anime")
	if err := h.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	h2 := newShellHistory(f.Name())
	if len(h2.all()) != 2 || h2.all()[0] != "ls /" {
		t.Fatalf("unexpected loaded history: %v", h2.all())
	}
}

// --- state persistence ---

func TestShellStateSaveLoad(t *testing.T) {
	f, err := os.CreateTemp("", "state_*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if err := saveShellState(f.Name(), shellPersistedState{LastCwd: "/anime"}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	s := loadShellState(f.Name())
	if s.LastCwd != "/anime" {
		t.Fatalf("expected /anime, got %q", s.LastCwd)
	}
}

func TestShellStateLoadMissing(t *testing.T) {
	s := loadShellState("/nonexistent/path/state.json")
	if s.LastCwd != "" {
		t.Fatalf("expected empty state, got %+v", s)
	}
}

// --- completion ---

func TestCompleteCommandNames(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	candidates := completeInput(ctx, session, "l")
	found := false
	for _, c := range candidates {
		if c == "ls" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'ls' in completions, got %v", candidates)
	}
}

func TestCompletePathAfterCommand(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	// "ls " — trailing space means we're completing a path argument
	candidates := completeInput(ctx, session, "ls ")
	found := false
	for _, c := range candidates {
		if strings.Contains(c, "anime") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'anime' in path completions, got %v", candidates)
	}
}

// --- argument validation ---

func TestShellPwdRejectsExtraArgs(t *testing.T) {
	out := runShellCmd(t, "pwd extra")
	if !strings.Contains(out, "usage") {
		t.Fatalf("expected usage error for 'pwd extra', got %q", out)
	}
}

func TestShellCdRejectsZeroArgs(t *testing.T) {
	out := runShellCmd(t, "cd")
	if !strings.Contains(out, "usage") {
		t.Fatalf("expected usage error for 'cd', got %q", out)
	}
}

func TestShellCdRejectsExtraArgs(t *testing.T) {
	out := runShellCmd(t, "cd /anime /extra")
	if !strings.Contains(out, "usage") {
		t.Fatalf("expected usage error for 'cd a b', got %q", out)
	}
}

func TestShellStatRejectsExtraArgs(t *testing.T) {
	out := runShellCmd(t, "stat /anime /extra")
	if !strings.Contains(out, "usage") {
		t.Fatalf("expected usage error for 'stat a b', got %q", out)
	}
}

func TestShellMkdirRejectsExtraArgs(t *testing.T) {
	out := runShellCmd(t, "mkdir /anime/a /anime/b")
	if !strings.Contains(out, "usage") {
		t.Fatalf("expected usage error for 'mkdir a b', got %q", out)
	}
}

func TestShellRenameRejectsExtraArgs(t *testing.T) {
	out := runShellCmd(t, "rename /anime shows extra")
	if !strings.Contains(out, "usage") {
		t.Fatalf("expected usage error for 'rename a b c', got %q", out)
	}
}

func TestShellRenameRejectsTooFewArgs(t *testing.T) {
	out := runShellCmd(t, "rename /anime")
	if !strings.Contains(out, "usage") {
		t.Fatalf("expected usage error for 'rename a', got %q", out)
	}
}

// --- history command and !N recall ---

func TestShellHistoryCommand(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	h := &shellHistory{}
	h.add("ls /")
	h.add("cd /anime")
	var buf bytes.Buffer
	dispatchShellCommand(ctx, session, h, &buf, "history")
	out := buf.String()
	if !strings.Contains(out, "ls /") || !strings.Contains(out, "cd /anime") {
		t.Fatalf("expected history entries in output, got %q", out)
	}
}

func TestShellRecallHistory(t *testing.T) {
	h := &shellHistory{}
	h.add("ls /")
	h.add("cd /anime")
	var buf bytes.Buffer
	line, ok := recallHistory(h, "1", &buf)
	if !ok || line != "ls /" {
		t.Fatalf("expected 'ls /', got %q (ok=%v)", line, ok)
	}
}

func TestShellRecallHistoryOutOfRange(t *testing.T) {
	h := &shellHistory{}
	h.add("ls /")
	var buf bytes.Buffer
	_, ok := recallHistory(h, "99", &buf)
	if ok {
		t.Fatal("expected failure for out-of-range index")
	}
	if !strings.Contains(buf.String(), "error") {
		t.Fatalf("expected error message, got %q", buf.String())
	}
}

// --- Tab completion suffix ---

func TestCompleteSuffix(t *testing.T) {
	// "ls an" → candidate "anime/" → suffix "ime/"
	suffix := completeSuffix("ls an", "anime/")
	if suffix != "ime/" {
		t.Fatalf("expected 'ime/', got %q", suffix)
	}
}

func TestCompleteSuffixNoMatch(t *testing.T) {
	suffix := completeSuffix("ls xyz", "anime/")
	if suffix != "" {
		t.Fatalf("expected empty suffix, got %q", suffix)
	}
}

// --- readLineWithCompletion via pipe ---

func TestReadLineWithCompletion_PlainInput(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	input := "ls /\n"
	var out bytes.Buffer
	line, eof := readLineWithCompletion(ctx, session, strings.NewReader(input), &out, &lineReadState{})
	if eof {
		t.Fatal("unexpected EOF")
	}
	if line != "ls /" {
		t.Fatalf("expected 'ls /', got %q", line)
	}
}

func TestReadLineWithCompletion_EOF(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	var out bytes.Buffer
	_, eof := readLineWithCompletion(ctx, session, strings.NewReader(""), &out, &lineReadState{})
	if !eof {
		t.Fatal("expected EOF on empty reader")
	}
}

func TestReadLineWithCompletion_TabCompletion(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	// "ls an\t\n" — Tab after "an" should complete to "anime/"
	input := "ls an\t\n"
	var out bytes.Buffer
	line, eof := readLineWithCompletion(ctx, session, strings.NewReader(input), &out, &lineReadState{})
	if eof {
		t.Fatal("unexpected EOF")
	}
	// The line buffer should contain the completed text.
	if !strings.Contains(line, "anime") {
		t.Fatalf("expected 'anime' in completed line, got %q", line)
	}
}

// --- \r handling: no peek, no byte loss ---

func TestReadLineWithCompletion_CRonly(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	// Raw-mode terminal sends only \r on Enter.
	input := "pwd\r"
	var out bytes.Buffer
	line, eof := readLineWithCompletion(ctx, session, strings.NewReader(input), &out, &lineReadState{})
	if eof {
		t.Fatal("unexpected EOF for \\r-terminated line")
	}
	if line != "pwd" {
		t.Fatalf("expected 'pwd', got %q", line)
	}
}

func TestReadLineWithCompletion_CRLFDoesNotLoseNextByte(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	// Two lines: "pwd\r\nls\n". The \n after \r must not create an empty line
	// or swallow the next command's first byte.
	r := strings.NewReader("pwd\r\nls\n")
	var out bytes.Buffer
	state := &lineReadState{}
	line1, eof1 := readLineWithCompletion(ctx, session, r, &out, state)
	if eof1 || line1 != "pwd" {
		t.Fatalf("first line: got %q eof=%v", line1, eof1)
	}
	line2, eof2 := readLineWithCompletion(ctx, session, r, &out, state)
	if eof2 || line2 != "ls" {
		t.Fatalf("second line: got %q eof=%v", line2, eof2)
	}
}

// --- EOF with unterminated last line ---

func TestRunShellLoop_EOFExecutesLastLine(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	history := &shellHistory{}
	// "pwd" with no trailing newline — must still be executed.
	input := strings.NewReader("pwd")
	var out bytes.Buffer
	runShellLoop(ctx, session, history, input, &out)
	if !strings.Contains(out.String(), "/") {
		t.Fatalf("expected pwd output, got %q", out.String())
	}
}

// --- completion quoting ---

func TestCompletePathQuotesSpaces(t *testing.T) {
	ctx := context.Background()
	d := newCmdFakeDriver()
	// Add an entry with a space in the name.
	d.entries["9"] = cloudfs.Entry{ID: "9", ParentID: "0", Name: "my show", Type: cloudfs.EntryTypeDirectory}
	d.children["0"] = append(d.children["0"], "9")
	d.children["9"] = []string{}
	session, _ := cloudfs.NewSession(ctx, d)

	candidates := completePath(ctx, session, "my")
	found := false
	for _, c := range candidates {
		if c == "'my show/'" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected quoted candidate \"'my show/'\", got %v", candidates)
	}
}

func TestCompleteSuffix_QuotedCandidate(t *testing.T) {
	// User typed "cd my", candidate is "'my show/'" — suffix replaces token.
	suffix := completeSuffix("cd my", "'my show/'")
	// Should signal full replacement.
	if !strings.HasPrefix(suffix, "\x00") {
		t.Fatalf("expected replacement signal, got %q", suffix)
	}
}

func TestCompletePathEscapesQuotes(t *testing.T) {
	ctx := context.Background()
	d := newCmdFakeDriver()
	d.entries["9"] = cloudfs.Entry{
		ID:       "9",
		ParentID: "0",
		Name:     `Bob's "show"`,
		Type:     cloudfs.EntryTypeDirectory,
	}
	d.children["0"] = append(d.children["0"], "9")
	d.children["9"] = []string{}
	session, _ := cloudfs.NewSession(ctx, d)

	candidates := completePath(ctx, session, "Bob")
	want := `Bob\'s\ \"show\"/`
	found := false
	for _, c := range candidates {
		if c == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected escaped candidate %q, got %v", want, candidates)
	}
}

func TestParseShellLine_BackslashEscapes(t *testing.T) {
	tokens := parseShellLine(`cd Bob\'s\ \"show\"/`)
	if len(tokens) != 2 || tokens[1] != `Bob's "show"/` {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestCompleteSuffix_UnquotedCandidate(t *testing.T) {
	suffix := completeSuffix("ls an", "anime/")
	if suffix != "ime/" {
		t.Fatalf("expected 'ime/', got %q", suffix)
	}
}
