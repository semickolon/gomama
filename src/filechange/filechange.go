package filechange

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dlclark/regexp2"
	replacer "github.com/semickolon/gomama/src/replacer"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type LineChange struct {
	LineNo        int
	Original      string
	Replaced      *string
	Diffs         []diffmatchpatch.Diff
	PreviewLineNo int
	enabled       bool
}

type Model struct {
	filename    string
	lineChanges []*LineChange
	lines       []string

	Cursor  int
	Preview []string
}

type UpdateMsg tea.Cmd
type PastHeadMsg struct{}
type PastTailMsg struct{}

var (
	base            = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0def4"))
	focused         = lipgloss.NewStyle().Background(lipgloss.Color("#524f67"))
	disabled        = lipgloss.NewStyle().Foreground(lipgloss.Color("#6e6a86"))
	focusedDisabled = lipgloss.NewStyle().Background(lipgloss.Color("#6e6a86")).Foreground(lipgloss.Color("#444"))

	diffInsert = lipgloss.NewStyle().Foreground(lipgloss.Color("#31748f"))
	diffDelete = lipgloss.NewStyle().Foreground(lipgloss.Color("#eb6f92")).Strikethrough(true)
	diffEqual  = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0def4"))

	diffDisabledInsert = lipgloss.NewStyle().Foreground(lipgloss.Color("#1b4251"))
	diffDisabledDelete = lipgloss.NewStyle().Foreground(lipgloss.Color("#512632")).Strikethrough(true)
	diffDisabledEqual  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4a4a51"))

	diffReview         = lipgloss.NewStyle().Foreground(lipgloss.Color("#c4a7e7"))
	diffDisabledReview = lipgloss.NewStyle().Foreground(lipgloss.Color("#453a51"))
)

func DiffPrettyText(diffs []diffmatchpatch.Diff, enabled bool, reviewMode bool) string {
	ternary := func(cond bool, a, b lipgloss.Style) lipgloss.Style {
		if cond {
			return a
		} else {
			return b
		}
	}

	styleFor := func(diff diffmatchpatch.Diff) lipgloss.Style {

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			return ternary(
				enabled,
				ternary(reviewMode, diffReview, diffInsert),
				ternary(reviewMode, diffDisabledReview, diffDisabledInsert),
			)
		case diffmatchpatch.DiffDelete:
			return ternary(enabled, diffDelete, diffDisabledDelete)
		case diffmatchpatch.DiffEqual:
			return ternary(enabled, diffEqual, diffDisabledEqual)
		}
		return diffDisabledEqual
	}

	var buff bytes.Buffer
	for _, diff := range diffs {
		buff.WriteString(styleFor(diff).Render(diff.Text))
	}

	return buff.String()
}

func New(file *os.File, regex *regexp2.Regexp, subst *string) (*Model, error) {
	defer file.Close()

	scanner := bufio.NewScanner(file)

	lineNo := 0
	lineChanges := []*LineChange{}
	lines := []string{}
	preview := []string{}

	for scanner.Scan() {
		lineNo += 1
		line := scanner.Text()
		lines = append(lines, line)

		if match, _ := replacer.Match(line, regex); match != nil {
			var change LineChange

			if subst == nil {
				diffs := []diffmatchpatch.Diff{
					{
						Type: diffmatchpatch.DiffEqual,
						Text: line[:match.Index],
					},
					{
						Type: diffmatchpatch.DiffInsert,
						Text: match.String(),
					},
					{
						Type: diffmatchpatch.DiffEqual,
						Text: line[match.Index+match.Length:],
					},
				}

				preview = append(preview, diffReview.Render(">")+DiffPrettyText(diffs, true, true))

				change = LineChange{lineNo, line, nil, diffs, len(preview), true}
			} else {
				replaced, err := replacer.Replace(line, regex, *subst)

				if err != nil {
					log.Fatal(err)
				}

				if line == replaced {
					continue
				}

				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(line, replaced, false)

				change = LineChange{lineNo, line, &replaced, diffs, len(preview), true}

				preview = append(preview, lipgloss.NewStyle().Foreground(lipgloss.Color("#eb6f92")).Render("-"+line))
				preview = append(preview, lipgloss.NewStyle().Foreground(lipgloss.Color("#31748f")).Render("+"+replaced))
			}

			lineChanges = append(lineChanges, &change)
		} else {
			preview = append(preview, " "+line)
		}
	}

	if len(lineChanges) > 0 {
		// TODO: Last empty line is not returned. We add an empty line here to compensate.
		//   Though this is unwanted behavior. We are just a find&replacer. Not a newline adder.
		lines = append(lines, "")
		return &Model{file.Name(), lineChanges, lines, -1, preview}, nil
	}

	return nil, nil
}

