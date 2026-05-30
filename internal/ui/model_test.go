package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/dpuwork/differ/internal/config"
	"github.com/dpuwork/differ/internal/git"
	"github.com/dpuwork/differ/internal/theme"
)

func TestBuildFileItems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		changes   []git.FileChange
		untracked []string
		wantLen   int
	}{
		{"empty", nil, nil, 0},
		{"changes_only", []git.FileChange{{Path: "a.go", Status: git.StatusModified}}, nil, 1},
		{"untracked_only", nil, []string{"b.go"}, 1},
		{"mixed", []git.FileChange{{Path: "a.go", Status: git.StatusModified}}, []string{"b.go"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildFileItems(nil, tt.changes, tt.untracked)
			if len(got) != tt.wantLen {
				t.Errorf("len=%d, want %d", len(got), tt.wantLen)
			}
			// Verify untracked items have the flag set
			for _, f := range got {
				if f.change.Status == git.StatusUntracked && !f.untracked {
					t.Error("untracked item should have untracked=true")
				}
			}
		})
	}
}

func TestBuildFileItems_Sorting(t *testing.T) {
	t.Parallel()
	changes := []git.FileChange{
		{Path: "unstaged.go", Staged: false},
		{Path: "staged.go", Staged: true},
	}
	untracked := []string{"untracked.go"}

	got := buildFileItems(nil, changes, untracked)

	if len(got) != 3 {
		t.Fatalf("len=%d, want 3", len(got))
	}

	if !got[0].change.Staged {
		t.Errorf("first item should be staged, got path=%s", got[0].change.Path)
	}
	if got[0].change.Path != "staged.go" {
		t.Errorf("got[0]=%s, want staged.go", got[0].change.Path)
	}
}

func TestTruncatePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		maxW int
		want string // empty means "just check length"
	}{
		{"short", "file.go", 20, "file.go"},
		{"exact", "file.go", 7, "file.go"},
		{"long", "very-long-filename-that-exceeds-limit.go", 10, ""},
		{"single_char", "x", 1, "x"},
		{"boundary", "abc", 3, "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := TruncatePath(tt.path, tt.maxW)
			if tt.want != "" && got != tt.want {
				t.Errorf("TruncatePath(%q, %d) = %q, want %q", tt.path, tt.maxW, got, tt.want)
			}
			if tt.want == "" {
				// For truncated paths, just verify it starts with ellipsis
				if !strings.HasPrefix(got, "…") {
					t.Errorf("expected truncated path to start with …, got %q", got)
				}
			}
		})
	}
}

func TestFilesEqual_Equal(t *testing.T) {
	t.Parallel()
	a := []fileItem{{change: git.FileChange{Path: "a.go", Status: git.StatusModified}}}
	b := []fileItem{{change: git.FileChange{Path: "a.go", Status: git.StatusModified}}}
	if !filesEqual(a, b) {
		t.Error("expected equal")
	}
}

func TestFilesEqual_DiffLength(t *testing.T) {
	t.Parallel()
	a := []fileItem{{change: git.FileChange{Path: "a.go"}}}
	b := []fileItem{{change: git.FileChange{Path: "a.go"}}, {change: git.FileChange{Path: "b.go"}}}
	if filesEqual(a, b) {
		t.Error("different lengths should not be equal")
	}
}

func TestFilesEqual_DiffContent(t *testing.T) {
	t.Parallel()
	a := []fileItem{{change: git.FileChange{Path: "a.go", Status: git.StatusModified}}}
	b := []fileItem{{change: git.FileChange{Path: "a.go", Status: git.StatusAdded}}}
	if filesEqual(a, b) {
		t.Error("different status should not be equal")
	}
}

func TestFilesEqual_BothEmpty(t *testing.T) {
	t.Parallel()
	if !filesEqual(nil, nil) {
		t.Error("two nil slices should be equal")
	}
}

func TestFilesEqual_OneEmpty(t *testing.T) {
	t.Parallel()
	a := []fileItem{{change: git.FileChange{Path: "a.go"}}}
	if filesEqual(a, nil) {
		t.Error("non-empty vs nil should not be equal")
	}
}

func TestContentHeight(t *testing.T) {
	t.Parallel()
	m := Model{height: 30}
	// height - 3: cards add top+bottom border (+2), status bar (1)
	if got := m.contentHeight(); got != 27 {
		t.Errorf("contentHeight()=%d, want 27", got)
	}
}

