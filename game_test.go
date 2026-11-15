package main

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func TestHardDropClearsFourLines(t *testing.T) {
	g := newGame()
	for y := 16; y < boardHeight; y++ {
		for x := range g.board[y] {
			if x != 5 {
				g.board[y][x] = 2
			}
		}
	}
	g.board[15][0] = 3
	g.current = piece{1, point{5, 0}, [4]point{{0, 0}, {0, 1}, {0, 2}, {0, 3}}}
	next := g.next
	g.drop()
	if g.clearCount != 4 || g.lines != 0 || g.score != 32 || g.board[19][5] != 1 {
		t.Fatal("점수와 줄은 애니메이션이 끝날 때까지 반영하지 않아야 합니다")
	}

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(viewWidth, viewHeight)
	draw(screen, g, nil)
	_, _, flash, _ := screen.GetContent(1, 22)
	if _, background, _ := flash.Decompose(); background != textColor {
		t.Fatal("채운 줄의 첫 화면에 반짝임이 표시되지 않았습니다")
	}

	for frame := 1; frame <= boardWidth/2; frame++ {
		g.advanceClear()
		if g.clearFrame != frame || g.clearCount != 4 || g.next != next {
			t.Fatalf("애니메이션 %d번째 화면에서 다음 블록이 너무 일찍 나왔습니다", frame)
		}
		draw(screen, g, nil)
		left, _, _, _ := screen.GetContent(2*frame-1, 22)
		right, _, _, _ := screen.GetContent(2*(boardWidth-frame)+1, 22)
		if left != '·' || right != '·' {
			t.Fatalf("애니메이션 %d번째 화면에서 양옆의 블록이 사라지지 않았습니다", frame)
		}
	}
	g.advanceClear()
	if g.lines != 4 || g.score != 832 || g.board[19][0] != 3 || g.board[15] != ([boardWidth]int{}) {
		t.Fatalf("줄 삭제 후 상태가 다릅니다 (줄 %d, 점수 %d, 마지막 줄 %v)", g.lines, g.score, g.board[19])
	}
	if g.current.id != next || g.over {
		t.Fatal("다음 블록이 정상적으로 나오지 않았습니다")
	}
}

func TestRotationAtWall(t *testing.T) {
	g := newGame()
	g.current = piece{1, point{7, 2}, [4]point{{2, 0}, {2, 1}, {2, 2}, {2, 3}}}
	g.rotate()
	if g.current.origin.x != 6 || !g.fits(g.current) {
		t.Fatalf("오른쪽 벽에서 회전할 공간을 찾지 못했습니다 (%+v)", g.current)
	}
	if g.move(1, 0) {
		t.Fatal("블록이 벽을 통과했습니다")
	}
}

