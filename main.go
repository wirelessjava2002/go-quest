package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand/v2"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil" 
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"example.com/go-quest/atlas"
	"example.com/go-quest/inventory"
	"example.com/go-quest/items"
	"example.com/go-quest/player"
	"example.com/go-quest/dungeon"
	"example.com/go-quest/rpg"
	"example.com/go-quest/enemies"

)

/*
	Top-Down Dungeon Crawler Scaffold (Go + Ebiten)

	What you get:
	- 32x32 tile world with a simple rooms+corridors generator
	- Smooth per-pixel player movement (in player package)
	- Pixel-based scrolling camera with a dead zone (prevents jitter/bouncing)
	- Spritesheet support (512x512 sheet, 16x16 cells of 32x32 tiles)
	- Inventory + items: pickup (E), use (Enter), drop (Q), cycle slots ([ / ])

	Assets (place in ./assets):
	- tiles.png   -> 512x512 sheet (16x16 grid). Pick which cells are floor/wall/icons below.
	- player.png  -> optional 32x32 player sprite (otherwise a blue square is used)
	- assets/fonts/pixel.ttf -> your pixel TTF
*/

type WorldItem struct {
	ID   string
	X, Y int // tile coords
	Inst items.Item
}

// Game holds all runtime state.
type Game struct {
	// Camera position in *pixels* (top-left of the screen in world coords)
	CamXpx, CamYpx float64

	// Tilemap
	W, H  int   // map width/height in tiles
	Tiles []int // len = W*H

	// Graphics
	Atlas    *atlas.Atlas
	imgFloor *ebiten.Image
	imgWall  *ebiten.Image
	imgDoor  *ebiten.Image
	imgWater *ebiten.Image

	// Player
	Player *player.Player

	// Enimies
	Enemies []enemies.Enemy

	// Timer (water shimmer, etc.)
	time float64

	// UI font
	uiFace font.Face

	// Inventory + world items
	Inv           *inventory.Inventory
	ItemsOnGround []WorldItem
	InvSel        int // selected inventory slot for use/drop (0..)
	tooltipText   string
	tooltipTimer  float64

}