func TestDiffWidth(t *testing.T) {
	t.Parallel()
	m := Model{width: 120}
	// width - fileListWidth(35) - 2(file card borders) - 1(gap) - 2(diff card borders)
	want := 120 - 40
	if got := m.diffWidth(); got != want {
		t.Errorf("diffWidth()=%d, want %d", got, want)
	}
}

func TestRenderCard_Dimensions(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	content := "line1\nline2\nline3"
	card := m.renderCard("Title", content, true, 20, 5)
	lines := strings.Split(card, "\n")
	// h=5 content lines + 2 border lines (top + bottom) = 7
	if len(lines) != 7 {
		t.Errorf("card line count=%d, want 7", len(lines))
	}
}

func TestRenderCard_Title(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	card := m.renderCard("MyTitle", "content", false, 20, 3)
	firstLine := strings.Split(card, "\n")[0]
	if !strings.Contains(firstLine, "MyTitle") {
		t.Errorf("first line should contain title, got %q", firstLine)
	}
}

func TestRenderCard_BorderChars(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	unfocused := m.renderCard("T", "x", false, 10, 2)
	for _, ch := range []string{"╭", "╮", "╰", "╯", "│"} {
		if !strings.Contains(unfocused, ch) {
			t.Errorf("unfocused card missing border char %q", ch)
		}
	}
	focused := m.renderCard("T", "x", true, 10, 2)
	for _, ch := range []string{"╭", "╮", "╰", "╯", "│"} {
		if !strings.Contains(focused, ch) {
			t.Errorf("focused card missing border char %q", ch)
		}
	}
}

func TestRenderCard_FocusedVsUnfocused(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	// Both should render without panic and contain border chars
	focused := m.renderCard("T", "x", true, 10, 2)
	unfocused := m.renderCard("T", "x", false, 10, 2)

	if !strings.Contains(focused, "╭") {
		t.Error("focused card should contain single border chars")
	}
	if !strings.Contains(unfocused, "╭") {
		t.Error("unfocused card should contain single border chars")
	}
}

func TestRenderCard_EmptyTitle(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	card := m.renderCard("", "content", false, 15, 2)
	firstLine := strings.Split(card, "\n")[0]
	if !strings.Contains(firstLine, "╭") || !strings.Contains(firstLine, "╮") {
		t.Error("card with empty title should still have border corners")
	}
}

func newTestModel(t *testing.T, files []fileItem) Model {
	t.Helper()
	th := theme.GetTheme(false)
	bf := textinput.New()
	bf.Placeholder = "filter..."
	bf.CharLimit = 100
	bf.SetWidth(fileListWidth - 8)
	bi := textinput.New()
	bi.Placeholder = "branch name..."
	bi.CharLimit = 100
	return Model{
		files:        files,
		styles:       NewStyles(th),
		theme:        th,
		cfg:          config.Default(),
		wrapDiff:     config.Default().WrapDiff,
		width:        120,
		height:       30,
		commitInput:  textinput.New(),
		branchFilter: bf,
		branchInput:  bi,
	}
}

func TestRenderFileList_Headers(t *testing.T) {
	t.Parallel()
	files := []fileItem{
		{change: git.FileChange{Path: "staged.go", Staged: true}},
		{change: git.FileChange{Path: "unstaged.go", Staged: false}},
	}
	m := newTestModel(t, files)
	out := m.renderFileList(10)

	if !strings.Contains(out, "STAGED") {
		t.Error("file list should contain STAGED header")
	}
	if !strings.Contains(out, "CHANGES") {
		t.Error("file list should contain CHANGES header")
	}
}

func TestRenderStatusBar_StagedCount(t *testing.T) {
	t.Parallel()
	files := []fileItem{
		{change: git.FileChange{Path: "a.go", Staged: true}},
		{change: git.FileChange{Path: "b.go", Staged: false}},
	}
	m := newTestModel(t, files)
	bar := m.renderStatusBar()
	if !strings.Contains(bar, "1 staged") {
		t.Errorf("status bar should show staged count, got %q", bar)
	}
}

