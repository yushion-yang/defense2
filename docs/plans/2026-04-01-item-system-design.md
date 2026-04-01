# Item System Design

## Overview

Consumable items that directly boost tower Base/Potential attribute values. 6 item types covering 3 dimensions x 2 layers. Each starts with 5 in stock.

## Items

| Item | Target Field | Boost | Color |
|------|-------------|-------|-------|
| Attack Whetstone (攻击磨石) | BaseDamage | +2 | Red |
| Attack Scroll (攻击秘卷) | PotentialDamage | +3 | DarkRed |
| RapidFire Gear (速射齿轮) | BaseSpeed | +0.15 | Yellow |
| RapidFire Scroll (速射秘卷) | PotentialSpeed | +0.2 | DarkYellow |
| Scope Lens (瞄准镜片) | BaseRange | +12 | Blue |
| Scope Scroll (瞄准秘卷) | PotentialRange | +18 | DarkBlue |

Naming: Base items use concrete nouns, Potential items use "Scroll" (秘卷).

## UI: ActionBar

New bottom-center toolbar replacing TopBar's build button:

- Height 40px, anchored above BottomMargin
- Buttons: [Build] [Items] (expandable for future)
- Button size 80x34, rounded, horizontally centered

## UI: Item Panel

Popup above ActionBar when "Items" button clicked:

- 2-column x 3-row grid, card size 90x60, gap 8
- Each card: color circle + dimension icon + name + quantity
- Zero-quantity cards greyed out, not draggable

## Interaction: Drag & Drop

1. Open item panel (modeItemPanel)
2. Press & drag item card -> enter drag state, panel goes semi-transparent
3. Floating item icon follows cursor
4. Release on tower -> apply boost + toast + consume item
5. Release elsewhere -> cancel, no consumption
6. ESC cancels drag

## New interactModes

- `modeItemPanel` - item panel open
- `modeItemDrag` - dragging an item

## Data Model

```go
// internal/core/item/item.go
type Kind int // 6 constants

type Def struct {
    Kind     Kind
    Name     string
    BoostVal float64
    Color    color.RGBA
}

type Inventory struct {
    Counts [6]int
}

func (inv *Inventory) Use(kind Kind) bool
func ApplyItem(t *tower.Tower, kind Kind)  // modifies Base/Potential + RecalcStats
```
