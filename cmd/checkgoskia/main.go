package main

import (
	"fmt"
	goskia "github.com/hoonfeng/goskia/skia"
)

func main() {
	goskia.Init()
	surf, err := goskia.NewRasterSurfaceN32Premul(100, 100)
	if err != nil {
		fmt.Println("FAIL:", err)
		return
	}
	defer surf.Release()
	fmt.Println("OK: surface created", surf.Width(), "x", surf.Height())
}
