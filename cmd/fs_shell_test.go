package cmd

import (
	"bytes"
	"context"
	"errors"
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

func TestParseShellLine_BackslashEscapes(t *testing.T) {
	tokens := parseShellLine(`cd Bob\'s\ \"show\"/`)
	if len(tokens) != 2 || tokens[1] != `Bob's "show"/` {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

// --- shell dispatch ---

func runShellCmd(t *testing.T, line string) string {
	t.Helper()
	ctx := context.Background()
	session := newTestSession(t)
	var buf bytes.Buffer
	dispatchShellCommand(ctx, session, &buf, line)
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
	if !strings.Contains(out, "pwd") || !strings.Contains(out, "ls") || !strings.Contains(out, "refresh") {
		t.Fatalf("help output missing commands: %q", out)
	}
}

func TestHelpRemovesLegacyHistoryCommands(t *testing.T) {
	out := runShellCmd(t, "help")
	if strings.Contains(out, "\n  history") || strings.Contains(out, "\n  !N") {
		t.Fatalf("expected help without legacy history commands, got %q", out)
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
	var buf bytes.Buffer

	dispatchShellCommand(ctx, session, &buf, "cd /anime")
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
	var buf bytes.Buffer
	done := dispatchShellCommand(ctx, session, &buf, "exit")
	if !done {
		t.Fatal("expected exit to return true")
	}
	done = dispatchShellCommand(ctx, session, &buf, "quit")
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

func TestShellSearchMvUnderscoreAliasOutput(t *testing.T) {
	out := runShellCmd(t, "search_mv /anime episode /")
	if !strings.Contains(out, "moved") {
		t.Fatalf("expected 'moved' in search_mv output, got %q", out)
	}
}

func TestShellFlattenOutput(t *testing.T) {
	out := runShellCmd(t, "flatten /anime")
	if !strings.Contains(out, "flattened") {
		t.Fatalf("expected flatten output, got %q", out)
	}
}

func TestShellRm(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	var buf bytes.Buffer
	dispatchShellCommand(ctx, session, &buf, "rm /notes.txt")
	dispatchShellCommand(ctx, session, &buf, "stat /notes.txt")
	if !strings.Contains(buf.String(), "error") {
		t.Fatalf("expected error after rm, got %q", buf.String())
	}
}

func TestShellRefreshClearsSessionCache(t *testing.T) {
	ctx := context.Background()
	d := newCountingCmdFakeDriver()
	session, err := cloudfs.NewSession(ctx, d)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	var buf bytes.Buffer

	if _, err := session.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) failed: %v", err)
	}
	dispatchShellCommand(ctx, session, &buf, "refresh")
	if _, err := session.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) after refresh failed: %v", err)
	}

	if got := d.listCalls["0"]; got != 2 {
		t.Fatalf("expected refresh to clear root cache, got %d calls", got)
	}
	if !strings.Contains(buf.String(), "cache cleared") {
		t.Fatalf("expected refresh output, got %q", buf.String())
	}
}

func TestShellRefreshRejectsArgs(t *testing.T) {
	out := runShellCmd(t, "refresh now")
	if !strings.Contains(out, "usage") {
		t.Fatalf("expected usage error for 'refresh now', got %q", out)
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

func TestCompleteCommandNamesIncludesRefresh(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
	candidates := completeInput(ctx, session, "ref")
	found := false
	for _, c := range candidates {
		if c == "refresh" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'refresh' in completions, got %v", candidates)
	}
}

func TestCompletePathAfterCommand(t *testing.T) {
	ctx := context.Background()
	session := newTestSession(t)
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

func TestCompletePathQuotesSpaces(t *testing.T) {
	ctx := context.Background()
	d := newCmdFakeDriver()
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

func TestCompletePathUsesCachedDirectoryListing(t *testing.T) {
	ctx := context.Background()
	d := newCountingCmdFakeDriver()
	session, err := cloudfs.NewSession(ctx, d)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	first := completePath(ctx, session, "")
	second := completePath(ctx, session, "")
	if len(first) == 0 || len(second) == 0 {
		t.Fatalf("expected non-empty completions, got %v and %v", first, second)
	}
	if got := d.listCalls["0"]; got != 1 {
		t.Fatalf("expected root List to be called once, got %d", got)
	}
}

func TestCompletePathReloadsAfterMkdir(t *testing.T) {
	ctx := context.Background()
	d := newCountingCmdFakeDriver()
	session, err := cloudfs.NewSession(ctx, d)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	_ = completePath(ctx, session, "")
	if _, err := session.Mkdir(ctx, "/fresh"); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	candidates := completePath(ctx, session, "fr")
	if got := d.listCalls["0"]; got != 2 {
		t.Fatalf("expected root List to be called twice after cache invalidation, got %d", got)
	}
	found := false
	for _, c := range candidates {
		if c == "fresh/" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected fresh/ in completions, got %v", candidates)
	}
}

func TestReadlineCompletionPrefix(t *testing.T) {
	if got := readlineCompletionPrefix("ls an"); got != "an" {
		t.Fatalf("expected prefix an, got %q", got)
	}
}

func TestReadlineCompletionCandidates(t *testing.T) {
	comps := readlineCompletionCandidates([]string{"ls", "list"}, "commands")
	if len(comps) != 2 {
		t.Fatalf("expected 2 completions, got %d", len(comps))
	}
	if comps[0].Value != "ls" || comps[0].Display != "ls" || comps[0].Tag != "commands" {
		t.Fatalf("unexpected completion: %+v", comps[0])
	}
}

func TestShellReadlineCompletionsQuotedPathCandidate(t *testing.T) {
	ctx := context.Background()
	d := newCmdFakeDriver()
	d.entries["9"] = cloudfs.Entry{ID: "9", ParentID: "0", Name: "my show", Type: cloudfs.EntryTypeDirectory}
	d.children["0"] = append(d.children["0"], "9")
	d.children["9"] = []string{}
	session, _ := cloudfs.NewSession(ctx, d)

	comps := shellReadlineCompletions(ctx, session, "cd my", len("cd my"))
	if comps.PREFIX != "my" {
		t.Fatalf("expected prefix my, got %q", comps.PREFIX)
	}

	raw := readlineCompletionCandidates(completeInput(ctx, session, "cd my"), "paths")
	if len(raw) != 1 || raw[0].Value != "'my show/'" {
		t.Fatalf("expected quoted readline candidate, got %+v", raw)
	}
}

func TestShellReadlineCompletionsEscapedQuoteCandidate(t *testing.T) {
	ctx := context.Background()
	d := newCmdFakeDriver()
	d.entries["9"] = cloudfs.Entry{ID: "9", ParentID: "0", Name: `Bob's "show"`, Type: cloudfs.EntryTypeDirectory}
	d.children["0"] = append(d.children["0"], "9")
	d.children["9"] = []string{}
	session, _ := cloudfs.NewSession(ctx, d)

	raw := readlineCompletionCandidates(completeInput(ctx, session, "cd Bob"), "paths")
	want := `Bob\'s\ \"show\"/`
	if len(raw) != 1 || raw[0].Value != want {
		t.Fatalf("expected escaped readline candidate %q, got %+v", want, raw)
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

// --- shell runtime helpers ---

func TestRequireInteractiveTerminalRejectsRegularFile(t *testing.T) {
	f, err := os.CreateTemp("", "not-tty")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	in, err := os.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	err = requireInteractiveTerminal(in)
	if !errors.Is(err, errNonInteractiveShell) {
		t.Fatalf("expected non-interactive shell error, got %v", err)
	}
}

func TestFsShellRunRejectsNonInteractiveBeforeInitSession(t *testing.T) {
	oldInit := initShellSession
	defer func() {
		initShellSession = oldInit
	}()

	called := false
	initShellSession = func(ctx context.Context) *cloudfs.Session {
		called = true
		return nil
	}

	f, err := os.CreateTemp("", "not-tty")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	in, err := os.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	oldStdin := os.Stdin
	oldCwd := fsCwd
	defer func() {
		os.Stdin = oldStdin
		fsCwd = oldCwd
	}()
	os.Stdin = in
	fsCwd = ""

	fsShellCmd.Run(fsShellCmd, nil)
	if called {
		t.Fatal("expected non-interactive shell to return before session init")
	}
}

func TestInitShellHistoryRebuildsLegacyFile(t *testing.T) {
	f, err := os.CreateTemp("", "legacy-history-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString("ls /\ncd /anime\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	history, err := initShellHistory(f.Name())
	if err != nil {
		t.Fatalf("init history failed: %v", err)
	}
	if history == nil {
		t.Fatal("expected readline history source")
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("expected legacy history to be truncated, got %q", string(data))
	}
}
