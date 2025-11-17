package enemies

import (
	"image/color"
	"math"

	"example.com/go-quest/rpg"
	"github.com/hajimehoshi/ebiten/v2"
)

// Base contains common fields for enemies and simple helpers.
// Individual enemy types embed Base and implement the rest.
type Base struct {
	id   string
	name string

	x, y float64 // pixel position

	// visuals
	icon *ebiten.Image

	// RPG data
	attr rpg.Attributes
	stats rpg.Stats
	mods []rpg.Modifier

	// combat
	hitCooldown    float64 // seconds between enemy attacks
	hitTimer       float64
	meleeRangePx   float64
	attackDamage   float64
	hitFlash float64

	// movement
	moveSpeed float64 // px per second

	alive bool
	time  float64

}

func (b *Base) ID() string            { return b.id }
func (b *Base) Name() string          { return b.name }
func (b *Base) X() float64            { return b.x }
func (b *Base) Y() float64            { return b.y }
func (b *Base) SetPos(x, y float64)   { b.x = x; b.y = y }
func (b *Base) Stats() rpg.Stats      { return b.stats }
func (b *Base) Attr() rpg.Attributes { return b.attr }
func (b *Base) IsAlive() bool        { return b.alive }

func (b *Base) TakeDamage(amount float64) {
	b.hitFlash = 0.15 // 150ms red flash
	b.stats.HP -= amount
	if b.stats.HP <= 0 {
		b.stats.HP = 0
		b.alive = false
	}
}

// simple cooldown-driven AttackIfInRange; returns true if attack occurred
func (b *Base) AttackIfInRange(px, py float64) bool {
	if !b.alive {
		return false
	}
	// cooldown
	if b.hitTimer > 0 {
		return false
	}
	dx := px - b.x
	dy := py - b.y
	dist := math.Hypot(dx, dy)
	if dist <= b.meleeRangePx {
		// do damage (caller should apply to player)
		b.hitTimer = b.hitCooldown
		return true
	}
	return false
}

func (b *Base) tick(dt float64) {
	b.time += dt
	if b.hitTimer > 0 {
		b.hitTimer -= dt
		if b.hitTimer < 0 {
			b.hitTimer = 0
		}
	}
	if b.hitFlash > 0 {
    b.hitFlash -= dt
}
}

// simple draw helper
func (b *Base) drawSelf(screen *ebiten.Image, camX, camY float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(b.x-camX, b.y-camY)
	if b.icon != nil {
		screen.DrawImage(b.icon, op)
		return
	}
	// fallback square
	img := ebiten.NewImage(28, 28)
	img.Fill(colorRGBA(200, 80, 80, 255))
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(b.x-camX-14+16, b.y-camY-14+16) // center on tile pixel
	screen.DrawImage(img, op2)

	if b.hitFlash > 0 {
    op.ColorM.Scale(1.5, 0.5, 0.5, 1.0) // red tint
}
}

func colorRGBA(r, g, b, a uint8) color.NRGBA {
	return color.NRGBA{r, g, b, a}
}
