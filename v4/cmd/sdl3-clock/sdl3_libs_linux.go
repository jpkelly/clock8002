//go:build linux

package main

import (
   "github.com/Zyko0/go-sdl3/img"
   "github.com/Zyko0/go-sdl3/sdl"
   "github.com/Zyko0/go-sdl3/ttf"
)

type library struct {
   unloadFn func()
}

func newLibrary() *library {
   return &library{}
}

func (l *library) load() {
   sdl.LoadLibrary(sdl.Path())
   ttf.LoadLibrary(ttf.Path())
   img.LoadLibrary(img.Path())
}

func (l *library) unload() {
}