// NewGame creates the world, loads assets, and positions the player.
func NewGame() *Game {
	g := &Game{
		W: 100, // 100x100 tiles of world (feel free to change)
		H: 100,
	}
	g.Tiles = make([]int, g.W*g.H)

	// --- Atlas setup ---
	g.Atlas = atlas.New(TileSize)

	// --- UI font (pixel 8-bit look) ---
	funcMust := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}
	{
		data, err := os.ReadFile("assets/fonts/pixel.ttf")
		funcMust(err)
		tt, err := opentype.Parse(data)
		funcMust(err)
		// Size & DPI: tweak to taste (10–14 looks good at 640x480)
		face, err := opentype.NewFace(tt, &opentype.FaceOptions{
			Size:    12,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		funcMust(err)
		g.uiFace = face
	}

	// Load main tilesheet (512x512, 16x16 grid)
	if err := g.Atlas.LoadSheet("tiles", "assets/tiles.png", 16, 16); err != nil {
		log.Printf("tiles.png not found, using fallback colors: %v", err)
	} else {
		// Register tiles (change tx,ty to match your sheet)
		_ = g.Atlas.AddGridTile("floor", "tiles", 0, 0)
		_ = g.Atlas.AddGridTile("wall", "tiles", 1, 0)
		_ = g.Atlas.AddGridTile("water", "tiles", 2, 0)
		_ = g.Atlas.AddGridTile("door", "tiles", 3, 0)

		// Register item icons (pick any cells you like)
		_ = g.Atlas.AddGridTile("icon.hp", "tiles", 0, 1)
		_ = g.Atlas.AddGridTile("icon.boots", "tiles", 1, 1)

		g.imgFloor, _ = g.Atlas.Get("floor")
		g.imgWall, _ = g.Atlas.Get("wall")
		g.imgWater, _ = g.Atlas.Get("water")
		g.imgDoor, _ = g.Atlas.Get("door")

		// enemies
		_ = g.Atlas.AddGridTile("enemy.slime", "tiles", 0, 2)
		_ = g.Atlas.AddGridTile("enemy.goblin", "tiles", 1, 2)

	}

	// Optional: standalone 32x32 player.png (register; ok if missing)
	_ = g.Atlas.LoadSingle("player", "assets/player.png")

	// Make a dungeon: rooms + L-shaped corridors.
	g.Tiles = dungeon.Generate(g.W, g.H, TFloor, TWall)

	// Create player from atlas (may be nil → fallback square)
	var pImg *ebiten.Image
	if img, ok := g.Atlas.Get("player"); ok {
		pImg = img
	}
	g.Player = player.New(pImg) // default speed is set in player.New()

	// Example mods (if your player package exposes these)
	g.Player.Mods = []rpg.Modifier{
		rpg.Flat{Attack: 3, MoveSpeed: 20}, // Leather Boots of Haste
		rpg.Mult{SpeedMul: 1.10},           // Minor Haste buff (+10%)
	}
	g.Player.RecomputeStats()

	// Place at first floor tile
placed:
	for ty := 0; ty < g.H; ty++ {
		for tx := 0; tx < g.W; tx++ {
			if g.at(tx, ty) == TFloor {
				g.Player.SetPosPixels(float64(tx*TileSize), float64(ty*TileSize))
				break placed
			}
		}
	}

	// Center camera on the player (pixel camera).
	g.centerCameraOnPlayer()

	// --- Inventory + ground items ---
	g.Inv = inventory.New(12)
	// icons already registered above
	g.spawnDemoNearPlayer()
	// Sprinkle a few example features (optional)
	g.placeRandomDoors(8)   // sprinkle a few doors on floor tiles
	g.paintWaterBlobs(5, 3) // 5 blobs, radius ~3 tiles each

	// --- Inventory + ground items ---
	g.Inv = inventory.New(12) // 12-slot bag

	// Spawn a couple of demo items on the ground (change coords to somewhere reachable)
	g.spawnItem("health_potion", 10, 10)
	g.spawnItem("boots_haste", 14, 12)

	// Enemies
	ptx := int((g.Player.X + TileSize/2) / TileSize)
	pty := int((g.Player.Y + TileSize/2) / TileSize)
	e1 := enemies.New("slime", g.Atlas)
	e1.SetPos(float64((ptx+3)*TileSize), float64(pty*TileSize))
	g.Enemies = append(g.Enemies, e1)

	e2 := enemies.New("goblin", g.Atlas)
	e2.SetPos(float64((ptx+6)*TileSize), float64(pty*TileSize))
	g.Enemies = append(g.Enemies, e2)

	// spawn some enemies for testing — e.g., 40 random monsters
	g.spawnEnemiesRandom(40, nil) // nil -> choose from all registered types

	// OR spawn a weighted mix:
	// g.spawnEnemiesRandom(20, []string{"slime"})
	// g.spawnEnemiesRandom(10, []string{"goblin"})


	return g
}

func (g *Game) spawnDemoNearPlayer() {
	ptx := int((g.Player.X + TileSize/2) / TileSize)
	pty := int((g.Player.Y + TileSize/2) / TileSize)

	// exactly at player tile and one to the right
	g.spawnItem("health_potion", ptx, pty)
	g.spawnItem("boots_haste", ptx+1, pty)
	log.Printf("spawned demo items at (%d,%d) and (%d,%d)", ptx, pty, ptx+1, pty)
}

func (g *Game) placeRandomDoors(n int) {
	placed := 0
	for tries := 0; tries < 500 && placed < n; tries++ {
		x := 1 + rand.IntN(g.W-2)
		y := 1 + rand.IntN(g.H-2)
		if g.at(x, y) != TFloor {
			continue
		}
		// Optional: only place on corridor “choke points”
		nFloor := 0
		if g.at(x+1, y) == TFloor {
			nFloor++
		}
		if g.at(x-1, y) == TFloor {
			nFloor++
		}
		if g.at(x, y+1) == TFloor {
			nFloor++
		}
		if g.at(x, y-1) == TFloor {
			nFloor++
		}
		if nFloor < 2 {
			continue
		}

		g.set(x, y, TDoor)
		placed++
	}
}

func (g *Game) paintWaterBlobs(count, radius int) {
	for i := 0; i < count; i++ {
		cx := 1 + rand.IntN(g.W-2)
		cy := 1 + rand.IntN(g.H-2)
		for y := cy - radius; y <= cy+radius; y++ {
			for x := cx - radius; x <= cx+radius; x++ {
				if !g.inBounds(x, y) {
					continue
				}
				dx := x - cx
				dy := y - cy
				if dx*dx+dy*dy <= radius*radius {
					if g.at(x, y) == TFloor {
						g.set(x, y, TWater)
					}
				}
			}
		}
	}
}

// spawnItem creates a world item at tile (tx,ty) using the items registry.
func (g *Game) spawnItem(id string, tx, ty int) {
	inst := items.New(id, g.Atlas)
	if inst == nil {
		log.Printf("item id %q not registered", id)
		return
	}
	g.ItemsOnGround = append(g.ItemsOnGround, WorldItem{
		ID: id, X: tx, Y: ty, Inst: inst,
	})
}

// Update handles input and world updates. Runs ~60x/sec by default.
func (g *Game) Update() error {
	// advance time (used for water shimmer etc.)
	dt := 1.0 / 60.0
	if tps := ebiten.ActualTPS(); tps > 0 {
		dt = 1.0 / tps
	}
	g.time += dt

	// player movement + collision via callback
	g.Player.Update(dt, TileSize, func(tx, ty int) bool {
		if tx < 0 || ty < 0 || tx >= g.W || ty >= g.H {
			return false
		}
		t := g.at(tx, ty)
		return t != TWall && t != TWater
	})

	// --- Player Attack ---
	didAttack := false
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.Player.CanAttack() {
		didAttack = true
		g.Player.DoAttack()
	}


	// Update enemies
	for i := 0; i < len(g.Enemies); i++ {
		ee := g.Enemies[i]
		if !ee.IsAlive() {
			// remove dead enemy
			g.Enemies = append(g.Enemies[:i], g.Enemies[i+1:]...)
			i--
			continue
		}

		// update AI
		ee.Update(dt, g.Player.X, g.Player.Y, func(tx, ty int) bool {
			if tx < 0 || ty < 0 || tx >= g.W || ty >= g.H {
				return false
			}
			t := g.at(tx, ty)
			return t != TWall && t != TWater
		})

		// enemy → player attacks (already implemented)
		if ee.AttackIfInRange(g.Player.X, g.Player.Y) {
			dmg := float64(ee.Stats().Attack) * 0.5
			if dmg <= 0 { dmg = 4 }
			g.Player.TakeDamage(dmg)
		}

		// --- NEW: player → enemy attack ---
		if didAttack {
			// distance from player to enemy
			dx := (ee.X() - g.Player.X)
			dy := (ee.Y() - g.Player.Y)
			dist := math.Hypot(dx, dy)

			if dist <= g.Player.AttackRangePx() {
				// deal damage
				dmg := g.Player.AttackDamage()
				ee.TakeDamage(dmg)

				// optional: add a knockback or flash here
			}
		}
	}


	// --- Inventory interactions ---

	// 1) Pickup when standing on an item, press E
	ptx := int((g.Player.X + TileSize/2) / TileSize)
	pty := int((g.Player.Y + TileSize/2) / TileSize)

	standingOnItem := false
	for i := range g.ItemsOnGround {
		wi := g.ItemsOnGround[i]
		if wi.X == ptx && wi.Y == pty {
			standingOnItem = true
			g.tooltipText = fmt.Sprintf("Press [E] to pick up %s", wi.Inst.Name())
			g.tooltipTimer = 1.0 // seconds visible after stepping off
			if inpututil.IsKeyJustPressed(ebiten.KeyE) {
				if g.Inv.Add(wi.Inst) {
					wi.Inst.OnPickup(g.Player)
					// remove from ground
					g.ItemsOnGround = append(g.ItemsOnGround[:i], g.ItemsOnGround[i+1:]...)
				}
			}
			break
		}
	}
	if !standingOnItem && g.tooltipTimer > 0 {
		g.tooltipTimer -= dt
		if g.tooltipTimer < 0 {
			g.tooltipTimer = 0
			g.tooltipText = ""
		}
	}

	// 2) Cycle selected slot with [ and ]
	if inpututil.IsKeyJustPressed(ebiten.KeyLeftBracket) {
		if g.InvSel > 0 {
			g.InvSel--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRightBracket) {
		if g.InvSel < g.Inv.Count()-1 {
			g.InvSel++
		}
	}

	// 3) Use selected item with ENTER
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		it := g.Inv.Get(g.InvSel)
		if it != nil {
			if it.OnUse(g.Player) { // consumed?
				g.Inv.RemoveAt(g.InvSel)
				if g.InvSel >= g.Inv.Count() {
					g.InvSel = g.Inv.Count() - 1
				}
				if g.InvSel < 0 {
					g.InvSel = 0
				}
			}
		}
	}

	// 4) Drop selected item with Q
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		it := g.Inv.Get(g.InvSel)
		if it != nil {
			ptx := int((g.Player.X + TileSize/2) / TileSize)
			pty := int((g.Player.Y + TileSize/2) / TileSize)
			if it.OnDrop(g.Player, ptx, pty) {
				g.spawnItem(it.ID(), ptx, pty)
				g.Inv.RemoveAt(g.InvSel)
				if g.InvSel >= g.Inv.Count() {
					g.InvSel = g.Inv.Count() - 1
				}
				if g.InvSel < 0 {
					g.InvSel = 0
				}
			}
		}
	}

	// camera follows player
	g.followCamera()

	return nil
}

