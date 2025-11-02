package tui

import (
	"database/sql"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"tudo/database"
)

const (
	_      = iota
	NORMAL = iota
	COMMAND
)

type cellbuffer struct {
	cells  []string
	stride int
}

func (c *cellbuffer) init(w, h int) {
	if w == 0 || h == 0 {
		return
	}
	c.stride = w
	c.cells = make([]string, w*h)
	c.wipe()
}

func (c *cellbuffer) ready() bool {
	return len(c.cells) > 0
}

func (c *cellbuffer) wipe() {
	for i := range c.cells {
		c.cells[i] = " "
	}
}

type model struct {
	cells cellbuffer

	Db *sql.DB

	State     int
	PrevState int

	Cmd string
}

func (m model) Init() tea.Cmd {
	tea.ClearScreen()
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch input := msg.String(); input {
		case ":":
			if m.State != COMMAND {
				m.PrevState = m.State
				m.State = COMMAND
			}
		case "ctrl+u":
			if m.State == COMMAND {
				m.Cmd = ""
			}

		case "enter":
			if m.State == COMMAND {
				if len(m.Cmd) > 0 {
					switch m.Cmd {
					case "q":
						return m, tea.Quit
					}
				} else {
					m.State = m.PrevState
				}
			}
		case "backspace":
			if m.State == COMMAND {
				if len(m.Cmd) > 0 {
					m.Cmd = m.Cmd[:len(m.Cmd)-1]
				} else {
					m.State = m.PrevState
				}
			}

		case "ctrl+c", "esc":
			if m.State == COMMAND {
				m.State = m.PrevState
			}

		case "h":
			if m.State != COMMAND {

			}
		case "j":
			if m.State != COMMAND {

			}
		case "k":
			if m.State != COMMAND {

			}
		case "l":
			if m.State != COMMAND {

			}

		default:
			if m.State == COMMAND {
				m.Cmd += input
			}
		}
	case tea.WindowSizeMsg:
		if !m.cells.ready() {
			m.cells.init(msg.Width, msg.Height)
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.State {
	case NORMAL:
		return "normal"
	case COMMAND:
		return ":" + m.Cmd
	default:
		return "tudo"
	}
}

func initModel(dbFile string) model {
	db, err := database.Connect(dbFile)
	if err != nil {
		fmt.Println("Could not connect to database :" + err.Error())
		os.Exit(1)
	}
	defer db.Close()

	return model{
		Db:    db,
		State: NORMAL,
	}
}

func Start(dbFile string) {
	p := tea.NewProgram(initModel(dbFile), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("ERROR: could not run bubbletea program \n" + err.Error())
		os.Exit(1)
	}
}