func TestRenderStatusBar_SplitIndicator(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.splitDiff = true
	bar := m.renderStatusBar()
	if !strings.Contains(bar, "split") {
		t.Error("status bar should show split indicator when splitDiff=true")
	}
}

func TestRenderStatusBar_WrapIndicator(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.wrapDiff = true
	bar := m.renderStatusBar()
	if !strings.Contains(bar, "wrap") {
		t.Error("status bar should show wrap indicator when wrapDiff=true")
	}
}

func TestRenderStatusBar_SplitAndWrapIndicator(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.splitDiff = true
	m.wrapDiff = true
	bar := m.renderStatusBar()
	if !strings.Contains(bar, "split wrap") {
		t.Error("status bar should show split and wrap together without a divider")
	}
	if strings.Contains(bar, "split │ wrap") || strings.Contains(bar, "split│wrap") {
		t.Error("modes should not be separated by divider")
	}
}

func TestRenderStatusBar_UpstreamStatus(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.upstream.Upstream = "origin/main"
	m.upstream.Ahead = 0
	m.upstream.Behind = 0
	bar := m.renderStatusBar()
	if !strings.Contains(bar, "↑0 ↓0") {
		t.Error("status bar should always show upstream status when configured")
	}
}

func TestRenderStatusBar_StatusMsg(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.statusMsg = "committed!"
	bar := m.renderStatusBar()
	if !strings.Contains(bar, "committed!") {
		t.Error("status bar should show status message")
	}

	m.statusMsg = "pull failed: error: cannot pull with rebase: You have unstaged changes.\nerror: additionally, your index contains uncommitted changes."
	barMultiline := m.renderStatusBar()
	if strings.Contains(barMultiline, "additionally") {
		t.Error("status bar should truncate multiline status messages")
	}
	if !strings.Contains(barMultiline, "pull failed: error: cannot pull with rebase: You have unstaged changes.") {
		t.Error("status bar should show first line of multiline status message")
	}
}

func TestUpdateFileListMode_ToggleWrap(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.wrapDiff = false

	// Press 'w'
	result, _ := m.updateFileListMode(tea.KeyPressMsg{Text: "w"})
	rm := result.(Model)

	if !rm.wrapDiff {
		t.Error("wrapDiff should be true after pressing 'w'")
	}
	if rm.prevCurs != -1 {
		t.Error("prevCurs should be reset to -1")
	}
	if rm.lastDiffContent != "" {
		t.Error("lastDiffContent should be empty to force re-render")
	}

	// Press 'w' again
	result2, _ := rm.updateFileListMode(tea.KeyPressMsg{Text: "w"})
	rm2 := result2.(Model)

	if rm2.wrapDiff {
		t.Error("wrapDiff should be false after pressing 'w' again")
	}
}

func TestUpdateDiffMode_ToggleWrap(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeDiff
	m.wrapDiff = false

	// Press 'w'
	result, _ := m.updateDiffMode(tea.KeyPressMsg{Text: "w"})
	rm := result.(Model)

	if !rm.wrapDiff {
		t.Error("wrapDiff should be true after pressing 'w'")
	}
	if rm.prevCurs != -1 {
		t.Error("prevCurs should be reset to -1")
	}
	if rm.lastDiffContent != "" {
		t.Error("lastDiffContent should be empty to force re-render")
	}
}

func TestRenderHelpPopup(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeHelp
	m.version = "v1.2.3-test"
	out := m.renderHelpPopup()
	for _, key := range []string{"j/k", "enter", "tab", "q", "s", "w", "b", "c", "P", "F"} {
		if !strings.Contains(out, key) {
			t.Errorf("help popup should contain %q", key)
		}
	}
	if !strings.Contains(out, "v1.2.3-test") {
		t.Error("help popup should contain version string in the title")
	}
}

func TestRenderBranchList(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.branches = []string{"main", "feature-a", "feature-b"}
	m.currentBranch = "main"
	m.branchCursor = 0
	out := m.renderBranchList(10)
	if !strings.Contains(out, "main") {
		t.Error("branch list should contain main")
	}
	if !strings.Contains(out, "feature-a") {
		t.Error("branch list should contain feature-a")
	}
	if !strings.Contains(out, "*") {
		t.Error("branch list should mark current branch with *")
	}
}

