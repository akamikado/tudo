package tui

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

const (
	_      = iota
	normal = iota - 1
	command
)

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

func drawTudoWindow(s tcell.Screen, style tcell.Style) {
	w, h := s.Size()
	drawRect(s, style, 0, 0, w, h)
	drawHorizontalLine(s, style, 1, h-2, w-2)
	s.SetContent(0, h-2, tcell.RuneLTee, nil, style)
	s.SetContent(w-1, h-2, tcell.RuneRTee, nil, style)
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
		switch state {
		case normal:
		case command:
		default:
		}
		s.Show()

		ev := s.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
			s.Clear()
			drawTudoWindow(s, defStyle)

		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				quit()

			default:
				switch ev.Rune() {
				case ':':
					if state != command {
						state = command
					}
				}
			}
		}
	}
}