// Draw renders only what's visible inside the viewport using the pixel camera.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{18, 18, 24, 255})

	// Determine visible tile range based on pixel camera.
	startTX := int(g.CamXpx) / TileSize
	startTY := int(g.CamYpx) / TileSize
	endTX := startTX + ViewTilesW + 1
	endTY := startTY + ViewTilesH + 1
	if startTX < 0 {
		startTX = 0
	}
	if startTY < 0 {
		startTY = 0
	}
	if endTX > g.W {
		endTX = g.W
	}
	if endTY > g.H {
		endTY = g.H
	}

	// Draw tiles.
	for ty := startTY; ty < endTY; ty++ {
		for tx := startTX; tx < endTX; tx++ {
			t := g.at(tx, ty)
			if t == TEmpty {
				continue
			}

			// World position (in pixels) of this tile.
			wx := float64(tx * TileSize)
			wy := float64(ty * TileSize)

			// Screen position = world - camera
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(wx-g.CamXpx, wy-g.CamYpx)

			// Choose tile graphic.
			switch t {
			case TFloor:
				if g.imgFloor != nil {
					screen.DrawImage(g.imgFloor, op)
				} else {
					img := ebiten.NewImage(TileSize, TileSize)
					img.Fill(color.NRGBA{45, 45, 55, 255})
					screen.DrawImage(img, op)
				}
			case TWall:
				if g.imgWall != nil {
					screen.DrawImage(g.imgWall, op)
				} else {
					img := ebiten.NewImage(TileSize, TileSize)
					img.Fill(color.NRGBA{80, 80, 90, 255})
					screen.DrawImage(img, op)
				}
			case TDoor:
				if g.imgDoor != nil {
					screen.DrawImage(g.imgDoor, op)
				} else {
					img := ebiten.NewImage(TileSize, TileSize)
					img.Fill(color.NRGBA{180, 140, 60, 255}) // brown fallback
					screen.DrawImage(img, op)
				}
			case TWater:
				if g.imgWater != nil {
					// Base pass (at the tile’s screen position)
					screen.DrawImage(g.imgWater, op)

					// Simple shimmer overlay (no shaders)
					dx := math.Sin(g.time*2.6) * 1.6
					dy := math.Cos(g.time*2.0) * 1.2

					op2 := &ebiten.DrawImageOptions{}
					op2.GeoM.Translate((wx-g.CamXpx)+dx, (wy-g.CamYpx)+dy)
					// brightness wobble + alpha
					b := 1.0 + 0.20*math.Sin(g.time*3.3)
					op2.ColorM.Scale(b, b, b, 0.35)

					screen.DrawImage(g.imgWater, op2)
				} else {
					// Fallback solid color (no shimmer possible)
					img := ebiten.NewImage(TileSize, TileSize)
					img.Fill(color.NRGBA{50, 90, 170, 255})
					screen.DrawImage(img, op)
				}
			}
		}
	}

	// Draw items on ground
	for _, wi := range g.ItemsOnGround {
		wx := float64(wi.X * TileSize)
		wy := float64(wi.Y * TileSize)
		icon := wi.Inst.Icon()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(wx-g.CamXpx, wy-g.CamYpx)
		if icon != nil {
			screen.DrawImage(icon, op)
		} else {
			// tiny fallback square
			img := ebiten.NewImage(TileSize, TileSize)
			img.Fill(color.NRGBA{240, 220, 80, 255})
			screen.DrawImage(img, op)
		}
	}

	// Draw enemies
	for _, ee := range g.Enemies {
		if ee.IsAlive() {
			ee.Draw(screen, g.CamXpx, g.CamYpx)
			// optionally draw a tiny HP bar above them
			s := ee.Stats()
			hpx := ee.X() - g.CamXpx
			hpy := ee.Y() - g.CamYpx - 10
			pct := clamp01(s.HP / float64(s.HPMax))
			drawBar(screen, int(hpx-16), int(hpy-6), 32, 6, pct,
				color.NRGBA{200, 40, 40, 255}, color.NRGBA{60, 20, 20, 255})
		}
	}

	// Draw player via its package method
	g.Player.Draw(screen, g.CamXpx, g.CamYpx, TileSize)

	// UI: Player stats panel (top-right corner)
	g.drawStatsPanel(screen)

	// UI: Inventory strip (bottom-left)
	g.drawInventory(screen)
	g.drawInventoryHelp(screen)

	g.drawTooltip(screen)
}