func TestRenderBranchItem_ContainsName(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	item := m.renderBranchItem("feature-branch", true, false)
	if !strings.Contains(item, "feature-branch") {
		t.Error("branch item should contain branch name")
	}
}

func TestRenderBranchItem_Current(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	item := m.renderBranchItem("main", false, true)
	if !strings.Contains(item, "*") {
		t.Error("current branch should have * prefix")
	}
}

func TestRenderFileItem_ShowsStats(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	item := fileItem{change: git.FileChange{Path: "main.go", Status: git.StatusModified, AddedLines: 12, DeletedLines: 3}}
	out := m.renderFileItem(item, false)
	if !strings.Contains(out, "+12") || !strings.Contains(out, "-3") {
		t.Errorf("expected stats in file item, got %q", out)
	}
}

func TestUpdateBranchMode_Navigation(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main", "dev", "feature"}
	m.branchCursor = 0

	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "down"})
	rm := result.(Model)
	if rm.branchCursor != 1 {
		t.Errorf("cursor=%d after down, want 1", rm.branchCursor)
	}

	result, _ = rm.updateBranchMode(tea.KeyPressMsg{Text: "up"})
	rm = result.(Model)
	if rm.branchCursor != 0 {
		t.Errorf("cursor=%d after up, want 0", rm.branchCursor)
	}
}

func TestUpdateBranchMode_Esc(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main"}
	m.branchCursor = 0

	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "esc"})
	rm := result.(Model)
	if rm.mode != modeFileList {
		t.Errorf("mode=%d after esc, want modeFileList", rm.mode)
	}
}

func TestHandleBranchesLoaded_Error(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	msg := branchesLoadedMsg{err: fmt.Errorf("permission denied")}
	result, _ := m.handleBranchesLoaded(msg)
	rm := result.(Model)
	if rm.mode != modeFileList {
		t.Error("should stay in file list mode on error")
	}
	if !strings.Contains(rm.statusMsg, "permission denied") {
		t.Errorf("statusMsg=%q, want error message", rm.statusMsg)
	}
}

func TestHandleResize_ClearsDiffCache(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, []fileItem{
		{change: git.FileChange{Path: "a.go", Status: git.StatusModified}},
	})
	m.cursor = 0

	// Simulate having cached diff content
	m.lastDiffContent = "old diff"
	m.viewport.SetContent("old diff")

	// Resize creates new viewport — cache must be cleared
	result, _ := m.handleResize(tea.WindowSizeMsg{Width: 100, Height: 40})
	rm := result.(Model)

	if rm.lastDiffContent != "" {
		t.Error("handleResize should clear lastDiffContent to force re-apply")
	}

	// handleDiffLoaded with same content should apply (not skip) after resize
	result2, _ := rm.handleDiffLoaded(diffLoadedMsg{content: "old diff", index: 0})
	rm2 := result2.(Model)
	if rm2.lastDiffContent != "old diff" {
		t.Error("handleDiffLoaded should apply content after resize cleared cache")
	}
	if !strings.Contains(rm2.viewport.View(), "old diff") {
		t.Errorf("viewport should contain reapplied content, got %q", rm2.viewport.View())
	}
}

func TestHandleDiffLoaded_SkipsDuplicate(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, []fileItem{
		{change: git.FileChange{Path: "a.go", Status: git.StatusModified}},
	})
	m.cursor = 0
	m.lastDiffContent = "same diff"

	// Same content as cache — should be a no-op
	result, _ := m.handleDiffLoaded(diffLoadedMsg{content: "same diff", index: 0})
	rm := result.(Model)
	if rm.lastDiffContent != "same diff" {
		t.Error("cache should remain unchanged on duplicate")
	}
}

func TestBranchListScroll(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	// height=30, contentHeight=26, itemH=25 (minus filter bar).
	branches := make([]string, 40)
	for i := range branches {
		branches[i] = fmt.Sprintf("branch-%02d", i)
	}
	m.branches = branches
	m.branchCursor = 35
	m.branchOffset = 35 - 25 + 1 // 11

	out := m.renderBranchList(m.contentHeight())
	if !strings.Contains(out, "branch-35") {
		t.Error("branch list should show cursor branch when scrolled")
	}
	if strings.Contains(out, "branch-00") {
		t.Error("branch list should not show first branch when scrolled down")
	}
}

