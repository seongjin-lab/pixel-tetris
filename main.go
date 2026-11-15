package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "터미널을 열 수 없습니다 (%v)\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) == 2 && os.Args[1] == "--window" {
		return runScreen(&windowScreen{
			SimulationScreen: tcell.NewSimulationScreen("UTF-8"),
		})
	}
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	return runScreen(screen)
}

func runScreen(screen tcell.Screen) error {
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()
	screen.HideCursor()
	screen.EnableMouse(tcell.MouseButtonEvents)

	audio := newSoundPlayer()
	defer audio.close()

	events := make(chan tcell.Event)
	done := make(chan struct{})
	defer close(done)
	go screen.ChannelEvents(events, done)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)

	var g *game
	fall := time.NewTimer(time.Hour)
	fall.Stop()
	defer fall.Stop()
	rewardTimer := time.NewTimer(time.Hour)
	rewardTimer.Stop()
	defer rewardTimer.Stop()

	for {
		draw(screen, g, audio)
		select {
		case <-signals:
			return nil
		case <-rewardTimer.C:
			g.reward.frame++
			if g.reward.visible() {
				rewardTimer.Reset(rewardTick)
			}
		case <-fall.C:
			width, height := screen.Size()
			if width >= viewWidth && height >= viewHeight && !g.paused && !g.over {
				switch {
				case g.dropping():
					if g.advanceDrop(time.Now()) {
						audio.playLock(g)
					}
				case g.clearCount != 0:
					if g.advanceClear() && g.reward.visible() {
						resetTimer(rewardTimer, rewardTick)
						if g.reward.comboPoints > 0 {
							audio.play("combo")
						}
					}
					if g.over {
						audio.play("over")
					}
				case g.step(false):
					audio.playLock(g)
				}
			}
			fall.Reset(g.interval())
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if g == nil && startRequested(screen, event) {
				g = newGame()
				screen.DisableMouse()
				fall.Reset(g.interval())
				continue
			}
			switch event := event.(type) {
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				switch {
				case event.Key() == tcell.KeyCtrlC || event.Key() == tcell.KeyEscape || event.Rune() == 'q':
					return nil
				case event.Rune() == 'm':
					audio.toggle()
				case g == nil:
				case event.Rune() == 'r':
					g = newGame()
					resetTimer(fall, g.interval())
				case event.Rune() == 'p' && !g.over:
					if g.dropping() {
						g.finishDrop()
						audio.playLock(g)
					}
					g.paused = !g.paused && !g.over
					resetTimer(fall, g.interval())
				case !g.paused && !g.over && g.clearCount == 0 && !g.dropping():
					width, height := screen.Size()
					if width >= viewWidth && height >= viewHeight {
						locked := playKey(g, event, audio)
						if locked {
							audio.playLock(g)
						}
						if locked || g.dropping() {
							resetTimer(fall, g.interval())
						}
					}
				}
			}
		}
	}
}

func startRequested(screen tcell.Screen, event tcell.Event) bool {
	width, height := screen.Size()
	if width < viewWidth || height < viewHeight {
		return false
	}
	switch event := event.(type) {
	case *tcell.EventKey:
		return event.Key() == tcell.KeyEnter
	case *tcell.EventMouse:
		x, y := event.Position()
		x -= (width - viewWidth) / 2
		y -= (height - viewHeight) / 2
		return event.Buttons()&tcell.Button1 != 0 &&
			x >= startButtonX && x < startButtonX+startButtonWidth &&
			y >= startButtonY && y < startButtonY+startButtonHeight
	}
	return false
}

func resetTimer(timer *time.Timer, delay time.Duration) {
	// 이전 방식의 타이머에서도 이미 도착한 신호를 비운다.
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(delay)
}

func playKey(g *game, event *tcell.EventKey, audio *soundPlayer) bool {
	switch {
	case event.Key() == tcell.KeyLeft || event.Rune() == 'a':
		g.move(-1, 0)
	case event.Key() == tcell.KeyRight || event.Rune() == 'd':
		g.move(1, 0)
	case event.Key() == tcell.KeyUp || event.Rune() == 'w':
		if g.rotate() {
			audio.play("rotate")
		}
	case event.Key() == tcell.KeyDown || event.Rune() == 's':
		return g.step(true)
	case event.Rune() == ' ':
		return g.beginDrop(time.Now())
	}
	return false
}