// drawInventoryHelp renders control hints under the inventory bar.
func (g *Game) drawInventoryHelp(screen *ebiten.Image) {
	if g.uiFace == nil {
		return
	}

	msg := "[ ]  Cycle  |  ENTER  Use  |  Q  Drop  |  E  Pick Up"
	white := color.NRGBA{230, 230, 240, 255}

	w := len(msg)*6 -16
	h := 18
	x := 8
	y := ViewH -4 // just under the inventory strip

	bg := ebiten.NewImage(w, h)
	bg.Fill(color.NRGBA{0, 0, 0, 140})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x-4), float64(y-10))
	screen.DrawImage(bg, op)

	text.Draw(screen, msg, g.uiFace, x, y, white)
}


func (g *Game) drawTooltip(screen *ebiten.Image) {
    if g.tooltipText == "" || g.uiFace == nil {
        return
    }

    // fade out over time (0–1)
    alpha := g.tooltipTimer
    if alpha > 1 {
        alpha = 1
    }
    if alpha < 0 {
        alpha = 0
    }

    // text width estimate
    msg := g.tooltipText
    w := len(msg)*6 + 16 // rough width
    h := 20
    x := (ViewW - w) / 2
    y := ViewH - 80

    bg := ebiten.NewImage(w, h)
    bg.Fill(color.NRGBA{0, 0, 0, uint8(180 * alpha)})
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Translate(float64(x), float64(y))
    screen.DrawImage(bg, op)

    white := color.NRGBA{255, 255, 255, uint8(255 * alpha)}
    text.Draw(screen, msg, g.uiFace, x+8, y+14, white)
}


