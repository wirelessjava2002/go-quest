package rpg

// Stats are the *computed* combat numbers derived from Attributes (+ mods).
type Stats struct {
	// Caps
	HPMax, MPMax, StaminaMax int
	// Offense / Defense
	Attack, Magic, Defense, Resist int
	// Rates / Chances
	CritChance float64 // 0..1
	CritMult   float64 // e.g. 1.5 = +50% damage
	// Movement
	MoveSpeed float64 // pixels per second for your player

	// Current resource values (runtime)
	HP, MP, Stamina float64
}

// Baseline computes stats from attributes (tweak the formulas to your taste).
func Baseline(a Attributes) Stats {
	s := Stats{}

	// Core resource caps (example formulas)
	s.HPMax = 50 + a.Vit*10 + a.Level*5
	s.MPMax = 20 + a.Wis*8 + a.Int*2
	s.StaminaMax = 50 + a.Vit*4 + a.Dex*3

	// Offense/Defense
	s.Attack = 2*a.Str + a.Level
	s.Magic = 2*a.Int + a.Level
	s.Defense = a.Vit + a.Level/2
	s.Resist = a.Wis + a.Level/2

	// Crits & speed
	s.CritChance = 0.05 + float64(a.Lck)*0.005 // 5% + 0.5% per LCK
	if s.CritChance > 0.5 {
		s.CritChance = 0.5
	}
	s.CritMult = 1.5
	// 120 px/s base + 2 per DEX
	s.MoveSpeed = 120.0 + float64(a.Dex*2)

	// Start full
	s.HP = float64(s.HPMax)
	s.MP = float64(s.MPMax)
	s.Stamina = float64(s.StaminaMax)
	return s
}

/* ---------- Modifiers (equipment, buffs, debuffs) ---------- */

// Modifier can apply changes to computed Stats (flat or multiplicative).
type Modifier interface {
	Apply(*Stats)
}

// Flat adds/subtracts flat values.
type Flat struct {
	HPMax, MPMax, StaminaMax int
	Attack, Magic            int
	Defense, Resist          int
	MoveSpeed                float64
	CritChance, CritMult     float64 // add to base (e.g. +0.05 chance)
}

func (m Flat) Apply(s *Stats) {
	s.HPMax += m.HPMax
	s.MPMax += m.MPMax
	s.StaminaMax += m.StaminaMax
	s.Attack += m.Attack
	s.Magic += m.Magic
	s.Defense += m.Defense
	s.Resist += m.Resist
	s.MoveSpeed += m.MoveSpeed
	s.CritChance += m.CritChance
	s.CritMult += m.CritMult
}

// Mult multiplies certain fields (1.10 = +10%).
type Mult struct {
	AttackMul, DefenseMul, SpeedMul float64
}

func (m Mult) Apply(s *Stats) {
	if m.AttackMul != 0 {
		s.Attack = int(float64(s.Attack) * m.AttackMul)
	}
	if m.DefenseMul != 0 {
		s.Defense = int(float64(s.Defense) * m.DefenseMul)
	}
	if m.SpeedMul != 0 {
		s.MoveSpeed = s.MoveSpeed * m.SpeedMul
	}
}

/* ---------- Helpers ---------- */

// Recompute recalculates Stats from Attributes and applies all modifiers.
// Use this whenever attributes/equipment/buffs change.
func Recompute(a Attributes, mods ...Modifier) Stats {
	s := Baseline(a)
	for _, m := range mods {
		m.Apply(&s)
	}
	// Clamp derived runtime resources to new caps (keep current percent if you prefer)
	if s.HP > float64(s.HPMax) {
		s.HP = float64(s.HPMax)
	}
	if s.MP > float64(s.MPMax) {
		s.MP = float64(s.MPMax)
	}
	if s.Stamina > float64(s.StaminaMax) {
		s.Stamina = float64(s.StaminaMax)
	}
	return s
}

// TickRegen applies simple passive regeneration per second.
func TickRegen(s *Stats, dt float64, a Attributes) {
	// Example regen rates (tune):
	hpRegen := 0.02*float64(s.HPMax) + float64(a.Vit)*0.05
	mpRegen := 0.01*float64(s.MPMax) + float64(a.Wis)*0.04
	stamRegen := 0.12*float64(s.StaminaMax) + float64(a.Dex)*0.08

	s.HP += hpRegen * dt
	if s.HP > float64(s.HPMax) {
		s.HP = float64(s.HPMax)
	}
	s.MP += mpRegen * dt
	if s.MP > float64(s.MPMax) {
		s.MP = float64(s.MPMax)
	}
	s.Stamina += stamRegen * dt
	if s.Stamina > float64(s.StaminaMax) {
		s.Stamina = float64(s.StaminaMax)
	}
}
