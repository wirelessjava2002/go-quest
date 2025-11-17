package enemies

import (
	"github.com/hajimehoshi/ebiten/v2"
	"example.com/go-quest/atlas"
	"example.com/go-quest/rpg"
)

// Enemy is the runtime interface the game uses.
type Enemy interface {
	ID() string
	Name() string

	// Position in pixels
	X() float64
	Y() float64
	SetPos(x, y float64)

	// Update AI (dt seconds), given a reference to player pos and a passable callback
	Update(dt float64, px, py float64, passable func(tx, ty int) bool)

	// Draw the enemy (screen coords are handled by caller, or provide camera)
	Draw(screen *ebiten.Image, camX, camY float64)

	// Combat API
	TakeDamage(amount float64)            // apply damage to the enemy
	AttackIfInRange(px, py float64) bool  // returns true if it attacked and did damage (you may want to handle damage externally)

	// Status
	IsAlive() bool

	// Expose stats/attrs for UI / debug
	Stats() rpg.Stats
	Attr() rpg.Attributes
}

// Registry (so enemy files can self-register)
type Ctor func(atl *atlas.Atlas) Enemy

var registry = map[string]Ctor{}

func Register(id string, c Ctor) {
	registry[id] = c
}

func New(id string, atl *atlas.Atlas) Enemy {
	if c, ok := registry[id]; ok {
		return c(atl)
	}
	return nil
}

func AllIDs() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	return out
}