// === UI: Stats panel (top-right) ===
func (g *Game) drawStatsPanel(screen *ebiten.Image) {
	p := g.Player
	if p == nil || g.uiFace == nil {
		return
	}

	const pad = 8
	panelW := 190
	panelH := 190
	x := ViewW - panelW - pad
	y := pad

	// Background
	bg := ebiten.NewImage(panelW, panelH)
	bg.Fill(color.NRGBA{0, 0, 0, 120})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(bg, op)

	tx := x + 10
	ty := y + 16
	white := color.White
	gray := color.NRGBA{180, 180, 200, 255}

	// Title
	text.Draw(screen, "PLAYER", g.uiFace, tx, ty, white)
	ty += 6
	// Separator line
	sep := ebiten.NewImage(panelW-20, 1)
	sep.Fill(color.NRGBA{80, 80, 90, 255})
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(float64(tx), float64(ty))
	screen.DrawImage(sep, op2)
	ty += 14

	// === Attributes in two columns ===
	text.Draw(screen, "ATTRIBUTES", g.uiFace, tx, ty, gray)
	ty += 14

	leftX := tx
	rightX := tx + 80 // spacing between columns
	lineH := 14

	text.Draw(screen, fmt.Sprintf("STR %2d", p.Attr.Str), g.uiFace, leftX, ty, white)
	text.Draw(screen, fmt.Sprintf("VIT %2d", p.Attr.Vit), g.uiFace, rightX, ty, white)
	ty += lineH

	text.Draw(screen, fmt.Sprintf("DEX %2d", p.Attr.Dex), g.uiFace, leftX, ty, white)
	text.Draw(screen, fmt.Sprintf("WIS %2d", p.Attr.Wis), g.uiFace, rightX, ty, white)
	ty += lineH

	text.Draw(screen, fmt.Sprintf("INT %2d", p.Attr.Int), g.uiFace, leftX, ty, white)
	text.Draw(screen, fmt.Sprintf("LCK %2d", p.Attr.Lck), g.uiFace, rightX, ty, white)
	ty += lineH + 6 // small gap before stats

	// === Stats section ===
	text.Draw(screen, "STATS", g.uiFace, tx, ty, gray)
	ty += 14

	// HP bar
	hpPct := clamp01(p.Stats.HP / float64(p.Stats.HPMax))
	drawBar(screen, tx, ty, panelW-20, 10, hpPct,
		color.NRGBA{200, 40, 40, 255},
		color.NRGBA{60, 20, 20, 255})
	text.Draw(screen, fmt.Sprintf("HP %3.0f/%d", p.Stats.HP, p.Stats.HPMax), g.uiFace, tx+4, ty+9, white)
	ty += 18

	// MP bar
	mpPct := clamp01(p.Stats.MP / float64(p.Stats.MPMax))
	drawBar(screen, tx, ty, panelW-20, 10, mpPct,
		color.NRGBA{60, 120, 230, 255},
		color.NRGBA{20, 30, 60, 255})
	text.Draw(screen, fmt.Sprintf("MP %3.0f/%d", p.Stats.MP, p.Stats.MPMax), g.uiFace, tx+4, ty+9, white)
	ty += 18

	// Stamina bar
	stPct := clamp01(p.Stats.Stamina / float64(p.Stats.StaminaMax))
	drawBar(screen, tx, ty, panelW-20, 10, stPct,
		color.NRGBA{60, 180, 80, 255},
		color.NRGBA{20, 50, 25, 255})
	text.Draw(screen, fmt.Sprintf("STM %3.0f/%d", p.Stats.Stamina, p.Stats.StaminaMax), g.uiFace, tx+4, ty+9, white)
	ty += 20

	// ATK/DEF/SPD
	text.Draw(screen, fmt.Sprintf("ATK %d", p.Stats.Attack), g.uiFace, tx, ty, white)
	text.Draw(screen, fmt.Sprintf("DEF %d", p.Stats.Defense), g.uiFace, tx+60, ty, white)
	ty += 14
	text.Draw(screen, fmt.Sprintf("SPD %d", int(p.Stats.MoveSpeed)), g.uiFace, tx, ty, white)
}