func (m Model) Replace() {
	replaceCount := 0

	for _, c := range m.lineChanges {
		if !c.enabled || c.Replaced == nil {
			continue
		}

		m.lines[c.LineNo-1] = *c.Replaced
		replaceCount++
	}

	if replaceCount == 0 {
		return
	}

	f, err := os.OpenFile(m.filename, os.O_WRONLY, os.ModePerm)

	if err != nil {
		panic(err)
	}

	f.Truncate(0)
	_, err = f.WriteString(strings.Join(m.lines, "\n"))

	if err != nil {
		panic(err)
	}

	f.Close()
}

func (m Model) OpenInEditor() tea.Cmd {
	c := exec.Command(os.Getenv("EDITOR"), m.filename)
	callback := func(err error) tea.Msg {
		return tea.ClearScreen
	}
	return tea.ExecProcess(c, callback)
}

func (m Model) GetHeight() int {
	return len(m.lineChanges) + 1
}

func (m Model) GetFocusedLineChange() *LineChange {
	if m.Cursor > 0 && m.Cursor <= len(m.lineChanges) {
		return m.lineChanges[m.Cursor-1]
	}
	return nil
}

func (m *Model) MoveCursor(lines int) int {
	newCursor := m.Cursor + lines
	lenChanges := len(m.lineChanges)

	if newCursor < 0 {
		m.Cursor = -1
		return newCursor
	} else if newCursor > lenChanges {
		m.Cursor = lenChanges + 1
		return newCursor - lenChanges
	} else {
		m.Cursor = newCursor
		return 0
	}
}

func (m *Model) MoveCursorToHead() {
	m.Cursor = 0
}

func (m *Model) MoveCursorToTail() {
	m.Cursor = len(m.lineChanges)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.ToggleItemEnabled(m.Cursor)
		}
	}

	return m, nil
}

func (m Model) IsItemEnabled(idx int) bool {
	if idx > 0 {
		return m.lineChanges[idx-1].enabled
	} else if idx == 0 {
		return !m.AllLineChangesDisabled()
	} else {
		return false
	}
}

func (m *Model) ToggleItemEnabled(idx int) {
	if idx > 0 {
		c := m.lineChanges[idx-1]
		c.enabled = !c.enabled
	} else if idx == 0 {
		m.SetLineChangesEnabled(!m.IsItemEnabled(0))
	}
}

func (m Model) AllLineChangesDisabled() bool {
	for _, c := range m.lineChanges {
		if c.enabled {
			return false
		}
	}
	return true
}

func (m *Model) SetLineChangesEnabled(enabled bool) {
	for _, c := range m.lineChanges {
		c.enabled = enabled
	}
}

func (m Model) View() string {
	styleFor := func(idx int) lipgloss.Style {
		enabled := m.IsItemEnabled(idx)

		if idx == m.Cursor {
			if enabled {
				return focused
			} else {
				return focusedDisabled
			}
		} else {
			if enabled {
				return base
			} else {
				return disabled
			}
		}
	}

	strs := make([]string, 0, m.GetHeight())
	strs = append(strs, styleFor(0).Copy().Bold(true).Render(m.filename))

	for i, c := range m.lineChanges {
		style := styleFor(i + 1)
		s := style.Render(fmt.Sprintf("%4d: ", c.LineNo)) + DiffPrettyText(c.Diffs, c.enabled, c.Replaced == nil)
		strs = append(strs, s)
	}

	return strings.Join(strs, "\n")
}
