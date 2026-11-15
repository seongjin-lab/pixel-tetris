package main

import "C"

import (
	"encoding/json"
	"time"
)

var mobileGame *game
var mobileFall, mobileReward time.Time

type mobileFrame struct {
	Cells       [boardHeight * boardWidth]int `json:"cells"`
	Next        []int                         `json:"next"`
	Score       int                           `json:"score"`
	Lines       int                           `json:"lines"`
	Level       int                           `json:"level"`
	Paused      bool                          `json:"paused"`
	Over        bool                          `json:"over"`
	Combo       int                           `json:"combo"`
	ComboPoints int                           `json:"comboPoints"`
	Multi       int                           `json:"multi"`
	LinePoints  int                           `json:"linePoints"`
	Bright      bool                          `json:"bright"`
	Sounds      []string                      `json:"sounds"`
	Drop        *mobileDrop                   `json:"drop,omitempty"`
}

type mobileDrop struct {
	ID     int       `json:"id"`
	Cells  [4][2]int `json:"cells"`
	Offset float64   `json:"offset"`
}

//export PTGame
func PTGame(input *C.char) *C.char {
	action := C.GoString(input)
	now := time.Now()
	if action == "start" || action == "restart" {
		mobileGame = newGame()
		mobileFall = now.Add(mobileGame.interval())
		mobileReward = time.Time{}
	}
	g := mobileGame
	if g == nil {
		return nil
	}

	var sounds []string
	locked := false
	switch action {
	case "pause", "background":
		if g.over || (action == "background" && g.paused) {
			return nil
		}
		if g.dropping() {
			g.finishDrop()
			locked = true
		}
		g.paused = !g.paused && !g.over
		mobileFall = now.Add(g.interval())
	case "tick":
		redraw := false
		if g.reward.visible() && !now.Before(mobileReward) {
			g.reward.frame++
			mobileReward = now.Add(rewardTick)
			redraw = true
		}
		if !g.paused && !g.over && g.dropping() {
			redraw = true
			locked = g.advanceDrop(now)
		} else if !g.paused && !g.over && !now.Before(mobileFall) {
			redraw = true
			if g.clearCount > 0 {
				if g.advanceClear() && g.reward.visible() {
					mobileReward = now.Add(rewardTick)
					if g.reward.comboPoints > 0 {
						sounds = append(sounds, "combo")
					}
				}
				if g.over {
					sounds = append(sounds, "over")
				}
			} else {
				locked = g.step(false)
			}
			mobileFall = now.Add(g.interval())
		}
		if !redraw {
			return nil
		}
	case "start", "restart":
	default:
		if g.paused || g.over || g.clearCount != 0 || g.dropping() {
			return nil
		}
		switch action {
		case "left":
			g.move(-1, 0)
		case "right":
			g.move(1, 0)
		case "rotate":
			if g.rotate() {
				sounds = append(sounds, "rotate")
			}
		case "down":
			if !g.move(0, 1) {
				return nil
			}
			g.score++
		case "drop":
			locked = g.beginDrop(now)
		default:
			return nil
		}
	}
	if locked {
		mobileFall = now.Add(g.interval())
		switch {
		case g.over:
			sounds = append(sounds, "over")
		case g.clearCount > 1:
			sounds = append(sounds, "multi")
		case g.clearCount == 1:
			sounds = append(sounds, "clear")
		default:
			sounds = append(sounds, "lock")
		}
	}

	data, err := json.Marshal(g.mobileFrame(sounds, now))
	if err != nil {
		return nil
	}
	return C.CString(string(data))
}

func (g *game) mobileFrame(sounds []string, now time.Time) mobileFrame {
	frame := mobileFrame{
		Score: g.score, Lines: g.lines, Level: g.level(), Paused: g.paused, Over: g.over,
		Next: []int{g.next}, Sounds: sounds,
	}
	if reward := g.reward; reward.visible() {
		frame.Combo, frame.ComboPoints = reward.streak, reward.comboPoints
		frame.Multi, frame.LinePoints = reward.lines, reward.linePoints
		frame.Bright = reward.frame == 0 || reward.frame == 2
	}
	for _, cell := range shapes[g.next-1] {
		frame.Next = append(frame.Next, cell.x, cell.y)
	}
	for y, row := range g.board {
		for x, id := range row {
			if g.clearing[y] {
				switch {
				case g.clearFrame == 0:
					id = 8
				case x < g.clearFrame || x >= boardWidth-g.clearFrame:
					id = 0
				default:
					id = 9
				}
			}
			frame.Cells[y*boardWidth+x] = id
		}
	}
	if !g.over && g.clearCount == 0 {
		for i, p := range [...]piece{g.landing(), g.current} {
			id := p.id
			if i == 0 {
				id = -id
			}
			var drop *mobileDrop
			if i == 1 && g.dropping() {
				drop = &mobileDrop{ID: id, Offset: g.dropOffset(now)}
				frame.Drop = drop
			}
			for index, cell := range p.cells {
				x, y := p.origin.x+cell.x, p.origin.y+cell.y
				if drop != nil {
					drop.Cells[index] = [2]int{x, y}
				} else if y >= 0 {
					frame.Cells[y*boardWidth+x] = id
				}
			}
		}
	}
	return frame
}

func main() {}
