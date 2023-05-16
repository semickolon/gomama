package changelist

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dlclark/regexp2"
	filechange "github.com/semickolon/gomama/src/filechange"
)

type Model struct {
	viewport  viewport.Model
	filePager viewport.Model

	changes      []filechange.Model
	selectedIdx  int
	ready        bool
	pagerFocused bool
}

func New(files []*os.File, regex *regexp2.Regexp, subst *string) (*Model, error) {
	changes := make([]filechange.Model, 0, len(files))

	for _, file := range files {
		if c, err := filechange.New(file, regex, subst); err != nil {
			return nil, err
		} else if c != nil {
			changes = append(changes, *c)
		}
	}

	if len(changes) == 0 {
		return nil, nil
	}

	changes[0].Cursor = 0

	m := Model{changes: changes}
	return &m, nil
}

func (m Model) GetCursorYOffset() int {
	y := m.selectedIdx

	for i := 0; i < m.selectedIdx; i++ {
		y += m.changes[i].GetHeight()
	}

	y += m.changes[m.selectedIdx].Cursor
	return y
}

func (m Model) GetCurrentChange() *filechange.Model {
	return &m.changes[m.selectedIdx]
}

func (m *Model) MoveCursor(lines int) {
	if m.pagerFocused {
		m.filePager.LineDown(lines)
		return
	}

	excess := m.GetCurrentChange().MoveCursor(lines)

	for excess != 0 {
		if excess < 0 {
			m.selectedIdx--
		} else {
			m.selectedIdx++
		}

		if m.selectedIdx < 0 {
			m.selectedIdx = 0
			m.GetCurrentChange().MoveCursorToHead()
			m.filePager.SetYOffset(0)
			return
		} else if m.selectedIdx >= len(m.changes) {
			m.selectedIdx = len(m.changes) - 1
			m.GetCurrentChange().MoveCursorToTail()
			m.filePager.SetYOffset(0)
			return
		}

		excess = m.GetCurrentChange().MoveCursor(excess)
	}

	if curLineChange := m.GetCurrentChange().GetFocusedLineChange(); curLineChange != nil {
		m.filePager.SetYOffset(curLineChange.PreviewLineNo - m.filePager.Height/2)
	}
}

func (m *Model) UpdateViewport() {
	m.viewport.SetContent(m.content())

	start := m.viewport.YOffset
	end := start + m.viewport.Height - 1
	cursor := m.GetCursorYOffset()

	if cursor > end {
		m.viewport.LineDown(cursor - end)
	} else if cursor < start {
		m.viewport.LineUp(start - cursor)
	}
}

func (m *Model) UpdateFilePager() {
	m.filePager.SetContent(strings.Join(m.changes[m.selectedIdx].Preview, "\n"))
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+C", "ctrl+Q":
			cmds = append(cmds, tea.Quit)
		case "j":
			m.MoveCursor(1)
		case "k":
			m.MoveCursor(-1)
		case "u":
			m.MoveCursor(10)
		case "i":
			m.MoveCursor(-10)
		case "h":
			m.pagerFocused = false
		case "l":
			m.pagerFocused = true
		case "e":
			// TODO: Use tea.Exec
			m.changes[m.selectedIdx].OpenInEditor()
			return m, tea.ClearScreen
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width/2, msg.Height)
			m.filePager = viewport.New(msg.Width/2, msg.Height)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width / 2
			m.viewport.Height = msg.Height
			m.filePager.Width = msg.Width / 2
			m.filePager.Height = msg.Height
		}
	}

	m.changes[m.selectedIdx], cmd = m.changes[m.selectedIdx].Update(msg)
	cmds = append(cmds, cmd)

	m.UpdateViewport()
	m.UpdateFilePager()

	// m.viewport, cmd = m.viewport.Update(msg)
	// cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	borderColor := lipgloss.Color("#555")
	if m.pagerFocused {
		borderColor = lipgloss.Color("#0000ff")
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(m.viewport.Width).Render(m.viewport.View()),
		lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderColor).
			BorderLeft(true).
			Width(m.filePager.Width).
			Render(m.filePager.View()),
	)
}

func (m Model) Replace() {
	for _, c := range m.changes {
		c.Replace()
	}
}

func (m Model) content() string {
	strs := make([]string, 0, len(m.changes))

	for _, c := range m.changes {
		strs = append(strs, c.View())
	}

	return strings.Join(strs, "\n\n")
}
