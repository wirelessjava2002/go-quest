package main

// Camera helpers (pixel-based camera centered on player)

func (g *Game) centerCameraOnPlayer() {
	// center camera on player's pixel coordinates
	g.CamXpx = g.Player.X - float64(ViewW)/2
	g.CamYpx = g.Player.Y - float64(ViewH)/2
	g.clampCamPx()
}

// followCamera keeps the player inside a dead zone to avoid micro-jitter.
// If the player leaves the dead zone, shift the camera by the overlap.
func (g *Game) followCamera() {
	// Dead zone bounds inside the viewport (in pixels).
	deadLeft := 160.0
	deadRight := float64(ViewW - 160)
	deadTop := 120.0
	deadBottom := float64(ViewH - 120)

	// Player position relative to current camera.
	px := g.Player.X - g.CamXpx
	py := g.Player.Y - g.CamYpx

	// Shift camera if player goes out of bounds of the dead zone.
	if px < deadLeft {
		g.CamXpx -= (deadLeft - px)
	} else if px > deadRight {
		g.CamXpx += (px - deadRight)
	}
	if py < deadTop {
		g.CamYpx -= (deadTop - py)
	} else if py > deadBottom {
		g.CamYpx += (py - deadBottom)
	}

	g.clampCamPx()
}

// clampCamPx keeps the camera inside world bounds (in pixels).
func (g *Game) clampCamPx() {
	maxX := float64(g.W*TileSize - ViewW)
	maxY := float64(g.H*TileSize - ViewH)

	if g.CamXpx < 0 {
		g.CamXpx = 0
	}
	if g.CamYpx < 0 {
		g.CamYpx = 0
	}
	if g.CamXpx > maxX {
		g.CamXpx = maxX
	}
	if g.CamYpx > maxY {
		g.CamYpx = maxY
	}
}
