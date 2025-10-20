package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
)

const (
	cmdLineAreaHeight = 3

	_      = iota
	normal = iota - 1
	command
)

var cmd []byte

func fatalError(err error, args ...string) {
	fmt.Println("Error occurred!!")
	e := err.Error()
	for _, arg := range args {
		e += arg + " "
	}
	fmt.Println(e)
	os.Exit(1)
}

func quit() {
	os.Exit(0)
}

func drawVerticalLine(s tcell.Screen, style tcell.Style, x, y, length int) {
	for i := y; i < y+length; i = i + 1 {
		s.SetContent(x, i, tcell.RuneVLine, nil, style)
	}
}

func drawHorizontalLine(s tcell.Screen, style tcell.Style, x, y, length int) {
	for i := x; i < x+length; i = i + 1 {
		s.SetContent(i, y, tcell.RuneHLine, nil, style)
	}
}

func drawRect(s tcell.Screen, style tcell.Style, x, y, w, h int) {
	drawHorizontalLine(s, style, x+1, y, w-2)
	drawHorizontalLine(s, style, x+1, y+h-1, w-2)
	drawVerticalLine(s, style, x, y+1, h-2)
	drawVerticalLine(s, style, x+w-1, y+1, h-2)

	s.SetContent(x, y, tcell.RuneULCorner, nil, style)
	s.SetContent(x+w-1, y, tcell.RuneURCorner, nil, style)
	s.SetContent(x, y+h-1, tcell.RuneLLCorner, nil, style)
	s.SetContent(x+w-1, y+h-1, tcell.RuneLRCorner, nil, style)
}

func drawText(s tcell.Screen, style tcell.Style, x, y int, r rune) {
	s.SetContent(x, y, r, nil, style)
}

func drawTudoWindow(s tcell.Screen, style tcell.Style) {
	w, h := s.Size()
	drawRect(s, style, 0, 0, w, h)
	drawHorizontalLine(s, style, 1, h-cmdLineAreaHeight, w-2)
	s.SetContent(0, h-cmdLineAreaHeight, tcell.RuneLTee, nil, style)
	s.SetContent(w-1, h-cmdLineAreaHeight, tcell.RuneRTee, nil, style)
}

func writeToCommand(s tcell.Screen, style tcell.Style, r rune) {
	w, h := s.Size()
	if w-2 > 2+len(cmd) {
		drawText(s, style, 2+len(cmd), h-cmdLineAreaHeight+1, r)
		cmd = append(cmd, byte(r))
	}
}

func popFromCommand(s tcell.Screen, style tcell.Style) {
	_, h := s.Size()
	if len(cmd) > 0 {
		cmd = cmd[:len(cmd)-1]
		drawText(s, style, 2+len(cmd), h-cmdLineAreaHeight+1, ' ')
	}
}

func clearCommand(s tcell.Screen, style tcell.Style) {
	cmd = []byte{}
	w, h := s.Size()
	for i := 2; i < w-1; i += 1 {
		drawText(s, style, i, h-cmdLineAreaHeight+1, ' ')
	}
}

func notify(s tcell.Screen, style tcell.Style, msg string) {
	cmd = []byte{}
	_, h := s.Size()
	for i, c := range []byte(msg) {
		drawText(s, style, 1+i, h-cmdLineAreaHeight+1, rune(c))
	}
}

func Start() {
	s, err := tcell.NewScreen()
	if err != nil {
		fatalError(err)
	}
	if err := s.Init(); err != nil {
		fatalError(err)
	}

	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	s.SetStyle(defStyle)

	s.Clear()

	state := normal

	drawTudoWindow(s, defStyle)

	for {
		s.Show()

		ev := s.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventKey:
			if k := ev.Key(); k == tcell.KeyBackspace || k == tcell.KeyBackspace2 {
				switch state {
				case command:
					popFromCommand(s, defStyle)
				}
			} else if k == tcell.KeyEnter {
				if state == command {
					cmds := strings.Split(string(cmd), " ")
					if len(cmds) > 0 {
						switch cmds[0] {
						case "quit":
							fallthrough
						case "q":
							quit()
						default:
							notify(s, defStyle, "unknown command")
							state = normal
						}
					}
				}
			} else if k == tcell.KeyCtrlU {
				if state == command {
					clearCommand(s, defStyle)
				}
			} else {
				if r := ev.Rune(); r == ':' {
					if state != command {
						state = command
						_, h := s.Size()
						clearCommand(s, defStyle)
						drawText(s, defStyle, 1, h-cmdLineAreaHeight+1, ':')
					}
				} else {
					if state == command {
						writeToCommand(s, defStyle, r)
					}
				}
			}
		}
	}
}
