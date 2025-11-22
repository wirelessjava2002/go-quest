package player

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"example.com/go-quest/rpg"
)

// Player holds position (in pixels), movement speed, and an optional sprite.
type Player struct {
	X, Y  float64       // pixel position (top-left of the 32x32 sprite)
	Speed float64       // pixels per second
	Img   *ebiten.Image // optional 32x32 image; if nil we draw a blue square
	time  float64       // local timer (for simple effects if you want)

	Attr  rpg.Attributes
	Stats rpg.Stats
	Mods  []rpg.Modifier // equipment/buffs currently applied

	// Stamina recovery behaviour
	stamRecoverDelay float64 // seconds to wait after moving before regen
	stamRecoverTimer float64 // counts down to 0, then regen resumes

	attackTimer float64
	attackCooldown float64

	Gold int
}

// New creates a player with a given sprite (can be nil). Speed is px/s.
func New(img *ebiten.Image) *Player {
	p := &Player{
		Img: img,
		Attr: rpg.Attributes{
			Level: 1, Str: 4, Dex: 6, Int: 3, Vit: 5, Wis: 3, Lck: 2,
		},
	}
	// Initial compute
	p.RecomputeStats()
	p.stamRecoverDelay = 0.6 // ~600ms feels good
	p.attackCooldown = 0.4 // 400 ms per swing
	return p
}

// RecomputeStats recalculates Stats from Attr and Mods, and syncs movement speed.
func (p *Player) RecomputeStats() {
	p.Stats = rpg.Recompute(p.Attr, p.Mods...)
	// Drive movement speed from Stats (so boots, buffs affect speed)
	p.Speed = p.Stats.MoveSpeed
}

// speedFactorFromStamina returns a multiplier [0..1] based on current stamina.
// - >=50% stamina: full speed (1.0)
// - 0..50%: linearly drops to a floor (e.g., 0.3)
// - 0%: hard stop (0.0)
func (p *Player) speedFactorFromStamina() float64 {
	if p.Stats.Stamina <= 0 {
		return 0.0 // out of stamina → can't move
	}
	max := float64(p.Stats.StaminaMax)
	if max <= 0 {
		return 1.0
	}
	frac := p.Stats.Stamina / max // 0..1

	if frac >= 0.5 {
		return 1.0
	}
	// Linear from 0.5 → 0.0: 1.0 → minFloor
	const minFloor = 0.3 // you can tweak (0.2..0.5)
	return minFloor + (1.0-minFloor)*(frac/0.5)
}

// EffectiveSpeed = base MoveSpeed (from Stats) × stamina factor.
func (p *Player) EffectiveSpeed() float64 {
	return p.Stats.MoveSpeed * p.speedFactorFromStamina()
}

func (p *Player) SetPosPixels(x, y float64) {
	p.X, p.Y = x, y
}

// Update reads input and attempts to move the player.
// - dt: seconds since last frame
// - tileSize: e.g., 32
// - passable(tx,ty): returns true if that tile is walkable
func (p *Player) Update(dt float64, tileSize int, passable func(tx, ty int) bool) {
	p.time += dt

	// ---- Movement input ----
	ax, ay := 0.0, 0.0
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		ax -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		ax += 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		ay -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		ay += 1
	}

	// Compute stamina-scaled speed (0 at zero stamina)
	speed := p.EffectiveSpeed()

	// No velocity if out of stamina (speed==0) or no input
	moving := (ax != 0 || ay != 0) && speed > 0

	// Optional: normalize diagonal so NE isn't faster than N
	if ax != 0 && ay != 0 {
		// 1/sqrt(2) ≈ 0.707
		ax *= 0.70710678
		ay *= 0.70710678
	}

	// ---- Move attempt ----
	if moving {
		dx := ax * speed * dt
		dy := ay * speed * dt

		nx := p.X + dx
		ny := p.Y + dy
		tx := int((nx + float64(tileSize)/2) / float64(tileSize))
		ty := int((ny + float64(tileSize)/2) / float64(tileSize))

		if passable(tx, ty) {
			p.X, p.Y = nx, ny

			// Stamina drain while moving
			// Tip: scale drain a touch with speedFactor so limping costs slightly less
			sf := p.speedFactorFromStamina()
			drainPerSec := 4.0 // base drain per second (tweak)
			p.Stats.Stamina -= (drainPerSec * (0.6 + 0.4*sf)) * dt
			if p.Stats.Stamina < 0 {
				p.Stats.Stamina = 0
			}

			// Reset regen cooldown whenever we actually move
			p.stamRecoverTimer = p.stamRecoverDelay
		}
	} else {
		// Not moving (or stamina=0) → count down recovery, then regen
		if p.stamRecoverTimer > 0 {
			p.stamRecoverTimer -= dt
		} else {
			// Passive regen
			rpg.TickRegen(&p.Stats, dt, p.Attr)
		}
	}

	// Clamp resources to caps
	if p.Stats.Stamina > float64(p.Stats.StaminaMax) {
		p.Stats.Stamina = float64(p.Stats.StaminaMax)
	}
	if p.Stats.Stamina < 0 {
		p.Stats.Stamina = 0
	}

	if p.attackTimer > 0 {
		p.attackTimer -= dt
	}
}

// Draw renders the player at (X,Y) relative to the camera (camX,camY).
// If Img is nil, draws a blue 32x32 square as a fallback.
func (p *Player) Draw(screen *ebiten.Image, camX, camY float64, tileSize int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(p.X-camX, p.Y-camY)

	if p.Img != nil {
		screen.DrawImage(p.Img, op)
		return
	}

	// Fallback: blue square with a tiny breathing effect
	img := ebiten.NewImage(tileSize, tileSize)
	// subtle brightness pulse
	b := 0.90 + 0.10*math.Sin(p.time*3.2)
	col := color.NRGBA{R: uint8(0x6b), G: uint8(0xc1), B: uint8(0xff), A: 0xff}
	img.Fill(color.NRGBA{
		R: uint8(float64(col.R) * b),
		G: uint8(float64(col.G) * b),
		B: uint8(float64(col.B) * b),
		A: col.A,
	})
	screen.DrawImage(img, op)
}

// TakeDamage applies damage to the player and clamps HP to zero.
func (p *Player) TakeDamage(d float64) {
	p.Stats.HP -= d
	if p.Stats.HP < 0 {
		p.Stats.HP = 0
	}
}

// IsAlive returns whether player still has HP > 0.
func (p *Player) IsAlive() bool {
	return p.Stats.HP > 0
}

// AttackDamage returns how much damage the player deals.
// You can mix in STR, DEX, weapons, etc.
func (p *Player) AttackDamage() float64 {
	return float64(p.Stats.Attack)
}

// AttackRangePx is the melee reach in pixels.
func (p *Player) AttackRangePx() float64 {
	return 24 // one tile edge = 32px, so 24 feels good
}

func (p *Player) CanAttack() bool {
	return p.attackTimer <= 0
}

func (p *Player) DoAttack() {
	p.attackTimer = p.attackCooldown
}


