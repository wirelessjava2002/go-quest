package enemies

import (
	"math"

	"example.com/go-quest/atlas"
	"example.com/go-quest/rpg"
	"github.com/hajimehoshi/ebiten/v2"
)

type Slime struct {
	Base
	// specific slime fields (e.g., wobble animation)
}

func newSlime(atl *atlas.Atlas) Enemy {
	s := &Slime{}
	s.id = "slime"
	s.name = "Slime"
	s.x = 0
	s.y = 0
	s.icon, _ = atl.Get("enemy.slime") // register an atlas key for slime
	// base attributes
	s.attr = rpg.Attributes{Level: 1, Str: 2, Dex: 2, Int: 1, Vit: 3, Wis: 1, Lck: 1}
	s.stats = rpg.Recompute(s.attr) // baseline
	// tweak stats
	s.stats.HP = float64(20)
	s.attackDamage = 4.0
	s.hitCooldown = 1.0
	s.meleeRangePx = 20.0
	s.moveSpeed = 24.0
	s.alive = true
	return s
}

func (s *Slime) Update(dt float64, px, py float64, passable func(tx, ty int) bool) {
	// basic wander + chase AI: if player within 128px, move towards; else simple wander
	s.tick(dt)
	dx := px - s.x
	dy := py - s.y
	dist := math.Hypot(dx, dy)
	if dist < 128 {
		// approach player
		if dist > s.meleeRangePx {
			nx := (dx / dist) * s.moveSpeed * dt
			ny := (dy / dist) * s.moveSpeed * dt
			s.x += nx
			s.y += ny
		}
		// attack handled by caller: AttackIfInRange
	} else {
		// simple idle wobble (no movement)
	}
}

func (s *Slime) Draw(screen *ebiten.Image, camX, camY float64) {
	s.drawSelf(screen, camX, camY)
}

func init() {
	Register("slime", func(atl *atlas.Atlas) Enemy {
		return newSlime(atl)
	})
}