func TestFilterBranches(t *testing.T) {
	t.Parallel()
	branches := []string{"main", "feature-auth", "feature-ui", "bugfix-login", "dev"}

	t.Run("empty query returns nil", func(t *testing.T) {
		t.Parallel()
		if got := filterBranches(branches, ""); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("substring match", func(t *testing.T) {
		t.Parallel()
		got := filterBranches(branches, "feature")
		if len(got) != 2 {
			t.Fatalf("expected 2 matches, got %d: %v", len(got), got)
		}
	})
	t.Run("case insensitive", func(t *testing.T) {
		t.Parallel()
		got := filterBranches(branches, "FEATURE")
		if len(got) != 2 {
			t.Fatalf("expected 2 matches, got %d: %v", len(got), got)
		}
	})
	t.Run("no match", func(t *testing.T) {
		t.Parallel()
		got := filterBranches(branches, "zzz")
		if len(got) != 0 {
			t.Fatalf("expected 0 matches, got %d: %v", len(got), got)
		}
	})
}

func TestUpdateBranchMode_TypeFilters(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main", "feature-auth", "feature-ui", "dev"}
	m.branchFilter.Focus()

	// Type 'f' — should filter to feature branches
	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "f"})
	rm := result.(Model)
	if rm.filteredBranches == nil {
		t.Fatal("filteredBranches should not be nil after typing")
	}
	if len(rm.filteredBranches) != 2 {
		t.Errorf("expected 2 filtered branches, got %d", len(rm.filteredBranches))
	}
	if rm.branchCursor != 0 {
		t.Errorf("cursor should reset to 0, got %d", rm.branchCursor)
	}
}

func TestUpdateBranchMode_EscClearsFilter(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main", "feature-auth", "dev"}
	m.branchFilter.Focus()
	m.branchFilter.SetValue("feat")
	m.filteredBranches = filterBranches(m.branches, "feat")

	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "esc"})
	rm := result.(Model)
	// First esc clears filter, stays in branch picker
	if rm.mode != modeBranchPicker {
		t.Errorf("mode=%d, want modeBranchPicker", rm.mode)
	}
	if rm.branchFilter.Value() != "" {
		t.Errorf("filter should be cleared, got %q", rm.branchFilter.Value())
	}
	if rm.filteredBranches != nil {
		t.Error("filteredBranches should be nil after clearing")
	}
}

func TestUpdateBranchMode_EscClosesWhenEmpty(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main"}
	m.branchFilter.Focus()
	// Filter is empty — esc should close
	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "esc"})
	rm := result.(Model)
	if rm.mode != modeFileList {
		t.Errorf("mode=%d, want modeFileList", rm.mode)
	}
}

func TestUpdateBranchMode_ArrowsInFilteredList(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main", "feature-auth", "feature-ui", "dev"}
	m.filteredBranches = []string{"feature-auth", "feature-ui"}
	m.branchCursor = 0

	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "down"})
	rm := result.(Model)
	if rm.branchCursor != 1 {
		t.Errorf("cursor=%d after down, want 1", rm.branchCursor)
	}
	// Should not go past end of filtered list
	result, _ = rm.updateBranchMode(tea.KeyPressMsg{Text: "down"})
	rm = result.(Model)
	if rm.branchCursor != 1 {
		t.Errorf("cursor=%d, should not exceed filtered list", rm.branchCursor)
	}
}

func TestUpdateBranchMode_CtrlJK(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main", "dev", "feature"}
	m.branchCursor = 0

	// ctrl+j moves down
	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "ctrl+j"})
	rm := result.(Model)
	if rm.branchCursor != 1 {
		t.Errorf("cursor=%d after ctrl+j, want 1", rm.branchCursor)
	}

	// ctrl+k moves up
	result, _ = rm.updateBranchMode(tea.KeyPressMsg{Text: "ctrl+k"})
	rm = result.(Model)
	if rm.branchCursor != 0 {
		t.Errorf("cursor=%d after ctrl+k, want 0", rm.branchCursor)
	}
}

func TestRenderBranchList_ShowsFilterBar(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main", "dev"}
	m.branchCursor = 0
	out := m.renderBranchList(10)
	// Should contain the match count
	if !strings.Contains(out, "2/2") {
		t.Error("branch list should show match count")
	}
}

