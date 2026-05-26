package ui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/dpuwork/differ/internal/theme"
)

// DiffLineType classifies a line in a unified diff.
type DiffLineType int

const (
	LineContext    DiffLineType = iota
	LineAdded
	LineRemoved
	LineHunkHeader
	LineFileHeader
)

// DiffLine is a single parsed line from a unified diff.
type DiffLine struct {
	Type    DiffLineType
	Content string
	OldNum  int // -1 if N/A
	NewNum  int // -1 if N/A
}

// ParsedDiff is the result of parsing a raw unified diff.
type ParsedDiff struct {
	Lines  []DiffLine
	Binary bool
}

const maxDiffLines = 10000

// ParseDiff parses raw unified diff output into structured lines.
func ParseDiff(raw string) ParsedDiff {
	if strings.Contains(raw, "Binary files") && strings.Contains(raw, "differ") {
		return ParsedDiff{Binary: true}
	}

	var lines []DiffLine
	oldNum, newNum := 0, 0

	for _, line := range strings.Split(raw, "\n") {
		if len(lines) >= maxDiffLines {
			lines = append(lines, DiffLine{
				Type: LineHunkHeader, Content: fmt.Sprintf("… truncated (%d+ lines)", maxDiffLines),
				OldNum: -1, NewNum: -1,
			})
			break
		}
		dl := parseDiffLine(line, &oldNum, &newNum)
		if dl != nil {
			lines = append(lines, *dl)
		}
	}
	return ParsedDiff{Lines: lines}
}

