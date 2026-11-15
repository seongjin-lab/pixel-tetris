package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const (
	viewWidth         = 47
	viewHeight        = 24
	startButtonWidth  = 19
	startButtonHeight = 3
	startButtonX      = (viewWidth - startButtonWidth) / 2
	startButtonY      = 12
)

var (
	canvasColor = tcell.NewHexColor(0x101222)
	wellColor   = tcell.NewHexColor(0x090b16)
	textColor   = tcell.NewHexColor(0xe4dbfb)
	accentColor = tcell.NewHexColor(0xdd8dff)
	frameColor  = tcell.NewHexColor(0x786799)
	gridColor   = tcell.NewHexColor(0x252840)
	rewardGold  = tcell.NewHexColor(0xffd553)
	blockColors = [...]int32{0x26c6da, 0x6486ff, 0xffa44b, 0xffd553, 0x63d98b, 0xcc7cf4, 0xff6786}
	blockShine  = [...]int32{0xa0f2ff, 0xb4c6ff, 0xffd39b, 0xffefae, 0xb8f9bf, 0xedd0ff, 0xffbdc9}
	ghostFill   = [...]int32{0x102f3b, 0x1a243f, 0x382a20, 0x383321, 0x193227, 0x2c223e, 0x381e2b}
	ghostEdge   = [...]int32{0x25606c, 0x3a4e80, 0x795437, 0x776637, 0x376b50, 0x694778, 0x783f51}
)

type display struct {
	screen tcell.Screen
	x, y   int
}

func draw(screen tcell.Screen, g *game, audio *soundPlayer) {
	width, height := screen.Size()
	screen.SetStyle(tcell.StyleDefault.Background(canvasColor))
	screen.Clear()
	defer screen.Show()

	if width < viewWidth || height < viewHeight {
		message := "터미널을 47열 × 24행 이상으로 넓혀 주세요"
		if width < runewidth.StringWidth(message) {
			message = "47 × 24"
		}
		d := display{screen, max(0, (width-runewidth.StringWidth(message))/2), height / 2}
		d.text(0, 0, message, textColor, canvasColor)
		return
	}

	d := display{screen, (width - viewWidth) / 2, (height - viewHeight) / 2}
	d.text(0, 0, "▰  픽셀 테트리스", accentColor, canvasColor)
	soundLabel := "소리 사용 불가"
	if audio != nil {
		soundLabel = "M  소리 켜짐"
		if audio.cancel == nil {
			soundLabel = "M  소리 꺼짐"
		}
	}
	d.text(viewWidth-runewidth.StringWidth(soundLabel), 0, soundLabel, textColor, canvasColor)

	if g == nil {
		d.startMenu()
		return
	}

	d.frame(0, 2, 22, 22, "")
	d.frame(24, 2, 23, 6, "기록")
	d.frame(24, 9, 23, 6, "다음 블록")
	d.frame(24, 16, 23, 8, "조작")

	for y, row := range g.board {
		for x, id := range row {
			switch {
			case g.clearing[y] && g.clearFrame == 0:
				d.text(1+2*x, 3+y, "▌ ", accentColor, textColor)
			case id == 0 || g.clearing[y] && (x < g.clearFrame || x >= boardWidth-g.clearFrame):
				d.text(1+2*x, 3+y, "· ", gridColor, wellColor)
			case g.clearing[y]:
				d.text(1+2*x, 3+y, "▌ ", textColor, frameColor)
			default:
				d.block(1+2*x, 3+y, id, false)
			}
		}
	}

	if !g.over && g.clearCount == 0 {
		d.drawPiece(g.landing(), true)
		p := g.current
		if g.dropping() {
			offset := min(g.dropDistance-1, int(g.dropOffset(time.Now())))
			if offset > 0 {
				trail := p
				trail.origin.y += max(0, offset-2)
				d.drawPiece(trail, true)
			}
			p.origin.y += offset
		}
		d.drawPiece(p, false)
	}

	for i, stat := range [...]struct {
		label string
		value int
	}{{"점수", g.score}, {"줄", g.lines}, {"레벨", g.level()}} {
		value := strconv.Itoa(stat.value)
		d.text(26, 3+i, stat.label, textColor, canvasColor)
		d.text(45-len(value), 3+i, value, accentColor, canvasColor)
	}

	if reward := g.reward; reward.visible() {
		if reward.comboPoints > 0 {
			d.rewardBanner(8, reward.frame, accentColor, fmt.Sprintf("%d 콤보  +%d", reward.streak, reward.comboPoints))
		}
		if reward.lines > 1 {
			d.rewardBanner(15, reward.frame, rewardGold, fmt.Sprintf("%d줄 동시 삭제  +%d", reward.lines, reward.linePoints))
		}
	}

	next := shapes[g.next-1]
	left, right := next[0].x, next[0].x
	top, bottom := next[0].y, next[0].y
	for _, cell := range next {
		left, right = min(left, cell.x), max(right, cell.x)
		top, bottom = min(top, cell.y), max(bottom, cell.y)
	}
	for _, cell := range next {
		d.block(35-(right-left+1)+(cell.x-left)*2, 10+(4-(bottom-top+1))/2+cell.y-top, g.next, false)
	}

	for i, control := range [...][2]string{
		{"← →", "이동"}, {"↑", "회전"}, {"↓", "내리기"},
		{"스페이스", "즉시 낙하"}, {"P / R", "정지·재시작"}, {"Q", "종료"},
	} {
		d.text(26, 17+i, control[0], accentColor, canvasColor)
		d.text(35, 17+i, control[1], textColor, canvasColor)
	}

	switch {
	case g.over:
		d.banner("게임 종료", "R  다시 시작")
	case g.paused:
		d.banner("일시정지", "P  계속하기")
	}
}

