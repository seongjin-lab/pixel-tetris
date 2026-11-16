package main

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

//go:embed sounds/*.wav
var clips embed.FS

type soundPlayer struct {
	commands []string
	dir      string
	ctx      context.Context
	cancel   context.CancelFunc
	playing  sync.WaitGroup
}

func newSoundPlayer() *soundPlayer {
	commands := soundCommands()
	if len(commands) == 0 && runtime.GOOS != "windows" {
		return nil
	}
	dir, err := os.MkdirTemp("", "pixel-tetris-")
	if err != nil {
		return nil
	}
	player := &soundPlayer{commands: commands, dir: dir}
	for _, name := range [...]string{"rotate", "lock", "clear", "multi", "combo", "over"} {
		file := name + ".wav"
		data, err := clips.ReadFile("sounds/" + file)
		if err != nil || os.WriteFile(filepath.Join(dir, file), data, 0600) != nil {
			player.close()
			return nil
		}
	}
	player.toggle()
	return player
}

func (p *soundPlayer) toggle() {
	if p == nil {
		return
	}
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
		stopPlayback()
	} else {
		p.ctx, p.cancel = context.WithCancel(context.Background())
	}
}

func (p *soundPlayer) play(name string) {
	if p == nil || p.cancel == nil {
		return
	}
	p.playFile(filepath.Join(p.dir, name+".wav"))
}

func (p *soundPlayer) close() {
	if p == nil {
		return
	}
	if p.cancel != nil {
		p.cancel()
		stopPlayback()
	}
	p.playing.Wait()
	_ = os.RemoveAll(p.dir)
}

func (p *soundPlayer) playLock(g *game) {
	switch {
	case g.over:
		p.play("over")
	case g.clearCount > 1:
		p.play("multi")
	case g.clearCount != 0:
		p.play("clear")
	default:
		p.play("lock")
	}
}
