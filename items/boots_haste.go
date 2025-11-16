package items

import (
	"example.com/go-quest/atlas"
	"example.com/go-quest/player"
	"github.com/hajimehoshi/ebiten/v2"
)

type BootsHaste struct{ icon *ebiten.Image }

func (b *BootsHaste) ID() string          { return "boots_haste" }
func (b *BootsHaste) Name() string        { return "Boots of Haste" }
func (b *BootsHaste) Icon() *ebiten.Image { return b.icon }

func (b *BootsHaste) OnPickup(p *player.Player) {
	// Flat speed bonus while in inventory (simple model)
	p.Stats.MoveSpeed += 20
	p.RecomputeStats() // keep movement synced if you derive Speed from Stats
}

func (b *BootsHaste) OnUse(p *player.Player) bool {
	// Could toggle equip/unequip, but here we just “blink” stamina:
	p.Stats.Stamina += 10
	if p.Stats.Stamina > float64(p.Stats.StaminaMax) {
		p.Stats.Stamina = float64(p.Stats.StaminaMax)
	}
	return false // not consumed
}

func (b *BootsHaste) OnDrop(p *player.Player, wx, wy int) bool {
	// remove the passive when leaving inventory
	p.Stats.MoveSpeed -= 20
	p.RecomputeStats()
	return true
}

func init() {
	Register("boots_haste", func(atl *atlas.Atlas) Item {
		img, _ := atl.Get("icon.boots") // add this icon in atlas
		return &BootsHaste{icon: img}
	})
}