func TestRenderBranchList_NoMatches(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main", "dev"}
	m.filteredBranches = []string{} // empty filter result
	m.branchFilter.SetValue("zzz")
	out := m.renderBranchList(10)
	if !strings.Contains(out, "no matches") {
		t.Error("should show 'no matches' placeholder")
	}
	if !strings.Contains(out, "0/2") {
		t.Error("should show 0/2 count")
	}
}

func TestUpdateBranchMode_CtrlN_EntersCreateMode(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branches = []string{"main", "dev"}
	m.branchCursor = 0
	m.branchFilter.Focus()

	result, cmd := m.updateBranchMode(tea.KeyPressMsg{Text: "ctrl+n"})
	rm := result.(Model)
	if !rm.branchCreating {
		t.Error("ctrl+n should set branchCreating=true")
	}
	if rm.mode != modeBranchPicker {
		t.Error("should stay in branch picker mode")
	}
	if cmd == nil {
		t.Error("expected textinput.Blink cmd")
	}
}

func TestUpdateBranchMode_CreateMode_RoutesToInput(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branchCreating = true
	m.branchInput.Focus()
	m.branches = []string{"main"}

	// Typing 'j' should go to text input, not move branch cursor
	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "j"})
	rm := result.(Model)
	if rm.branchInput.Value() != "j" {
		t.Errorf("input=%q, want %q", rm.branchInput.Value(), "j")
	}
}

func TestUpdateBranchMode_CreateMode_Esc_Cancels(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branchCreating = true
	m.branchInput.Focus()
	m.branchInput.SetValue("feature-x")
	m.branches = []string{"main"}

	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "esc"})
	rm := result.(Model)
	if rm.branchCreating {
		t.Error("esc should cancel branch creation")
	}
	if rm.branchInput.Value() != "" {
		t.Error("input should be reset on cancel")
	}
}

func TestUpdateBranchMode_CreateMode_CtrlC_Cancels(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branchCreating = true
	m.branchInput.Focus()
	m.branches = []string{"main"}

	result, _ := m.updateBranchMode(tea.KeyPressMsg{Text: "ctrl+c"})
	rm := result.(Model)
	if rm.branchCreating {
		t.Error("ctrl+c should cancel branch creation, not quit")
	}
}

func TestUpdateBranchMode_CreateMode_Enter_EmptyName(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branchCreating = true
	m.branchInput.Focus()
	m.branches = []string{"main"}

	result, cmd := m.updateBranchMode(tea.KeyPressMsg{Text: "enter"})
	rm := result.(Model)
	if !strings.Contains(rm.statusMsg, "empty") {
		t.Errorf("statusMsg=%q, want empty branch name error", rm.statusMsg)
	}
	if cmd == nil {
		t.Error("expected status clear cmd on empty name")
	}
}

func TestUpdateBranchMode_CreateMode_Enter_SubmitsCmd(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branchCreating = true
	m.branchInput.Focus()
	m.branchInput.SetValue("feature-x")
	m.branches = []string{"main"}

	_, cmd := m.updateBranchMode(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Error("expected async create branch cmd")
	}
}

func TestHandleBranchCreated_Success(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branchCreating = true

	result, cmd := m.handleBranchCreated(branchCreatedMsg{name: "feature-x"})
	rm := result.(Model)
	if rm.mode != modeFileList {
		t.Errorf("mode=%d, want modeFileList", rm.mode)
	}
	if rm.branchCreating {
		t.Error("branchCreating should be false")
	}
	if !strings.Contains(rm.statusMsg, "feature-x") {
		t.Errorf("statusMsg=%q, want branch name", rm.statusMsg)
	}
	if cmd == nil {
		t.Error("expected refresh files cmd")
	}
}

func TestHandleBranchCreated_Error(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branchCreating = true

	result, cmd := m.handleBranchCreated(branchCreatedMsg{
		name: "bad",
		err:  fmt.Errorf("already exists"),
	})
	rm := result.(Model)
	if rm.mode != modeBranchPicker {
		t.Error("should stay in branch picker on error")
	}
	if !strings.Contains(rm.statusMsg, "already exists") {
		t.Errorf("statusMsg=%q, want error", rm.statusMsg)
	}
	if cmd == nil {
		t.Error("expected status clear cmd on error")
	}
}

