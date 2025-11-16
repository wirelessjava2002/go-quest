package atlas

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

/*
Package atlas: minimal sprite atlas for 32x32 tiles (or any fixed tile size).

Features:
- Load a spritesheet and reference tiles by grid coordinates (tx,ty).
- Register human-readable names (e.g., "floor.stone1") -> sheet rect.
- Load standalone PNGs and treat them like single-tile entries.
- Fetch *ebiten.Image for drawing with ebiten.

This keeps things simple and dependency-light.
*/

type region struct {
	sheet string
	rect  image.Rectangle
}

type Atlas struct {
	TileSize int
	// name -> sheet image
	sheets map[string]*ebiten.Image
	// logical name -> (sheet, rect)
	entries map[string]region
}

// New creates an empty atlas with a fixed tile size (e.g., 32 for 32x32).
func New(tileSize int) *Atlas {
	return &Atlas{
		TileSize: tileSize,
		sheets:   make(map[string]*ebiten.Image),
		entries:  make(map[string]region),
	}
}

// LoadSheet loads an image file as a spritesheet registered under 'sheetName'.
// cols/rows are optional hints; you can pass 0,0 if not needed.
func (a *Atlas) LoadSheet(sheetName, path string, cols, rows int) error {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return fmt.Errorf("load sheet %q: %w", sheetName, err)
	}
	a.sheets[sheetName] = img
	return nil
}

// AddGridTile registers a tile name located at grid cell (tx,ty) on a sheet.
func (a *Atlas) AddGridTile(name, sheetName string, tx, ty int) error {
	sheet, ok := a.sheets[sheetName]
	if !ok {
		return fmt.Errorf("sheet %q not loaded", sheetName)
	}
	// Build rectangle in pixels from tx,ty.
	ts := a.TileSize
	r := image.Rect(tx*ts, ty*ts, (tx+1)*ts, (ty+1)*ts)

	// clamp to sheet bounds just in case
	r = r.Intersect(sheet.Bounds())
	a.entries[name] = region{sheet: sheetName, rect: r}
	return nil
}

// LoadSingle loads a single PNG file and registers it as 'name'.
// Useful for standalone 32x32 tiles (walls, floors, player, etc.).
func (a *Atlas) LoadSingle(name, path string) error {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return fmt.Errorf("load single %q: %w", name, err)
	}
	// Store the image as a dedicated "sheet" keyed by name,
	// and the entry covers the whole image.
	a.sheets[name] = img
	a.entries[name] = region{sheet: name, rect: img.Bounds()}
	return nil
}

// Get returns a *sub-image* view for a registered name.
// The view references the underlying sheet; safe to reuse across frames.
func (a *Atlas) Get(name string) (*ebiten.Image, bool) {
	reg, ok := a.entries[name]
	if !ok {
		return nil, false
	}
	img := a.sheets[reg.sheet]
	if img == nil {
		return nil, false
	}
	return img.SubImage(reg.rect).(*ebiten.Image), true
}

// MustGet panics if the name is missing (handy during setup).
func (a *Atlas) MustGet(name string) *ebiten.Image {
	img, ok := a.Get(name)
	if !ok {
		panic("atlas: missing entry: " + name)
	}
	return img
}
