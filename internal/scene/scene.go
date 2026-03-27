package scene

import "github.com/hajimehoshi/ebiten/v2"

// Scene defines the interface for all game scenes.
type Scene interface {
	Update() error
	Draw(screen *ebiten.Image)
}

// Switcher allows scenes to request a transition.
type Switcher interface {
	SwitchScene(next Scene)
}
