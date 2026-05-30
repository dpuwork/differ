package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/dpuwork/differ/internal/config"
	"github.com/dpuwork/differ/internal/git"
	"github.com/dpuwork/differ/internal/theme"
	"github.com/dpuwork/differ/internal/ui"
	"github.com/spf13/cobra"

	tea "charm.land/bubbletea/v2"
)

var version = "dev"

var (
	flagStaged bool
	flagRef    string
	flagCommit bool
)

var rootCmd = &cobra.Command{
	Use:     "differ",
	Short:   "Git diff TUI viewer",
	Version: version,
	RunE:    runDiff,
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Browse recent commits with diff preview",
	RunE:  runLog,
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Review staged changes and commit",
	RunE:  runCommit,
}

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
	rootCmd.Flags().BoolVarP(&flagStaged, "staged", "s", false, "show only staged changes")
	rootCmd.Flags().StringVarP(&flagRef, "ref", "r", "", "compare against branch/tag/commit")
	rootCmd.Flags().BoolVarP(&flagCommit, "commit", "c", false, "enter commit mode after review")
	rootCmd.AddCommand(logCmd, commitCmd)
}

// Execute runs the root CLI command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runDiff(cmd *cobra.Command, args []string) error {
	repo, err := git.NewRepo(".")
	if err != nil {
		return err
	}

	files, err := repo.ChangedFiles(flagStaged, flagRef)
	if err != nil {
		return err
	}

	var untracked []string
	if !flagStaged && flagRef == "" {
		untracked, err = repo.UntrackedFiles()
		if err != nil {
			return err
		}
	}

	cfg := config.Load()
	isDark := theme.IsDarkBackground()
	t := theme.GetTheme(isDark)
	styles := ui.NewStyles(t)

	model := ui.NewModel(repo, cfg, files, untracked, styles, t, flagStaged, flagRef, version)
	if flagCommit {
		model.StartInCommitMode()
	}
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	if m, ok := finalModel.(ui.Model); ok && m.SelectedFile != "" {
		return openInEditor(cfg.EditorCmd, m.SelectedFile, repo.Dir())
	}
	return nil
}

func openInEditor(editorCmd, file, repoRoot string) error {
	absPath := filepath.Join(repoRoot, file)
	if editorCmd == "" {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		editorCmd = editor + " {file}"
	}
	expanded := strings.ReplaceAll(editorCmd, "{file}", absPath)
	expanded = strings.ReplaceAll(expanded, "{repo}", repoRoot)
	parts := strings.Fields(expanded)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = repoRoot
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommit(cmd *cobra.Command, args []string) error {
	repo, err := git.NewRepo(".")
	if err != nil {
		return err
	}

	files, err := repo.ChangedFiles(true, "")
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("No staged changes to commit.")
		return nil
	}

	cfg := config.Load()
	isDark := theme.IsDarkBackground()
	t := theme.GetTheme(isDark)
	styles := ui.NewStyles(t)

	model := ui.NewModel(repo, cfg, files, nil, styles, t, true, "", version)
	model.StartInCommitMode()
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}

func runLog(cmd *cobra.Command, args []string) error {
	repo, err := git.NewRepo(".")
	if err != nil {
		return err
	}
	if !repo.HasCommits() {
		fmt.Println("No commits yet.")
		return nil
	}

	cfg := config.Load()
	isDark := theme.IsDarkBackground()
	t := theme.GetTheme(isDark)
	styles := ui.NewStyles(t)

	model := ui.NewLogModel(repo, cfg, styles, t, version)
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}
