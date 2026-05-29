package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/dpuwork/differ/internal/config"
	"github.com/dpuwork/differ/internal/git"
	"github.com/dpuwork/differ/internal/theme"
)

type viewMode int

const (
	modeFileList viewMode = iota
	modeDiff
	modeCommit
	modeBranchPicker
	modeHelp
)

const fileListWidth = 35
const pollInterval = 2 * time.Second
const statusTimeout = 7 * time.Second

const (
	minWidth  = 60
	minHeight = 10
)

type tickMsg time.Time

type diffLoadedMsg struct {
	content     string
	index       int
	resetScroll bool
}

type filesRefreshedMsg struct{ files []fileItem }
type commitDoneMsg struct{ err error }

type branchesLoadedMsg struct {
	branches []string
	current  string
	err      error
}

type branchSwitchedMsg struct{ err error }

type upstreamStatusMsg struct{ info git.UpstreamInfo }
type pushDoneMsg struct{ err error }
type pullDoneMsg struct{ err error }
type savePrefDoneMsg struct{ err error }
type clearStatusMsg struct{ id int }

type editorFinishedMsg struct{ err error }

type branchCreatedMsg struct {
	name string
	err  error
}

// Model holds all UI state; behavior split across focused files.
type Model struct {
	repo       *git.Repo
	cfg        config.Config
	files      []fileItem
	styles     Styles
	theme      theme.Theme
	stagedOnly bool
	ref        string
	version    string

	mode          viewMode
	prevMode      viewMode
	cursor        int
	prevCurs      int
	viewport      viewport.Model
	commitInput   textinput.Model
	statusMsg     string
	statusMsgID   int
	splitDiff     bool
	wrapDiff      bool
	width         int
	height        int
	ready         bool
	SelectedFile  string

	lastDiffContent string

	branches         []string
	filteredBranches []string
	branchCursor     int
	branchOffset     int
	currentBranch    string
	branchFilter     textinput.Model
	branchCreating   bool
	branchInput      textinput.Model
	initialFetchDone bool

	upstream    git.UpstreamInfo
	pushConfirm bool
	discardConfirm bool
}

type fileItem struct {
	change    git.FileChange
	untracked bool
}

func NewModel(repo *git.Repo, cfg config.Config, changes []git.FileChange, untracked []string, styles Styles, t theme.Theme, stagedOnly bool, ref string, version string) Model {
	files := buildFileItems(repo, changes, untracked)

	ti := textinput.New()
	ti.Placeholder = "commit message..."
	ti.CharLimit = 200
	ti.Prompt = ""

	bf := textinput.New()
	bf.Placeholder = "filter..."
	bf.CharLimit = 100
	bf.SetWidth(fileListWidth - 8)
	bf.Prompt = ""

	bi := textinput.New()
	bi.Placeholder = "branch name..."
	bi.CharLimit = 100
	bi.Prompt = ""

	return Model{
		repo:         repo,
		cfg:          cfg,
		files:        files,
		styles:       styles,
		theme:        t,
		stagedOnly:   stagedOnly,
		ref:          ref,
		version:      version,
		splitDiff:    cfg.SplitDiff,
		wrapDiff:     cfg.WrapDiff,
		prevCurs:     -1,
		commitInput:  ti,
		branchFilter: bf,
		branchInput:  bi,
	}
}

func buildFileItems(repo *git.Repo, changes []git.FileChange, untracked []string) []fileItem {
	var files []fileItem
	for _, c := range changes {
		files = append(files, fileItem{change: c})
	}
	for _, path := range untracked {
		added := 0
		if repo != nil {
			raw, err := repo.ReadFileContent(path)
			if err == nil {
				added = countLines(raw)
			}
		}
		files = append(files, fileItem{change: git.FileChange{Path: path, Status: git.StatusUntracked, AddedLines: added}, untracked: true})
	}

	sort.SliceStable(files, func(i, j int) bool {
		if files[i].change.Staged != files[j].change.Staged {
			return files[i].change.Staged
		}
		return files[i].change.Path < files[j].change.Path
	})

	return files
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	count := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		count++
	}
	return count
}

func filesEqual(a, b []fileItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (m *Model) refreshTheme() {
	m.styles = NewStyles(m.theme)
}

func (m *Model) StartInCommitMode() {
	m.mode = modeCommit
	m.commitInput.Focus()
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.loadDiffCmd(true), m.fetchUpstreamStatusCmd(), tickFetchCmd(), tickCmd()}
	if m.mode == modeCommit {
		cmds = append(cmds, textinput.Blink)
	}
	return tea.Batch(cmds...)
}

func (m Model) contentHeight() int {
	h := m.height - 3 // 2 for card borders + 1 for status bar
	if m.mode == modeCommit || (m.mode == modeBranchPicker && m.branchCreating) {
		h--
	}
	return h
}
func (m Model) diffWidth() int     { return m.width - fileListWidth - 2 - 1 - 2 }

func (m Model) setStatus(msg string, autoClear bool) (Model, tea.Cmd) {
	m.statusMsg = msg
	if autoClear && msg != "" {
		m.statusMsgID++
		id := m.statusMsgID
		return m, tea.Tick(statusTimeout, func(t time.Time) tea.Msg {
			return clearStatusMsg{id: id}
		})
	}
	return m, nil
}

func isTmux() bool {
	return os.Getenv("TMUX") != ""
}

func (m Model) openEditorTmuxCmd(file string) tea.Cmd {
	return func() tea.Msg {
		repoRoot := m.repo.Dir()
		absPath := filepath.Join(repoRoot, file)
		editorCmd := m.cfg.EditorCmd
		if editorCmd == "" {
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			editorCmd = editor + " {file}"
		}
		expanded := strings.ReplaceAll(editorCmd, "{file}", absPath)
		expanded = strings.ReplaceAll(expanded, "{repo}", repoRoot)

		// Same parsing logic as root.go
		parts := strings.Fields(expanded)

		// Run in tmux popup
		args := []string{"popup", "-w", "85%", "-h", "85%", "-E"}
		args = append(args, parts...)
		cmd := exec.Command("tmux", args...)
		cmd.Dir = repoRoot
		err := cmd.Run()
		return editorFinishedMsg{err: err}
	}
}
