package filechange

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dlclark/regexp2"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type LineChange struct {
	LineNo        int
	Original      string
	Replaced      *string
	DiffPretty    string
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
	base            = lipgloss.NewStyle().Foreground(lipgloss.Color("#888"))
	focused         = lipgloss.NewStyle().Background(lipgloss.Color("#0000ff"))
	disabled        = lipgloss.NewStyle().Foreground(lipgloss.Color("#555")).Strikethrough(true)
	focusedDisabled = lipgloss.NewStyle().Background(lipgloss.Color("#888")).Foreground(lipgloss.Color("#555")).Strikethrough(true)
)

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

		if match, _ := regex.FindStringMatch(line); match != nil {
			var change LineChange

			if subst == nil {
				diffPretty := line[:match.Index] + lipgloss.NewStyle().Foreground(lipgloss.Color("#0000ff")).Render(match.String()) + line[match.Index+match.Length:]

				preview = append(preview, ">"+diffPretty)

				change = LineChange{lineNo, line, nil, diffPretty, len(preview), true}
			} else {
				replaced, err := regex.Replace(line, *subst, -1, -1)

				if err != nil {
					log.Fatal(err)
				}

				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(line, replaced, false)
				diffPretty := dmp.DiffPrettyText(diffs)

				change = LineChange{lineNo, line, &replaced, diffPretty, len(preview), true}

				preview = append(preview, lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("-"+line))
				preview = append(preview, lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Render("+"+replaced))
			}

			lineChanges = append(lineChanges, &change)
		} else {
			preview = append(preview, " "+line)
		}
	}

	if len(lineChanges) > 0 {
		return &Model{file.Name(), lineChanges, lines, -1, preview}, nil
	}

	return nil, nil
}

func (m Model) Replace() {
	for _, c := range m.lineChanges {
		if !c.enabled || c.Replaced == nil {
			continue
		}

		m.lines[c.LineNo-1] = *c.Replaced
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

func (m Model) OpenInEditor() {
	exec.Command(os.Getenv("EDITOR"), m.filename).Output()
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
	strs = append(strs, styleFor(0).Copy().Foreground(lipgloss.Color("#fff")).Bold(true).Render(m.filename))

	for i, c := range m.lineChanges {
		s := styleFor(i+1).Render(fmt.Sprintf("%4s: ", strconv.Itoa(c.LineNo))) + c.DiffPretty
		strs = append(strs, s)
	}

	return strings.Join(strs, "\n")
}
