package main

import "image"

// Tile IDs — kept small and centralised
const (
	TEmpty = iota
	TFloor
	TWall
	TDoor
	TWater
)

// Tile & viewport sizing (move these here if you prefer,
// but if ViewW/ViewH/TileSize are already declared in main.go keep them there.
// If you already have these constants in main.go, **do not duplicate** them here.)
const (
	TileSize   = 32
	ViewTilesW = 20
	ViewTilesH = 15

	// Derived pixel sizes (these may already exist in main.go)
	ViewW = ViewTilesW * TileSize
	ViewH = ViewTilesH * TileSize
)

// Helpers for tile indexing and bounds
func (g *Game) idx(x, y int) int       { return y*g.W + x }
func (g *Game) inBounds(x, y int) bool { return x >= 0 && y >= 0 && x < g.W && y < g.H }
func (g *Game) at(x, y int) int        { return g.Tiles[g.idx(x, y)] }
func (g *Game) set(x, y, v int)        { g.Tiles[g.idx(x, y)] = v }

// small utility helpers used by dungeon/camera/etc.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func center(r image.Rectangle) (int, int) { return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2 }