func (d display) rewardBanner(y, frame int, color tcell.Color, label string) {
	const x, width = 24, 23
	fg, bg := color, wellColor
	if frame == 0 || frame == 2 {
		fg, bg = canvasColor, color
	}
	d.text(x, y, strings.Repeat(" ", width), fg, bg)
	d.text(x+(width-runewidth.StringWidth(label))/2, y, label, fg, bg)
	if frame < 4 {
		d.text(x, y, "▪", fg, bg)
		d.text(x+width-1, y, "▪", fg, bg)
	}
}

func (d display) startMenu() {
	d.frame(7, 5, 33, 15, "")
	for _, cell := range shapes[5] {
		d.block(20+2*cell.x, 8+cell.y, 6, false)
	}
	for y := 0; y < startButtonHeight; y++ {
		d.text(startButtonX, startButtonY+y, strings.Repeat(" ", startButtonWidth), canvasColor, accentColor)
	}

	label := "게임 시작"
	d.text((viewWidth-runewidth.StringWidth(label))/2, startButtonY+1, label, canvasColor, accentColor)
	hint := "Enter 시작 · Q 종료"
	d.text((viewWidth-runewidth.StringWidth(hint))/2, 17, hint, textColor, canvasColor)
}

func (d display) frame(x, y, width, height int, title string) {
	line := strings.Repeat("─", width-2)
	d.text(x, y, "┌"+line+"┐", frameColor, canvasColor)
	d.text(x, y+height-1, "└"+line+"┘", frameColor, canvasColor)
	for row := y + 1; row < y+height-1; row++ {
		d.text(x, row, "│", frameColor, canvasColor)
		d.text(x+width-1, row, "│", frameColor, canvasColor)
	}
	if title != "" {
		d.text(x+2, y, " "+title+" ", accentColor, canvasColor)
	}
}

func (d display) drawPiece(p piece, ghost bool) {
	for _, cell := range p.cells {
		x, y := p.origin.x+cell.x, p.origin.y+cell.y
		if y < 0 {
			continue
		}
		d.block(1+2*x, 3+y, p.id, ghost)
	}
}

func (d display) block(x, y, id int, ghost bool) {
	edge, fill := blockShine[id-1], blockColors[id-1]
	if ghost {
		edge, fill = ghostEdge[id-1], ghostFill[id-1]
	}
	d.text(x, y, "▌ ", tcell.NewHexColor(edge), tcell.NewHexColor(fill))
}

func (d display) banner(title, hint string) {
	d.text(2, 10, strings.Repeat("▀", 18), frameColor, wellColor)
	for y := 11; y <= 13; y++ {
		d.text(2, y, strings.Repeat(" ", 18), textColor, canvasColor)
	}
	d.text(2, 14, strings.Repeat("▄", 18), frameColor, wellColor)
	d.text((22-runewidth.StringWidth(title))/2, 11, title, accentColor, canvasColor)
	d.text((22-runewidth.StringWidth(hint))/2, 13, hint, textColor, canvasColor)
}

func (d display) text(x, y int, value string, fg, bg tcell.Color) {
	style := tcell.StyleDefault.Foreground(fg).Background(bg)
	for _, char := range value {
		d.screen.SetContent(d.x+x, d.y+y, char, nil, style)
		x += runewidth.RuneWidth(char)
	}
}
