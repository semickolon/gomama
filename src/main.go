package gomama

import (
	"fmt"
	"os"

	"github.com/dlclark/regexp2"
	changelist "github.com/semickolon/gomama/src/changelist"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModelArgs struct {
	regex       *regexp2.Regexp
	subst       *string
	filenames   []string
	infoTitle   string
	infoMessage string
	w           int
	h           int
}

type Model struct {
	args        ModelArgs
	changeList  changelist.Model
	showInfoBox bool
}

func New(args ModelArgs) (*Model, error) {
	files := make([]*os.File, 0, len(args.filenames))

	for _, filename := range args.filenames {
		file, err := os.OpenFile(filename, os.O_RDWR, os.ModePerm)

		if err != nil {
			return nil, err
		}

		files = append(files, file)
	}

	changeList, err := changelist.New(files, args.regex, args.subst)
	if err != nil {
		return nil, err
	}

	if changeList == nil {
		return nil, nil
	}

	return &Model{args, *changeList, false}, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	args := m.args

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.args.w = msg.Width
		m.args.h = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "i":
			if m.args.infoMessage != "" {
				m.showInfoBox = !m.showInfoBox
			}
		case "r":
			m.changeList.Replace()

			if args.subst == nil {
				return m, tea.Quit // we are already in review mode, exit
			} else {
				args.subst = nil // review mode time
			}
			fallthrough
		case "v":
			newModel, err := New(args)

			if err != nil {
				panic(err)
			} else if newModel == nil {
				return m, tea.Quit
			}

			return newModel, func() tea.Msg { // hack to trigger redraw
				return tea.WindowSizeMsg{Width: m.args.w, Height: m.args.h}
			}
		}
	}

	if !m.showInfoBox {
		m.changeList, cmd = m.changeList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.showInfoBox {
		infoText := m.args.infoMessage

		if m.args.infoTitle != "" {
			infoText = fmt.Sprintf("%s\n%s", lipgloss.NewStyle().Bold(true).Render(m.args.infoTitle), infoText)
		}

		info := lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#ebbcba")).
			PaddingLeft(1).
			PaddingRight(1).
			Width(m.args.w - 32).
			Render(infoText)

		return lipgloss.Place(m.args.w, m.args.h, lipgloss.Center, lipgloss.Center, info,
			lipgloss.WithWhitespaceChars("i "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("#1f1d2e")))
	} else {
		return m.changeList.View()
	}
}

func Run(regexStr string, subst *string, filenames []string, infoTitle string, infoMessage string) error {
	regex, err := regexp2.Compile(regexStr, 0)
	if err != nil {
		return err
	}

	m, err := New(ModelArgs{regex, subst, filenames, infoTitle, infoMessage, 0, 0})

	if err != nil {
		return err
	} else if m == nil {
		fmt.Println("No matches found")
		return nil
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