func TestComboBonusAndMultiLineReward(t *testing.T) {
	g := newGame()
	g.lines = 8
	dropForLines := func(lines int) {
		g.board = [boardHeight][boardWidth]int{}
		for y := boardHeight - lines; y < boardHeight; y++ {
			for x := range g.board[y] {
				if x != 5 {
					g.board[y][x] = 2
				}
			}
		}
		g.current = piece{1, point{5, boardHeight - 4}, [4]point{{0, 0}, {0, 1}, {0, 2}, {0, 3}}}
		g.drop()
		for g.clearCount > 0 {
			g.advanceClear()
		}
	}

	dropForLines(1)
	if g.score != 100 || g.streak != 1 || g.reward.visible() {
		t.Fatalf("첫 줄 삭제에 콤보가 적용되었습니다 (점수 %d, 연속 %d)", g.score, g.streak)
	}

	dropForLines(2)
	if g.score != 450 || g.level() != 2 || g.reward != (clearReward{lines: 2, linePoints: 300, streak: 2, comboPoints: 50}) {
		t.Fatalf("두 줄 삭제와 콤보 점수가 다릅니다 (점수 %d, 연출 %+v)", g.score, g.reward)
	}

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(viewWidth, viewHeight)
	draw(screen, g, nil)
	_, _, blank, _ := screen.GetContent(23, 8)
	_, blankBG, _ := blank.Decompose()
	for _, phase := range []struct {
		frame        int
		combo, multi tcell.Color
	}{{0, accentColor, rewardGold}, {1, wellColor, wellColor}, {2, accentColor, rewardGold}, {rewardFrames, blankBG, blankBG}} {
		g.reward.frame = phase.frame
		draw(screen, g, nil)
		_, _, combo, _ := screen.GetContent(24, 8)
		_, _, multi, _ := screen.GetContent(24, 15)
		_, comboBG, _ := combo.Decompose()
		_, multiBG, _ := multi.Decompose()
		if comboBG != phase.combo || multiBG != phase.multi {
			t.Fatalf("연출의 %d번째 화면에 표시된 색상이 다릅니다", phase.frame)
		}
	}

	dropForLines(1)
	if g.score != 850 || g.reward.comboPoints != 200 || g.streak != 3 {
		t.Fatalf("레벨 2의 세 번째 콤보 점수가 다릅니다 (점수 %d, 연출 %+v)", g.score, g.reward)
	}

	dropForLines(0)
	dropForLines(1)
	if g.score != 1050 || g.streak != 1 || g.reward.visible() {
		t.Fatalf("줄을 지우지 못한 뒤 콤보가 초기화되지 않았습니다 (점수 %d, 연속 %d)", g.score, g.streak)
	}
}

func TestGhostUsesSameBlockShape(t *testing.T) {
	g := newGame()
	g.current = piece{4, point{3, 8}, shapes[3]}
	landing := g.landing()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(viewWidth, viewHeight)
	draw(screen, g, nil)

	for _, cell := range g.current.cells {
		for column := range 2 {
			x := 1 + 2*(g.current.origin.x+cell.x) + column
			currentRow := 3 + g.current.origin.y + cell.y
			landingRow := 3 + landing.origin.y + cell.y
			active, _, activeStyle, _ := screen.GetContent(x, currentRow)
			ghost, _, ghostStyle, _ := screen.GetContent(x, landingRow)
			_, ghostBackground, _ := ghostStyle.Decompose()
			if active != ghost || activeStyle == ghostStyle || ghostBackground != tcell.NewHexColor(ghostFill[3]) {
				t.Fatalf("네모 블록의 착지 미리보기에 다른 모양이나 색상이 표시되었습니다 (열 %d, 행 %d)", x, landingRow)
			}
		}
	}
}

func TestAnimatedDropScoresOnlyOnLanding(t *testing.T) {
	g := newGame()
	g.current = piece{4, point{3, 0}, shapes[3]}
	start := time.Unix(1, 0)
	landing := g.landing()
	distance := landing.origin.y - g.current.origin.y
	if g.beginDrop(start) || !g.dropping() || g.interval() != dropTick {
		t.Fatal("즉시 낙하의 애니메이션이 시작되지 않았습니다")
	}
	half := start.Add(dropDuration / 2)
	if offset := g.dropOffset(half); offset != float64(distance)/4 {
		t.Fatalf("낙하 중간 위치가 다릅니다 (%f)", offset)
	}
	if g.advanceDrop(half) || g.score != 0 || g.board[boardHeight-1][4] != 0 {
		t.Fatal("착지 전에 점수 또는 보드가 바뀌었습니다")
	}
	if !g.advanceDrop(start.Add(dropDuration)) || g.dropping() {
		t.Fatal("애니메이션이 끝난 뒤에도 블록이 고정되지 않았습니다")
	}
	if g.score != 2*distance || g.board[boardHeight-1][4] != 4 {
		t.Fatalf("착지 점수 또는 블록 위치가 다릅니다 (점수 %d)", g.score)
	}
}