// === UI: Inventory strip (bottom-left) ===
func (g *Game) drawInventory(screen *ebiten.Image) {
	const slotSize = 36
	const cols = 8
	x0, y0 := 8, ViewH-(slotSize+12)

	// background
	bg := ebiten.NewImage(cols*slotSize+8, slotSize+8)
	bg.Fill(color.NRGBA{0, 0, 0, 160})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x0-4), float64(y0-4))
	screen.DrawImage(bg, op)

	for i := 0; i < g.Inv.Count() && i < cols; i++ {
		it := g.Inv.Get(i)
		x := x0 + i*slotSize
		y := y0

		// slot border
		slot := ebiten.NewImage(slotSize-4, slotSize-4)
		slot.Fill(color.NRGBA{50, 50, 60, 255})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(slot, op)

		// icon
		if it != nil && it.Icon() != nil {
			op2 := &ebiten.DrawImageOptions{}
			op2.GeoM.Translate(float64(x+2), float64(y+2))
			screen.DrawImage(it.Icon(), op2)
		}

		// selection highlight
		if i == g.InvSel {
			hl := ebiten.NewImage(slotSize-4, slotSize-4)
			hl.Fill(color.NRGBA{255, 255, 255, 60})
			op3 := &ebiten.DrawImageOptions{}
			op3.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(hl, op3)
		}
	}
}

