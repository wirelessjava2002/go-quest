package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Draw renders the world, the player, enemies, items, and UI.
// This was previously in main.go; moving it to its own file keeps main.go smaller.
func (g *Game) Draw(screen *ebiten.Image) {
	// clear background
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
				if img, ok := g.Atlas.Get("floor"); ok && img != nil {
					screen.DrawImage(img, op)
				} else {
					img := ebiten.NewImage(TileSize, TileSize)
					img.Fill(color.NRGBA{45, 45, 55, 255})
					screen.DrawImage(img, op)
				}
			case TWall:
				if img, ok := g.Atlas.Get("wall"); ok && img != nil {
					screen.DrawImage(img, op)
				} else {
					img := ebiten.NewImage(TileSize, TileSize)
					img.Fill(color.NRGBA{80, 80, 90, 255})
					screen.DrawImage(img, op)
				}
			case TDoor:
				if img, ok := g.Atlas.Get("door"); ok && img != nil {
					screen.DrawImage(img, op)
				} else {
					img := ebiten.NewImage(TileSize, TileSize)
					img.Fill(color.NRGBA{180, 140, 60, 255})
					screen.DrawImage(img, op)
				}
			case TWater:
				// Ensure we actually have the water image
				if img, ok := g.Atlas.Get("water"); ok && img != nil {
					// Base pass (at the tile’s screen position)
					screen.DrawImage(img, op)

					// Stronger, visible shimmer overlay (no shaders)
					dx := math.Sin(g.time*2.6) * 1.6
					dy := math.Cos(g.time*2.0) * 1.2

					op2 := &ebiten.DrawImageOptions{}
					op2.GeoM.Translate((wx-g.CamXpx)+dx, (wy-g.CamYpx)+dy)
					// 15–25% brightness wobble; 0.35 alpha so it blends
					b := 1.0 + 0.20*math.Sin(g.time*3.3)
					op2.ColorM.Scale(b, b, b, 0.35)

					screen.DrawImage(img, op2)
				} else {
					// Fallback solid color (no shimmer possible)
					img := ebiten.NewImage(TileSize, TileSize)
					img.Fill(color.NRGBA{50, 90, 170, 255})
					screen.DrawImage(img, op)
				}
			}
		}
	}

	// Draw items on ground (if you have ItemsOnGround and icons)
	if g.ItemsOnGround != nil {
		for _, it := range g.ItemsOnGround {
			ix := float64(it.X*TileSize) - g.CamXpx
			iy := float64(it.Y*TileSize) - g.CamYpx

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(ix, iy)

			// --- GOLD DRAWING (uses atlas sprite at 2,1) ---
			if it.ID == "gold" {
				if img, ok := g.Atlas.Get("gold"); ok && img != nil {
					screen.DrawImage(img, op)
				} else {
					// fallback color if atlas missing
					fb := ebiten.NewImage(TileSize, TileSize)
					fb.Fill(color.NRGBA{255, 215, 0, 255}) // gold color
					screen.DrawImage(fb, op)
				}
				continue
			}

			// --- NORMAL ITEMS ---
			if it.Inst != nil && it.Inst.Icon() != nil {
				screen.DrawImage(it.Inst.Icon(), op)
				continue
			}

			// fallback if item has no icon
			fb := ebiten.NewImage(TileSize, TileSize)
			fb.Fill(color.NRGBA{200, 200, 0, 255})
			screen.DrawImage(fb, op)
		}
	}


	// Player
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(g.Player.X-g.CamXpx, g.Player.Y-g.CamYpx)

		// Prefer an atlas "player" image if available
		if img, ok := g.Atlas.Get("player"); ok && img != nil {
			screen.DrawImage(img, op)
		} else {
			// Fallback colored square
			img := ebiten.NewImage(TileSize, TileSize)
			img.Fill(color.NRGBA{0x6b, 0xc1, 0xff, 0xff})
			screen.DrawImage(img, op)
		}
	}


	// Draw enemies 
	// Assume g.Enemies []enemies.Enemy with Draw(screen, camX, camY) method
	if g.Enemies != nil {
	for _, e := range g.Enemies {
		// draw the enemy normally
		e.Draw(screen, g.CamXpx, g.CamYpx)

		// draw floating HP bar above enemy
		stats := e.Stats()
		hp := stats.HP
		hpMax := float64(stats.HPMax)

		if hpMax > 0 {
			pct := hp / hpMax
			if pct < 0 { pct = 0 }
			if pct > 1 { pct = 1 }

			// world position → screen coords
			sx := e.X() - g.CamXpx
			sy := e.Y() - g.CamYpx

			// bar size & placement
			barW := 28
			barH := 4
			bx := sx - float64(barW)/2                   // center horizontally
			by := sy - float64(TileSize)/2 - 2          // above the head

			// background (dark bar)
			bg := ebiten.NewImage(barW, barH)
			bg.Fill(color.NRGBA{40, 40, 40, 200})
			opbg := &ebiten.DrawImageOptions{}
			opbg.GeoM.Translate(bx, by)
			screen.DrawImage(bg, opbg)

			// foreground (red health)
			fw := int(float64(barW) * pct)
			if fw > 0 {
				fg := ebiten.NewImage(fw, barH)
				fg.Fill(color.NRGBA{200, 40, 40, 255})
				opfg := &ebiten.DrawImageOptions{}
				opfg.GeoM.Translate(bx, by)
				screen.DrawImage(fg, opfg)
			}
		}
	}

	}


	// === UI: Player stats panel (top-right corner) ===
	g.drawStatsPanel(screen)

	// Inventory, help text, tooltips etc.
	g.drawInventory(screen)
	g.drawInventoryHelp(screen)
	g.drawTooltip(screen)
}
