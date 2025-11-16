package inventory

import "example.com/go-quest/items"

type Inventory struct {
	Slots []items.Item
	Max   int
}

func New(max int) *Inventory {
	return &Inventory{Max: max, Slots: make([]items.Item, 0, max)}
}

func (inv *Inventory) Add(it items.Item) bool {
	if it == nil || len(inv.Slots) >= inv.Max {
		return false
	}
	inv.Slots = append(inv.Slots, it)
	return true
}

func (inv *Inventory) RemoveAt(idx int) items.Item {
	if idx < 0 || idx >= len(inv.Slots) {
		return nil
	}
	it := inv.Slots[idx]
	inv.Slots = append(inv.Slots[:idx], inv.Slots[idx+1:]...)
	return it
}

func (inv *Inventory) Get(idx int) items.Item {
	if idx < 0 || idx >= len(inv.Slots) {
		return nil
	}
	return inv.Slots[idx]
}

func (inv *Inventory) Count() int { return len(inv.Slots) }
