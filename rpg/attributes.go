package rpg

// Attributes are "base" character traits, usually changed by levelling or points.
type Attributes struct {
	Level int
	Str   int // Strength
	Dex   int // Dexterity
	Int   int // Intelligence
	Vit   int // Vitality
	Wis   int // Wisdom
	Lck   int // Luck

	Unspent int // optional: points to distribute
}
