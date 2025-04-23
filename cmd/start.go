package cmd

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type Script struct {
	Name     string
	Callback func()
}

type start struct {
	scripts  []Script         // list of scripts
	selected map[int]struct{} // which Script is selected
	cursor   int              // which Script is the cursor pointing at
}

// cmdStart initialize starting command for all scripts.
func cmdStart() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start script",
		RunE: func(cmd *cobra.Command, args []string) error {

			cli := start{
				scripts:  make([]Script, 0),
				selected: make(map[int]struct{}),
				cursor:   0,
			}

			// default to select first item
			cli.selected[cli.cursor] = struct{}{}

			p := tea.NewProgram(cli)

			if _, err := p.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func (m start) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m start) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Check for key press
	case tea.KeyMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
			m.selected = make(map[int]struct{})
			m.selected[m.cursor] = struct{}{}

		case "down":
			if m.cursor < len(m.scripts)-1 {
				m.cursor++
			}
			m.selected = make(map[int]struct{})
			m.selected[m.cursor] = struct{}{}

		case "enter", " ":
			// tea.Printf("\n\nRunning script...\n\n")

			// execute function
			m.scripts[m.cursor].Callback()

			// tea.Printf("\n\nDONE\n\n")
			return m, tea.Quit
		}
	}

	// Return the updated start to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m start) View() string {
	// The header
	s := "\nSelect available script\n"

	// Iterate over our scripts
	for i, choice := range m.scripts {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s (%s) %s\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}
