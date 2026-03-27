package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"defense2/internal/scene"
)

func main() {
	ebiten.SetWindowSize(1200, 540)
	ebiten.SetWindowTitle("Tower Defense")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	g := scene.NewGame()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
