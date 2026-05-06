//go:build !linux

package main

import (
   "github.com/Zyko0/go-sdl3/bin/binimg"
   "github.com/Zyko0/go-sdl3/bin/binsdl"
   "github.com/Zyko0/go-sdl3/bin/binttf"
)

type library struct {
   unloadFn func()
}

func newLibrary() *library {
   return &library{}
}

func (l *library) load() {
   libsdl := binsdl.Load()
   libimg := binimg.Load()
   libttf := binttf.Load()
   l.unloadFn = func() {
      libsdl.Unload()
      libimg.Unload()
      libttf.Unload()
   }
}

func (l *library) unload() {
   if l.unloadFn != nil {
      l.unloadFn()
   }
}
