package ui

import (
	"strings"
	"testing"

	"github.com/dpuwork/differ/internal/git"
)

func TestStyleStatus(t *testing.T) {
	m := newTestModel(t, nil)

	statuses := []struct {
		name   string
		status git.FileStatus
		icon   string
	}{
		{"modified", git.StatusModified, "M"},
		{"added", git.StatusAdded, "A"},
		{"deleted", git.StatusDeleted, "D"},
		{"renamed", git.StatusRenamed, "R"},
		{"untracked", git.StatusUntracked, "?"},
		{"type_changed", git.StatusTypeChanged, "T"},
		{"unmerged", git.StatusUnmerged, "U"},
		{"ignored", git.StatusIgnored, "!"},
		{"unknown", git.StatusUnknown, "X"},
	}

	for _, tt := range statuses {
		t.Run(tt.name, func(t *testing.T) {
			got := m.styleStatus(tt.icon, tt.status)
			if !strings.Contains(got, tt.icon) {
				t.Errorf("styleStatus(%q, %c) = %q, want it to contain %q", tt.icon, tt.status, got, tt.icon)
			}
			// Check that it's styled (contains escape codes if not default)
			if tt.status != 0 && !strings.Contains(got, "\x1b[") {
				t.Errorf("styleStatus(%q, %c) = %q, expected escape codes for styling", tt.icon, tt.status, got)
			}
		})
	}
}

func TestRenderFileItem_NewStatuses(t *testing.T) {
	t.Parallel()
	m := newTestModel(t, nil)

	tests := []struct {
		name   string
		status git.FileStatus
		icon   string
	}{
		{"type_changed", git.StatusTypeChanged, "T"},
		{"unmerged", git.StatusUnmerged, "U"},
		{"ignored", git.StatusIgnored, "!"},
		{"unknown", git.StatusUnknown, "X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := fileItem{change: git.FileChange{Path: "test.txt", Status: tt.status}}
			out := m.renderFileItem(item, false)
			if !strings.Contains(out, tt.icon) {
				t.Errorf("renderFileItem with status %c missing icon %q, got %q", tt.status, tt.icon, out)
			}
		})
	}
}
