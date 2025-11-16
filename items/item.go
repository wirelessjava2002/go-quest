package items

import (
	"github.com/hajimehoshi/ebiten/v2"
	"example.com/go-quest/atlas"
	"example.com/go-quest/player"
)

// Item is your "script" API: each item file implements this.
type Item interface {
	ID() string
	Name() string
	Icon() *ebiten.Image

	// Called when player picks this up (e.g., auto-apply buff or stack)
	OnPickup(p *player.Player)                 // optional behavior; keep fast
	OnUse(p *player.Player) bool               // return true if consumed/changed
	OnDrop(p *player.Player, wx, wy int) bool  // return true if dropped in world
}

// -------- Registry so items can self-register via init() --------

type Ctor func(atl *atlas.Atlas) Item

var registry = map[string]Ctor{}

func Register(id string, c Ctor) {
	registry[id] = c
}

func New(id string, atl *atlas.Atlas) Item {
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