// drawBar draws a simple filled bar (background + foreground by percentage).
func drawBar(screen *ebiten.Image, x, y, w, h int, pct float64, fg, bg color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	// background
	bgImg := ebiten.NewImage(w, h)
	bgImg.Fill(bg)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(bgImg, op)

	// foreground
	fw := int(float64(w) * pct)
	if fw < 0 {
		fw = 0
	}
	if fw > w {
		fw = w
	}
	if fw == 0 {
		return
	}
	fgImg := ebiten.NewImage(fw, h)
	fgImg.Fill(fg)
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(fgImg, op2)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Layout fixes the logical resolution of the window (Ebiten will scale as needed).
func (g *Game) Layout(ow, oh int) (int, int) { return ViewW, ViewH }


// pick a random floor tile and spawn enemies there.
func (g *Game) spawnEnemiesRandom(n int, allowedTypes []string) {
    if n <= 0 {
        return
    }

    // build list of candidate floor tile coords
    candidates := make([]struct{ x, y int }, 0, 1024)
    for y := 0; y < g.H; y++ {
        for x := 0; x < g.W; x++ {
            if g.at(x, y) == TFloor {
                // avoid spawning on player tile
                px := int((g.Player.X + TileSize/2) / TileSize)
                py := int((g.Player.Y + TileSize/2) / TileSize)
                if x == px && y == py {
                    continue
                }
                // skip tiles that already hold an item or enemy
                if g.tileHasItemOrEnemy(x, y) {
                    continue
                }
                candidates = append(candidates, struct{ x, y int }{x, y})
            }
        }
    }

    if len(candidates) == 0 {
        return
    }

    // fallback: if no allowedTypes provided, choose from registry
    types := allowedTypes
    if len(types) == 0 {
        types = enemies.AllIDs()
    }
    if len(types) == 0 {
        log.Println("no enemy types registered")
        return
    }

	for i := 0; i < n && len(candidates) > 0; i++ {

		// pick a candidate tile
		idx := rand.IntN(len(candidates))
		c := candidates[idx]

		// pick random enemy type
		et := types[rand.IntN(len(types))]
		e := enemies.New(et, g.Atlas)
		if e == nil {
			// if enemy type is not registered, skip
			// but don’t crash
			continue
		}

		// center enemy on tile (px, py)
		px := float64(c.x*TileSize + TileSize/2)
		py := float64(c.y*TileSize + TileSize/2)
		e.SetPos(px, py)

		g.Enemies = append(g.Enemies, e)

		// remove tile from candidate list so we don’t spawn duplicates there
		candidates[idx] = candidates[len(candidates)-1]
		candidates = candidates[:len(candidates)-1]
	}

}

// helper: check if a tile already contains an item or enemy
func (g *Game) tileHasItemOrEnemy(tx, ty int) bool {
    // item check
    for _, it := range g.ItemsOnGround {
        if it.X == tx && it.Y == ty {
            return true
        }
    }
    // enemy check: convert enemy pixel pos to tile coords
    for _, e := range g.Enemies {
        ex := int((e.X()) / TileSize)
        ey := int((e.Y()) / TileSize)
        if ex == tx && ey == ty {
            return true
        }
    }
    // you can also check player here, but spawnEnemiesRandom already avoids player tile
    return false
}


/* =========================
   Misc helpers
   ========================= */

func spriteRect(tx, ty int) image.Rectangle {
	// Convert spritesheet grid coords (tx,ty) into pixel rectangle.
	return image.Rect(tx*TileSize, ty*TileSize, (tx+1)*TileSize, (ty+1)*TileSize)
}

func abs(a int) int { if a < 0 { return -a }; return a }


/* =========================
   Entry point
   ========================= */

func main() {
	ebiten.SetWindowSize(ViewW, ViewH)
	ebiten.SetWindowTitle("Go Quest")

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
