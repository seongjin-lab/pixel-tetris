package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

type windowScreen struct {
	tcell.SimulationScreen
}

type windowCell struct {
	Index int    `json:"i"`
	Text  string `json:"t,omitempty"`
	Width int    `json:"w,omitempty"`
	FG    int32  `json:"f"`
	BG    int32  `json:"b"`
}

type windowFrame struct {
	Width      int          `json:"width"`
	Height     int          `json:"height"`
	Background int32        `json:"background"`
	Cells      []windowCell `json:"cells"`
}

func (s *windowScreen) Init() error {
	if err := s.SimulationScreen.Init(); err != nil {
		return err
	}
	s.SetSize(viewWidth, viewHeight)
	go s.readInput()
	return nil
}

func (s *windowScreen) Show() {
	w, h := s.Size()
	background := canvasColor.Hex()
	frame := windowFrame{Width: w, Height: h, Background: background, Cells: make([]windowCell, 0, w*h/2)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; {
			r, extra, style, width := s.GetContent(x, y)
			fg, bg, _ := style.Decompose()
			fill := bg.Hex()
			if fill < 0 {
				fill = background
			}
			width = max(1, width)
			cell := windowCell{Index: y*w + x, Width: width, FG: fg.Hex(), BG: fill}
			if r != 0 && r != ' ' {
				cell.Text = string(r) + string(extra)
			}
			if cell.Text != "" || fill != background {
				frame.Cells = append(frame.Cells, cell)
			}
			x += width
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(frame)
}

func (s *windowScreen) readInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		var key tcell.Key
		switch line {
		case "enter":
			key = tcell.KeyEnter
		case "escape":
			key = tcell.KeyEscape
		case "up":
			key = tcell.KeyUp
		case "down":
			key = tcell.KeyDown
		case "left":
			key = tcell.KeyLeft
		case "right":
			key = tcell.KeyRight
		default:
			var x, y int
			if n, _ := fmt.Sscanf(line, "click %d %d", &x, &y); n == 2 {
				s.PostEventWait(tcell.NewEventMouse(x, y, tcell.Button1, tcell.ModNone))
			} else if len(line) == 1 {
				s.PostEventWait(tcell.NewEventKey(tcell.KeyRune, rune(line[0]), tcell.ModNone))
			}
			continue
		}
		s.PostEventWait(tcell.NewEventKey(key, 0, tcell.ModNone))
	}
	s.PostEventWait(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
}
