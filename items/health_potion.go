package items

import (
	"example.com/go-quest/atlas"
	"example.com/go-quest/player"
	"github.com/hajimehoshi/ebiten/v2"
)

type HealthPotion struct {
	icon *ebiten.Image
}

func (h *HealthPotion) ID() string         { return "health_potion" }
func (h *HealthPotion) Name() string       { return "Health Potion" }
func (h *HealthPotion) Icon() *ebiten.Image { return h.icon }

func (h *HealthPotion) OnPickup(p *player.Player) {}

func (h *HealthPotion) OnUse(p *player.Player) bool {
	// heal ~30% of max HP, clamp
	max := float64(p.Stats.HPMax)
	p.Stats.HP += 0.30 * max
	if p.Stats.HP > max {
		p.Stats.HP = max
	}
	return true // consumed
}

func (h *HealthPotion) OnDrop(p *player.Player, wx, wy int) bool {
	return true // allow dropping
}

func init() {
	Register("health_potion", func(atl *atlas.Atlas) Item {
		img, _ := atl.Get("icon.hp") // register this in atlas (or load single)
		return &HealthPotion{icon: img}
	})
}
