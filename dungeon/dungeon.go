package dungeon

import (
	"image"
	"math/rand/v2"
)

// Generate returns a W*H tile slice filled with walls and carved floors.
// floorID and wallID come from your game's tile constants (e.g., TFloor/TWall).
func Generate(W, H int, floorID, wallID int) []int {
	tiles := make([]int, W*H)

	// Start fully walled.
	for i := range tiles {
		tiles[i] = wallID
	}

	rooms := make([]image.Rectangle, 0, 32)
	const maxRooms = 24

	for r := 0; r < maxRooms; r++ {
		w := 4 + rand.IntN(8) // room width: 4..11 tiles
		h := 4 + rand.IntN(8) // room height: 4..11 tiles
		x := 1 + rand.IntN(W-w-2)
		y := 1 + rand.IntN(H-h-2)
		room := image.Rect(x, y, x+w, y+h)

		if overlaps(room, rooms) {
			continue // skip if it overlaps (with padding)
		}

		carveRoom(tiles, W, room, floorID)
		if len(rooms) > 0 {
			// Connect to previous room center with an L-shaped corridor.
			pcx, pcy := center(rooms[len(rooms)-1])
			cx, cy := center(room)
			// Horizontal then vertical (simple and readable).
			carveH(tiles, W, min(pcx, cx), max(pcx, cx), pcy, floorID)
			carveV(tiles, W, min(pcy, cy), max(pcy, cy), cx, floorID)
		}
		rooms = append(rooms, room)
	}

	return tiles
}

/* ---------- helpers (private) ---------- */

func idx(W, x, y int) int { return y*W + x }

func carveRoom(tiles []int, W int, r image.Rectangle, floor int) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			tiles[idx(W, x, y)] = floor
		}
	}
}
func carveH(tiles []int, W, x1, x2, y, floor int) {
	for x := x1; x <= x2; x++ {
		tiles[idx(W, x, y)] = floor
	}
}
func carveV(tiles []int, W, y1, y2, x, floor int) {
	for y := y1; y <= y2; y++ {
		tiles[idx(W, x, y)] = floor
	}
}

func overlaps(r image.Rectangle, rooms []image.Rectangle) bool {
	for _, o := range rooms {
		// Expand existing room by 1 tile of padding to avoid merging.
		pad := image.Rect(o.Min.X-1, o.Min.Y-1, o.Max.X+1, o.Max.Y+1)
		if r.Overlaps(pad) {
			return true
		}
	}
	return false
}
func center(r image.Rectangle) (int, int) { return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2 }
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
