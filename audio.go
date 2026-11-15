package main

import (
	"context"
	"embed"
	"os"
	"os/exec"
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
	names := []string{"afplay"}
	if runtime.GOOS == "linux" {
		names = []string{"pw-play", "paplay", "aplay"}
	}
	var commands []string
	for _, name := range names {
		if command, err := exec.LookPath(name); err == nil {
			commands = append(commands, command)
		}
	}
	if len(commands) == 0 {
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
	} else {
		p.ctx, p.cancel = context.WithCancel(context.Background())
	}
}

func (p *soundPlayer) play(name string) {
	if p == nil || p.cancel == nil {
		return
	}
	ctx := p.ctx
	file := filepath.Join(p.dir, name+".wav")
	p.playing.Add(1)
	go func() {
		defer p.playing.Done()
		for _, command := range p.commands {
			if ctx.Err() != nil || exec.CommandContext(ctx, command, file).Run() == nil {
				return
			}
		}
	}()
}

func (p *soundPlayer) close() {
	if p == nil {
		return
	}
	if p.cancel != nil {
		p.cancel()
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
