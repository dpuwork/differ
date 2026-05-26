package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscardFile_Tracked(t *testing.T) {
	t.Parallel()
	repo := setupTestRepo(t)
	addCommit(t, repo, "f.txt", "v1", "init")
	writeFile(t, repo, "f.txt", "v2")

	if err := repo.DiscardFile("f.txt", false); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(repo.Dir(), "f.txt"))
	if string(content) != "v1" {
		t.Errorf("content=%q, want v1", string(content))
	}
}

func TestDiscardFile_Untracked(t *testing.T) {
	t.Parallel()
	repo := setupTestRepo(t)
	addCommit(t, repo, "f.txt", "v1", "init")
	writeFile(t, repo, "untracked.txt", "x")

	if err := repo.DiscardFile("untracked.txt", true); err != nil {
		t.Fatal(err)
	}

	_, err := os.Stat(filepath.Join(repo.Dir(), "untracked.txt"))
	if !os.IsNotExist(err) {
		t.Error("untracked file should be deleted")
	}
}