func parseDiffLine(line string, oldNum, newNum *int) *DiffLine {
	switch {
	case strings.HasPrefix(line, "diff --git"),
		strings.HasPrefix(line, "index "),
		strings.HasPrefix(line, "new file"),
		strings.HasPrefix(line, "deleted file"),
		strings.HasPrefix(line, "similarity"),
		strings.HasPrefix(line, "rename"),
		strings.HasPrefix(line, "old mode"),
		strings.HasPrefix(line, "new mode"),
		strings.HasPrefix(line, "--- "),
		strings.HasPrefix(line, "+++ "):
		// Skip raw git headers — we show a clean file banner instead
		return nil
	case strings.HasPrefix(line, "@@"):
		parseHunkHeader(line, oldNum, newNum)
		content := extractHunkContext(line)
		return &DiffLine{Type: LineHunkHeader, Content: content, OldNum: -1, NewNum: -1}
	case strings.HasPrefix(line, "+"):
		dl := &DiffLine{Type: LineAdded, Content: line[1:], OldNum: -1, NewNum: *newNum}
		*newNum++
		return dl
	case strings.HasPrefix(line, "-"):
		dl := &DiffLine{Type: LineRemoved, Content: line[1:], OldNum: *oldNum, NewNum: -1}
		*oldNum++
		return dl
	case strings.HasPrefix(line, `\`):
		return nil
	case line == "":
		return nil
	default:
		content := line
		if strings.HasPrefix(line, " ") {
			content = line[1:]
		}
		dl := &DiffLine{Type: LineContext, Content: content, OldNum: *oldNum, NewNum: *newNum}
		*oldNum++
		*newNum++
		return dl
	}
}

// extractHunkContext pulls the function/context part from a hunk header.
// "@@ -13,6 +13,7 @@ func main() {" → "func main() {"
// "@@ -13,6 +13,7 @@" → ""
func extractHunkContext(line string) string {
	parts := strings.SplitN(line, "@@", 3)
	if len(parts) == 3 {
		ctx := strings.TrimSpace(parts[2])
		if ctx != "" {
			return ctx
		}
	}
	// Show the range info as fallback
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[1])
	}
	return line
}

// parseHunkHeader extracts line numbers from @@ -old,count +new,count @@
func parseHunkHeader(line string, oldNum, newNum *int) {
	parts := strings.SplitN(line, "@@", 3)
	if len(parts) < 2 {
		return
	}
	ranges := strings.TrimSpace(parts[1])
	for _, r := range strings.Fields(ranges) {
		if strings.HasPrefix(r, "-") {
			nums := strings.SplitN(r[1:], ",", 2)
			if n, err := strconv.Atoi(nums[0]); err == nil {
				*oldNum = n
			}
		} else if strings.HasPrefix(r, "+") {
			nums := strings.SplitN(r[1:], ",", 2)
			if n, err := strconv.Atoi(nums[0]); err == nil {
				*newNum = n
			}
		}
	}
}

const lineNumWidth = 4

// RenderDiff renders parsed diff lines into a styled string.
func RenderDiff(parsed ParsedDiff, filename string, styles Styles, t theme.Theme, width int, wrap bool) string {
	if parsed.Binary {
		return RenderBinaryFile(styles, width)
	}
	initChromaStyle(t)

	var b strings.Builder
	for _, dl := range parsed.Lines {
		b.WriteString(renderDiffLine(dl, filename, styles, t, width, wrap))
		b.WriteByte('\n')
	}
	return b.String()
}

func renderDiffLine(dl DiffLine, filename string, styles Styles, t theme.Theme, width int, wrap bool) string {
	switch dl.Type {
	case LineHunkHeader:
		return renderHunkLine(dl, styles, width)
	default:
		return renderCodeLine(dl, filename, styles, t, width, wrap)
	}
}

func renderHunkLine(dl DiffLine, styles Styles, width int) string {
	prefix := styles.DiffLineNum.Render("    ···  ")
	text := dl.Content
	if text != "" {
		text = " " + text
	}
	return prefix + styles.DiffHunkHeader.Render(text)
}

func renderCodeLine(dl DiffLine, filename string, styles Styles, t theme.Theme, width int, wrap bool) string {
	oldNum := fmtLineNum(dl.OldNum)
	newNum := fmtLineNum(dl.NewNum)

	indicator := " "
	var bgColor string
	var numStyle lipgloss.Style
	var indStyle lipgloss.Style
	var bgStyle lipgloss.Style
	switch dl.Type {
	case LineAdded:
		indicator = "+"
		bgColor = t.AddedBg
		numStyle = styles.DiffLineNumAdded
		indStyle = styles.DiffAdded
		bgStyle = styles.DiffAddedBg
	case LineRemoved:
		indicator = "-"
		bgColor = t.RemovedBg
		numStyle = styles.DiffLineNumRemoved
		indStyle = styles.DiffRemoved
		bgStyle = styles.DiffRemovedBg
	default:
		numStyle = styles.DiffLineNum
		indStyle = styles.DiffContext
		bgStyle = lipgloss.NewStyle()
	}

	nums := numStyle.Render(oldNum + " " + newNum)

	// Syntax highlight the content
	highlighted := highlightLine(dl.Content, filename, bgColor)

	codeWidth := width - lineNumWidth*2 - 3 // nums + spaces
	prefix := indStyle.Render(indicator + " ")
	textWidth := codeWidth - lipgloss.Width(prefix)
	if textWidth < 1 {
		textWidth = 1
	}

	if !wrap {
		highlighted = lipgloss.NewStyle().MaxWidth(textWidth).Render(highlighted)
		contentWidth := lipgloss.Width(prefix) + lipgloss.Width(highlighted)
		padding := ""
		if pad := codeWidth - contentWidth; pad > 0 {
			padding = bgStyle.Render(strings.Repeat(" ", pad))
		}
		return nums + " " + prefix + highlighted + padding
	}

	// Wrap text
	wrapped := lipgloss.NewStyle().Width(textWidth).Render(highlighted)
	lines := strings.Split(wrapped, "\n")
	var b strings.Builder
	for i, line := range lines {
		line = strings.TrimRight(line, " ")
		if i > 0 {
			b.WriteByte('\n')
			nums = numStyle.Render("    " + " " + "    ")
			prefix = indStyle.Render("  ")
		}
		contentWidth := lipgloss.Width(prefix) + lipgloss.Width(line)
		padding := ""
		if pad := codeWidth - contentWidth; pad > 0 {
			padding = bgStyle.Render(strings.Repeat(" ", pad))
		}
		b.WriteString(nums + " " + prefix + line + padding)
	}
	return b.String()
}

func fmtLineNum(n int) string {
	if n < 0 {
		return "    "
	}
	return fmt.Sprintf("%4d", n)
}

// RenderNewFile renders file content as an all-added diff (for untracked files).
func RenderNewFile(content, filename string, styles Styles, t theme.Theme, width int, wrap bool) string {
	initChromaStyle(t)

	var b strings.Builder
	codeWidth := width - lineNumWidth*2 - 3
	prefix := styles.DiffAdded.Render("+ ")
	textWidth := codeWidth - lipgloss.Width(prefix)
	if textWidth < 1 {
		textWidth = 1
	}

	for i, line := range strings.Split(content, "\n") {
		num := i + 1
		nums := styles.DiffLineNumAdded.Render("     " + fmt.Sprintf("%4d", num))
		highlighted := highlightLine(line, filename, t.AddedBg)

		if !wrap {
			highlighted = lipgloss.NewStyle().MaxWidth(textWidth).Render(highlighted)
			contentWidth := lipgloss.Width(prefix) + lipgloss.Width(highlighted)
			padding := ""
			if pad := codeWidth - contentWidth; pad > 0 {
				padding = styles.DiffAddedBg.Render(strings.Repeat(" ", pad))
			}
			b.WriteString(nums + " " + prefix + highlighted + padding)
			b.WriteByte('\n')
			continue
		}

		wrapped := lipgloss.NewStyle().Width(textWidth).Render(highlighted)
		wrappedLines := strings.Split(wrapped, "\n")
		for j, wl := range wrappedLines {
			wl = strings.TrimRight(wl, " ")
			curNums := nums
			curPrefix := prefix
			if j > 0 {
				curNums = styles.DiffLineNumAdded.Render("         ")
				curPrefix = styles.DiffAdded.Render("  ")
			}
			contentWidth := lipgloss.Width(curPrefix) + lipgloss.Width(wl)
			padding := ""
			if pad := codeWidth - contentWidth; pad > 0 {
				padding = styles.DiffAddedBg.Render(strings.Repeat(" ", pad))
			}
			b.WriteString(curNums + " " + curPrefix + wl + padding)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// RenderBinaryFile renders a placeholder for binary files.
func RenderBinaryFile(styles Styles, width int) string {
	return styles.DiffHunkHeader.Width(width).Render("  Binary file — cannot display diff")
}

// --- Split (side-by-side) diff ---

const minSplitWidth = 60

// SplitLine pairs left (old) and right (new) sides for side-by-side display.
type SplitLine struct {
	Left  *DiffLine // nil = blank padding
	Right *DiffLine // nil = blank padding
}

// PairLines converts unified diff lines into paired split lines.
func PairLines(lines []DiffLine) []SplitLine {
	var result []SplitLine
	i := 0
	for i < len(lines) {
		dl := lines[i]
		switch dl.Type {
		case LineHunkHeader:
			result = append(result, SplitLine{Left: &dl})
			i++
		case LineContext:
			result = append(result, SplitLine{Left: &dl, Right: &dl})
			i++
		case LineRemoved:
			// Collect contiguous removed, then contiguous added
			var removed, added []DiffLine
			for i < len(lines) && lines[i].Type == LineRemoved {
				removed = append(removed, lines[i])
				i++
			}
			for i < len(lines) && lines[i].Type == LineAdded {
				added = append(added, lines[i])
				i++
			}
			maxLen := len(removed)
			if len(added) > maxLen {
				maxLen = len(added)
			}
			for j := 0; j < maxLen; j++ {
				var l, r *DiffLine
				if j < len(removed) {
					l = &removed[j]
				}
				if j < len(added) {
					r = &added[j]
				}
				result = append(result, SplitLine{Left: l, Right: r})
			}
		case LineAdded:
			// Orphan added (no preceding removed)
			result = append(result, SplitLine{Right: &dl})
			i++
		default:
			i++
		}
	}
	return result
}

// RenderSplitDiff renders parsed diff in side-by-side layout.
func RenderSplitDiff(parsed ParsedDiff, filename string, styles Styles, t theme.Theme, width int, wrap bool) string {
	if parsed.Binary {
		return RenderBinaryFile(styles, width)
	}
	initChromaStyle(t)

	pairs := PairLines(parsed.Lines)
	panelW := (width - 1) / 2 // 1 char for separator

	var b strings.Builder
	for _, sl := range pairs {
		// Hunk headers span full width
		if sl.Left != nil && sl.Left.Type == LineHunkHeader {
			b.WriteString(renderHunkLine(*sl.Left, styles, width))
			b.WriteByte('\n')
			continue
		}
		
		leftLines := renderSplitSideLines(sl.Left, filename, styles, t, panelW, true, wrap)
		rightLines := renderSplitSideLines(sl.Right, filename, styles, t, panelW, false, wrap)
		
		maxLines := len(leftLines)
		if len(rightLines) > maxLines {
			maxLines = len(rightLines)
		}
		
		for i := 0; i < maxLines; i++ {
			left := ""
			if i < len(leftLines) {
				left = leftLines[i]
			} else {
				left = renderSplitSideLines(nil, filename, styles, t, panelW, true, wrap)[0]
			}
			right := ""
			if i < len(rightLines) {
				right = rightLines[i]
			} else {
				right = renderSplitSideLines(nil, filename, styles, t, panelW, false, wrap)[0]
			}
			b.WriteString(left)
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(t.BorderFg)).Render("│"))
			b.WriteString(right)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func RenderNewFileSplit(content, filename string, styles Styles, t theme.Theme, width int, wrap bool) string {
	initChromaStyle(t)

	panelW := (width - 1) / 2
	var b strings.Builder
	for i, line := range strings.Split(content, "\n") {
		dl := DiffLine{Type: LineAdded, Content: line, OldNum: -1, NewNum: i + 1}
		
		leftLines := renderSplitSideLines(nil, filename, styles, t, panelW, true, wrap)
		rightLines := renderSplitSideLines(&dl, filename, styles, t, panelW, false, wrap)
		
		maxLines := len(leftLines)
		if len(rightLines) > maxLines {
			maxLines = len(rightLines)
		}
		
		for j := 0; j < maxLines; j++ {
			left := ""
			if j < len(leftLines) {
				left = leftLines[j]
			} else {
				left = renderSplitSideLines(nil, filename, styles, t, panelW, true, wrap)[0]
			}
			right := ""
			if j < len(rightLines) {
				right = rightLines[j]
			} else {
				right = renderSplitSideLines(nil, filename, styles, t, panelW, false, wrap)[0]
			}
			b.WriteString(left)
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(t.BorderFg)).Render("│"))
			b.WriteString(right)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

const splitLineNumWidth = 4

func renderSplitSideLines(dl *DiffLine, filename string, styles Styles, t theme.Theme, panelW int, isLeft bool, wrap bool) []string {
	if dl == nil {
		if panelW > 0 {
			return []string{strings.Repeat(" ", panelW)}
		}
		return []string{""}
	}

	// Pick line number
	num := dl.OldNum
	if !isLeft {
		num = dl.NewNum
	}
	numStr := fmtLineNum(num)

	// Style selection
	indicator := " "
	var bgColor string
	var numStyle lipgloss.Style
	var indStyle lipgloss.Style
	var bgStyle lipgloss.Style

	switch dl.Type {
	case LineAdded:
		indicator = "+"
		bgColor = t.AddedBg
		numStyle = styles.DiffLineNumAdded
		indStyle = styles.DiffAdded
		bgStyle = styles.DiffAddedBg
	case LineRemoved:
		indicator = "-"
		bgColor = t.RemovedBg
		numStyle = styles.DiffLineNumRemoved
		indStyle = styles.DiffRemoved
		bgStyle = styles.DiffRemovedBg
	default:
		numStyle = styles.DiffLineNum
		indStyle = styles.DiffContext
		bgStyle = lipgloss.NewStyle()
	}

	nums := numStyle.Render(numStr)
	highlighted := highlightLine(dl.Content, filename, bgColor)
	prefix := indStyle.Render(indicator + " ")

	codeWidth := max(0, panelW-splitLineNumWidth-1)
	textWidth := codeWidth - lipgloss.Width(prefix)
	if textWidth < 1 {
		textWidth = 1
	}

	if !wrap {
		highlighted = lipgloss.NewStyle().MaxWidth(textWidth).Render(highlighted)
		contentWidth := lipgloss.Width(prefix) + lipgloss.Width(highlighted)
		padding := ""
		if pad := codeWidth - contentWidth; pad > 0 {
			padding = bgStyle.Render(strings.Repeat(" ", pad))
		}
		return []string{nums + " " + prefix + highlighted + padding}
	}

	wrapped := lipgloss.NewStyle().Width(textWidth).Render(highlighted)
	lines := strings.Split(wrapped, "\n")
	var result []string
	for i, line := range lines {
		line = strings.TrimRight(line, " ")
		curNums := nums
		curPrefix := prefix
		if i > 0 {
			curNums = numStyle.Render("    ")
			curPrefix = indStyle.Render("  ")
		}
		contentWidth := lipgloss.Width(curPrefix) + lipgloss.Width(line)
		padding := ""
		if pad := codeWidth - contentWidth; pad > 0 {
			padding = bgStyle.Render(strings.Repeat(" ", pad))
		}
		result = append(result, curNums+" "+curPrefix+line+padding)
	}
	return result
}
