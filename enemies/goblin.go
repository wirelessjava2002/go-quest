package enemies

import (
	"math"

	"example.com/go-quest/atlas"
	"example.com/go-quest/rpg"
	"github.com/hajimehoshi/ebiten/v2"
)

type Goblin struct {
	Base
}

func newGoblin(atl *atlas.Atlas) Enemy {
	g := &Goblin{}
	g.id = "goblin"
	g.name = "Goblin"
	g.icon, _ = atl.Get("enemy.goblin")
	// Attributes a bit higher
	g.attr = rpg.Attributes{Level: 2, Str: 4, Dex: 3, Int: 2, Vit: 4, Wis: 1, Lck: 1}
	g.stats = rpg.Recompute(g.attr)
	g.stats.HP = float64(30)
	g.attackDamage = 8.0
	g.hitCooldown = 0.9
	g.meleeRangePx = 20.0
	g.moveSpeed = 48.0
	g.alive = true
	return g
}

func (g *Goblin) Update(dt float64, px, py float64, passable func(tx, ty int) bool) {
	g.tick(dt)
	// Goblin patttern: if player within 160px, dash in bursts
	dx := px - g.x
	dy := py - g.y
	dist := math.Hypot(dx, dy)
	if dist < 160 {
		if dist > g.meleeRangePx {
			// dash toward player
			speed := g.moveSpeed
			// if very close, slow down (avoid overshoot)
			nx := (dx / dist) * speed * dt
			ny := (dy / dist) * speed * dt
			g.x += nx
			g.y += ny
		}
	}
}

func (g *Goblin) Draw(screen *ebiten.Image, camX, camY float64) {
	g.drawSelf(screen, camX, camY)
}

func init() {
	Register("goblin", func(atl *atlas.Atlas) Enemy {
		return newGoblin(atl)
	})
}
