package main

import (
	"math/rand/v2"
	"time"
)

const (
	boardWidth   = 10
	boardHeight  = 20
	clearTick    = 45 * time.Millisecond
	dropTick     = 16 * time.Millisecond
	dropDuration = 105 * time.Millisecond
	rewardTick   = 140 * time.Millisecond
	rewardFrames = 6
)

type point struct{ x, y int }

type piece struct {
	id     int
	origin point
	cells  [4]point
}

type clearReward struct {
	lines       int
	linePoints  int
	streak      int
	comboPoints int
	frame       int
}

func (r clearReward) visible() bool {
	return r.frame < rewardFrames && (r.lines > 1 || r.comboPoints > 0)
}

var shapes = [7][4]point{
	{{0, 1}, {1, 1}, {2, 1}, {3, 1}},
	{{0, 0}, {0, 1}, {1, 1}, {2, 1}},
	{{2, 0}, {0, 1}, {1, 1}, {2, 1}},
	{{1, 0}, {2, 0}, {1, 1}, {2, 1}},
	{{1, 0}, {2, 0}, {0, 1}, {1, 1}},
	{{1, 0}, {0, 1}, {1, 1}, {2, 1}},
	{{0, 0}, {1, 0}, {1, 1}, {2, 1}},
}

type game struct {
	board        [boardHeight][boardWidth]int
	current      piece
	next         int
	bag          []int
	score        int
	lines        int
	streak       int
	reward       clearReward
	paused       bool
	over         bool
	clearing     [boardHeight]bool
	clearCount   int
	clearFrame   int
	dropStarted  time.Time
	dropDistance int
}

func newGame() *game {
	g := &game{}
	g.next = g.drawPiece()
	g.spawn()
	return g
}

func (g *game) drawPiece() int {
	if len(g.bag) == 0 {
		g.bag = rand.Perm(len(shapes))
	}
	id := g.bag[0] + 1
	g.bag = g.bag[1:]
	return id
}

func (g *game) spawn() {
	g.current = piece{g.next, point{3, 0}, shapes[g.next-1]}
	g.next = g.drawPiece()
	g.over = !g.fits(g.current)
}

func (g *game) fits(p piece) bool {
	for _, cell := range p.cells {
		x, y := p.origin.x+cell.x, p.origin.y+cell.y
		if x < 0 || x >= boardWidth || y >= boardHeight {
			return false
		}
		if y >= 0 && g.board[y][x] != 0 {
			return false
		}
	}
	return true
}

func (g *game) move(dx, dy int) bool {
	p := g.current
	p.origin.x += dx
	p.origin.y += dy
	if !g.fits(p) {
		return false
	}
	g.current = p
	return true
}

func (g *game) rotate() bool {
	p := g.current
	if p.id == 4 {
		return false
	}
	last := 2
	if p.id == 1 {
		last = 3
	}
	for i, cell := range p.cells {
		p.cells[i] = point{last - cell.y, cell.x}
	}

	// 벽과 바닥에서 회전할 공간을 가까운 칸부터 찾는다.
	for _, kick := range [...]point{{0, 0}, {-1, 0}, {1, 0}, {-2, 0}, {2, 0}, {0, -1}, {0, -2}} {
		p.origin = point{g.current.origin.x + kick.x, g.current.origin.y + kick.y}
		if g.fits(p) {
			g.current = p
			return true
		}
	}
	return false
}

func (g *game) landing() piece {
	p := g.current
	p.origin.y++
	for g.fits(p) {
		p.origin.y++
	}
	p.origin.y--
	return p
}

func (g *game) drop() {
	p := g.landing()
	g.score += 2 * (p.origin.y - g.current.origin.y)
	g.current = p
	g.lock()
}

func (g *game) beginDrop(now time.Time) bool {
	g.dropDistance = g.landing().origin.y - g.current.origin.y
	if g.dropDistance == 0 {
		g.drop()
		return true
	}
	g.dropStarted = now
	return false
}

func (g *game) dropping() bool {
	return !g.dropStarted.IsZero()
}

func (g *game) dropOffset(now time.Time) float64 {
	progress := min(1, max(0, float64(now.Sub(g.dropStarted))/float64(dropDuration)))
	return float64(g.dropDistance) * progress * progress
}

func (g *game) advanceDrop(now time.Time) bool {
	if now.Sub(g.dropStarted) < dropDuration {
		return false
	}
	g.finishDrop()
	return true
}

func (g *game) finishDrop() {
	g.dropStarted = time.Time{}
	g.drop()
}

func (g *game) step(soft bool) bool {
	if !g.move(0, 1) {
		g.lock()
		return true
	}
	if soft {
		g.score++
	}
	return false
}

func (g *game) lock() {
	for _, cell := range g.current.cells {
		if g.current.origin.y+cell.y < 0 {
			g.over = true
			return
		}
	}
	for _, cell := range g.current.cells {
		p := g.current.origin
		g.board[p.y+cell.y][p.x+cell.x] = g.current.id
	}

	for y, row := range g.board {
		full := true
		for _, id := range row {
			if id == 0 {
				full = false
				break
			}
		}
		if full {
			g.clearing[y] = true
			g.clearCount++
		}
	}
	if g.clearCount == 0 {
		g.streak = 0
		g.spawn()
	}
}

func (g *game) advanceClear() bool {
	g.clearFrame++
	if g.clearFrame <= boardWidth/2 {
		return false
	}

	write := boardHeight - 1
	for read := boardHeight - 1; read >= 0; read-- {
		if !g.clearing[read] {
			g.board[write] = g.board[read]
			write--
		}
	}
	for ; write >= 0; write-- {
		g.board[write] = [boardWidth]int{}
	}
	g.streak++
	level := g.level()
	g.reward = clearReward{
		lines:       g.clearCount,
		linePoints:  [...]int{0, 100, 300, 500, 800}[g.clearCount] * level,
		streak:      g.streak,
		comboPoints: 50 * (g.streak - 1) * level,
	}
	g.score += g.reward.linePoints + g.reward.comboPoints
	g.lines += g.clearCount
	g.clearing = [boardHeight]bool{}
	g.clearCount, g.clearFrame = 0, 0
	g.spawn()
	return true
}

func (g *game) level() int {
	return 1 + g.lines/10
}

func (g *game) interval() time.Duration {
	if g.dropping() {
		return dropTick
	}
	if g.clearCount != 0 {
		return clearTick
	}
	return time.Duration(max(85, 800-65*(g.level()-1))) * time.Millisecond
}
