package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	sndAsync     = 0x0001
	sndNoDefault = 0x0002
	sndFilename  = 0x20000
)

var playSound = windows.NewLazySystemDLL("winmm.dll").NewProc("PlaySoundW")

func soundCommands() []string { return nil }

func (p *soundPlayer) playFile(file string) {
	path, err := windows.UTF16PtrFromString(file)
	if err == nil {
		playSound.Call(uintptr(unsafe.Pointer(path)), 0, sndAsync|sndNoDefault|sndFilename)
	}
}

func stopPlayback() {
	playSound.Call(0, 0, 0)
}
