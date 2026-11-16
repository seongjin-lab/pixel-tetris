//go:build !windows

package main

import (
	"os/exec"
	"runtime"
)

func soundCommands() []string {
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
	return commands
}

func (p *soundPlayer) playFile(file string) {
	ctx := p.ctx
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

func stopPlayback() {}