func TestRenderBranchCreateBar(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.branchCreating = true
	m.branchInput.Focus()
	bar := m.renderBranchCreateBar()
	if !strings.Contains(bar, "branch") {
		t.Error("create bar should contain 'branch' prompt")
	}
	if !strings.Contains(bar, "esc") {
		t.Error("create bar should show esc hint")
	}
	if !strings.Contains(bar, "enter") {
		t.Error("create bar should show enter hint")
	}
}

func TestView_BranchCreating_ShowsCreateBar(t *testing.T) {
	t.Parallel()
	// View() calls fileCardTitle() which needs a real repo for BranchName()
	// Use renderBranchCreateBar() directly to test view integration
	m := newTestModel(t, nil)
	m.mode = modeBranchPicker
	m.branchCreating = true
	m.branchInput.Focus()

	bar := m.renderBranchCreateBar()
	if !strings.Contains(bar, "new branch") {
		t.Error("create bar should show 'new branch' prompt")
	}
	if !strings.Contains(bar, "enter create") {
		t.Error("create bar should show 'enter create' hint")
	}
}

func TestPush_NoUpstream_OffersSetUpstream(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeFileList
	m.upstream = git.UpstreamInfo{} // no upstream
	m.currentBranch = "feature-x"

	// First P should offer set-upstream, not block
	result, _ := m.updateFileListMode(tea.KeyPressMsg{Text: "P"})
	rm := result.(Model)
	if !strings.Contains(rm.statusMsg, "set-upstream") {
		t.Errorf("statusMsg=%q, should mention set-upstream", rm.statusMsg)
	}
	if !rm.pushConfirm {
		t.Error("should enter push confirm state")
	}
	if !strings.Contains(rm.statusMsg, "feature-x") {
		t.Errorf("statusMsg=%q, should mention branch name", rm.statusMsg)
	}
}

func TestPush_NoUpstream_ConfirmPushes(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeFileList
	m.upstream = git.UpstreamInfo{} // no upstream
	m.pushConfirm = true
	m.currentBranch = "feature-x"

	// Second P should issue a push cmd
	result, cmd := m.updateFileListMode(tea.KeyPressMsg{Text: "P"})
	rm := result.(Model)
	if rm.pushConfirm {
		t.Error("pushConfirm should be cleared")
	}
	if !strings.Contains(rm.statusMsg, "pushing") {
		t.Errorf("statusMsg=%q, should say pushing", rm.statusMsg)
	}
	if cmd == nil {
		t.Error("expected push cmd")
	}
}

func TestEnterCommitMode_NoStaged_SetsStatus(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, []fileItem{{change: git.FileChange{Path: "a.go", Staged: false}}})

	result, cmd := m.enterCommitMode()
	rm := result.(Model)

	if cmd == nil {
		t.Error("expected status clear cmd")
	}
	if rm.mode != modeFileList {
		t.Errorf("mode=%v, want file list", rm.mode)
	}
	if !strings.Contains(rm.statusMsg, "no staged files") {
		t.Errorf("statusMsg=%q", rm.statusMsg)
	}
}

func TestEnterCommitMode_WithStaged_EntersCommitMode(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, []fileItem{{change: git.FileChange{Path: "a.go", Staged: true}}})

	result, cmd := m.enterCommitMode()
	rm := result.(Model)

	if rm.mode != modeCommit {
		t.Errorf("mode=%v, want commit", rm.mode)
	}
	if cmd == nil {
		t.Error("expected blink cmd")
	}
}

func TestUpdateCommitMode_EnterEmpty_ShowsError(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeCommit

	result, cmd := m.updateCommitMode(tea.KeyPressMsg{Text: "enter"})
	rm := result.(Model)

	if cmd == nil {
		t.Error("expected status clear cmd")
	}
	if !strings.Contains(rm.statusMsg, "empty commit message") {
		t.Errorf("statusMsg=%q", rm.statusMsg)
	}
}

func TestToggleStage_DisabledInStagedOnlyOrRef(t *testing.T) {
	t.Parallel()
	files := []fileItem{{change: git.FileChange{Path: "a.go", Staged: false}}}

	m1 := newTestModel(t, files)
	m1.stagedOnly = true
	_, cmd1 := m1.toggleStage()
	if cmd1 != nil {
		t.Error("toggleStage should be disabled for stagedOnly")
	}

	m2 := newTestModel(t, files)
	m2.ref = "main"
	_, cmd2 := m2.toggleStage()
	if cmd2 != nil {
		t.Error("toggleStage should be disabled for ref mode")
	}
}

func TestStageAll_DisabledInRefMode(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.ref = "main"

	_, cmd := m.stageAll()
	if cmd != nil {
		t.Error("stageAll should be disabled in ref mode")
	}
}

func TestHandleFilesRefreshed_EmptyClearsViewport(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, []fileItem{{change: git.FileChange{Path: "a.go"}}})
	m.viewport.SetContent("old")

	result, cmd := m.handleFilesRefreshed(filesRefreshedMsg{files: nil})
	rm := result.(Model)

	if cmd != nil {
		t.Error("expected no cmd")
	}
	if rm.cursor != 0 {
		t.Errorf("cursor=%d, want 0", rm.cursor)
	}
	if rm.viewport.View() != "" {
		t.Errorf("viewport=%q, want empty", rm.viewport.View())
	}
}

func TestHandleTick_SkipsPollingDuringCommit(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeCommit

	_, cmd := m.handleTick()
	if cmd == nil {
		t.Fatal("expected tick cmd")
	}
	msg := cmd()
	if _, ok := msg.(tickMsg); !ok {
		t.Errorf("msg type=%T, want tickMsg", msg)
	}
}

func TestPushConfirm_ResetOnNonPKey(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.mode = modeFileList
	m.pushConfirm = true

	result, _ := m.updateFileListMode(tea.KeyPressMsg{Text: "j"})
	rm := result.(Model)
	if rm.pushConfirm {
		t.Error("pushConfirm should reset on non-P key")
	}
}

func TestLoadDiffCmd_NilWhenZeroWidth(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, []fileItem{{change: git.FileChange{Path: "a.go"}}})
	m.width = 0

	cmd := m.loadDiffCmd(true)
	if cmd != nil {
		t.Error("expected loadDiffCmd to be nil when width is 0")
	}
}

func TestHandleResize_ReadyStateWithFiles(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, []fileItem{{change: git.FileChange{Path: "a.go"}}})
	m.ready = false

	res, _ := m.handleResize(tea.WindowSizeMsg{Width: 100, Height: 30})
	rm := res.(Model)

	if rm.ready {
		t.Error("expected ready to be false until diff is loaded when files exist")
	}
}

func TestHandleResize_ReadyStateNoFiles(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.ready = false

	res, _ := m.handleResize(tea.WindowSizeMsg{Width: 100, Height: 30})
	rm := res.(Model)

	if !rm.ready {
		t.Error("expected ready to be true immediately when no files exist")
	}
}

func TestHandleDiffLoaded_SetsReady(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, []fileItem{{change: git.FileChange{Path: "a.go"}}})
	m.ready = false
	m.cursor = 0

	res, _ := m.handleDiffLoaded(diffLoadedMsg{content: "some diff", index: 0, resetScroll: true})
	rm := res.(Model)

	if !rm.ready {
		t.Error("expected ready to be true after diff is loaded")
	}
}

func TestStatusBar_AutoDisappear(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)
	m.ready = true

	// 1. Send commitDoneMsg, which sets the final message "committed!"
	res, cmd := m.Update(commitDoneMsg{err: nil})
	rm := res.(Model)

	if rm.statusMsg != "committed!" {
		t.Errorf("expected statusMsg to be 'committed!', got %q", rm.statusMsg)
	}

	if cmd == nil {
		t.Error("expected a non-nil command to clear the status message")
	}

	// 2. Sending a clearStatusMsg with matching ID should clear statusMsg
	res2, _ := rm.Update(clearStatusMsg{id: rm.statusMsgID})
	rm2 := res2.(Model)

	if rm2.statusMsg != "" {
		t.Errorf("expected statusMsg to be cleared, got %q", rm2.statusMsg)
	}

	// 3. Sending clearStatusMsg with non-matching ID should NOT clear statusMsg
	rm.statusMsgID = 42
	res3, _ := rm.Update(clearStatusMsg{id: 99})
	rm3 := res3.(Model)

	if rm3.statusMsg != "committed!" {
		t.Errorf("expected statusMsg to remain 'committed!' for mismatched ID, got %q", rm3.statusMsg)
	}
}

