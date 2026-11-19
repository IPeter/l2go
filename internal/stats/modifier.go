package stats

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// ErrInvalid is wrapped by the errors this package returns for bad input:
// unknown names, out-of-range values, missing templates.
var ErrInvalid = errors.New("invalid stat input")

// Op is how a modifier changes a stat value (Mobius StatFunction and the
// Func* classes).
type Op uint8

// The operations.
const (
	OpSet       Op = iota + 1 // replace the value with Value (FuncSet)
	OpAdd                     // add Value (FuncAdd)
	OpSub                     // subtract Value (FuncSub)
	OpMul                     // multiply by Value (FuncMul)
	OpDiv                     // divide by Value (FuncDiv)
	OpShare                   // servitors only; no effect on players and NPCs (FuncShare)
	OpEnchant                 // add an item enchant bonus, precomputed into Value (FuncEnchant)
	OpEnchantHP               // add an item enchant HP bonus, precomputed into Value (FuncEnchantHp)
)

// Orders. Lower orders run first. The formula functions all run at
// OrderFormula; the others are the default orders of StatFunction.java.
const (
	OrderSet       = 0  // Set and Enchant
	OrderFormula   = 1  // the formula functions (FuncPAtkMod, FuncMaxHpMul, ...)
	OrderMul       = 20 // Mul and Div
	OrderAdd       = 30 // Add, Sub and Share
	OrderEnchantHP = 40 // EnchantHp
)

var opNames = [...]string{
	OpSet:       "Set",
	OpAdd:       "Add",
	OpSub:       "Sub",
	OpMul:       "Mul",
	OpDiv:       "Div",
	OpShare:     "Share",
	OpEnchant:   "Enchant",
	OpEnchantHP: "EnchantHp",
}

var opOrders = [...]int{
	OpSet:       OrderSet,
	OpAdd:       OrderAdd,
	OpSub:       OrderAdd,
	OpMul:       OrderMul,
	OpDiv:       OrderMul,
	OpShare:     OrderAdd,
	OpEnchant:   OrderSet,
	OpEnchantHP: OrderEnchantHP,
}

// Valid reports whether op is one of the operations.
func (op Op) Valid() bool { return op >= OpSet && op <= OpEnchantHP }

// String returns the Mobius name of the operation ("Set", "Add", ...).
func (op Op) String() string {
	if op.Valid() {
		return opNames[op]
	}
	return fmt.Sprintf("Op(%d)", uint8(op))
}

// DefaultOrder returns the order Mobius gives the operation when the data
// names none (StatFunction): Set and Enchant 0, Mul and Div 20, Add, Sub and
// Share 30, EnchantHp 40. It is -1 for an invalid op.
func (op Op) DefaultOrder() int {
	if op.Valid() {
		return opOrders[op]
	}
	return -1
}

// ParseOp returns the operation with the given name, ignoring case, as
// StatFunction.valueOf(name.toUpperCase()) does.
func ParseOp(name string) (Op, bool) {
	for op := OpSet; op <= OpEnchantHP; op++ {
		if strings.EqualFold(opNames[op], name) {
			return op, true
		}
	}
	return 0, false
}

// OwnerKind says what added a modifier.
type OwnerKind uint8

// The owner kinds.
const (
	OwnerNone   OwnerKind = iota // nobody; the formula functions have no owner either
	OwnerItem                    // an equipped item; ID is its object ID
	OwnerSkill                   // a passive skill; the caller chooses the ID
	OwnerEffect                  // a buff or debuff instance; the caller chooses the ID
)

// Owner identifies who added a modifier, so that all of its modifiers can be
// removed together (Mobius removeStatsOwner). Owners compare by value.
type Owner struct {
	Kind OwnerKind
	ID   int64
}

// ItemOwner returns the owner of the modifiers of the item with the given
// object ID.
func ItemOwner(objectID int32) Owner { return Owner{Kind: OwnerItem, ID: int64(objectID)} }

// Context is what a condition sees during one calculation: the other
// creature and the skill involved, if any (the target and skill of Mobius
// calcStat). The views use an empty Context, as the Mobius packets do.
type Context struct {
	Target any
	Skill  any
}

// Cond decides, on every calculation, whether a modifier applies (Mobius
// Condition). A nil Cond always applies.
type Cond interface {
	Applies(ctx Context) bool
}

// CondFunc adapts a function to Cond.
type CondFunc func(ctx Context) bool

// Applies calls f.
func (f CondFunc) Applies(ctx Context) bool { return f(ctx) }

// Always is a condition that always holds. It behaves like a nil Cond.
var Always Cond = CondFunc(func(Context) bool { return true })

// Modifier changes one stat: Op with Value at position Order of the stat's
// chain. Equal orders run in the order the modifiers were added, and the
// formula functions run before the modifiers added at OrderFormula, as in a
// Mobius Calculator where they were added first.
type Modifier struct {
	Stat  Stat
	Op    Op
	Order int
	Value float64
	Owner Owner
	Cond  Cond // nil: always applies
}

// NewModifier returns a modifier at the default order of op.
func NewModifier(stat Stat, op Op, value float64, owner Owner) Modifier {
	return Modifier{Stat: stat, Op: op, Order: op.DefaultOrder(), Value: value, Owner: owner}
}

func (m *Modifier) check() error {
	switch {
	case !m.Stat.Valid():
		return fmt.Errorf("stats: %w: modifier for unknown stat %d", ErrInvalid, m.Stat)
	case !m.Op.Valid():
		return fmt.Errorf("stats: %w: modifier of %s with unknown op %d", ErrInvalid, m.Stat, m.Op)
	case m.Order < 0:
		// Mobius replaces a negative order with the default one
		// (FuncTemplate); a modifier built here must say what it means.
		return fmt.Errorf("stats: %w: modifier of %s with negative order %d", ErrInvalid, m.Stat, m.Order)
	}
	return nil
}

// apply runs the modifier on v.
func (m *Modifier) apply(v float64, ctx Context) float64 {
	if m.Cond != nil && !m.Cond.Applies(ctx) {
		return v
	}
	switch m.Op {
	case OpSet:
		return m.Value
	case OpAdd, OpEnchant, OpEnchantHP:
		return v + m.Value
	case OpSub:
		return v - m.Value
	case OpMul:
		return mul(v, m.Value)
	case OpDiv:
		// FuncDiv means to skip a zero divisor (it catches the error and
		// logs it), but a Java double division never throws, so Mobius
		// would return an infinity. A zero divisor is skipped here.
		if m.Value == 0 {
			return v
		}
		return v / m.Value
	}
	// OpShare adds a share of the owner's stat, and only for servitors
	// (FuncShare checks effector.isServitor()).
	return v
}

// Modifiers holds a creature's modifiers, per stat in evaluation order. The
// zero value is empty and ready to use. Not safe for concurrent use.
type Modifiers struct {
	byStat map[Stat][]Modifier
	n      int
}

// Add inserts modifiers after all modifiers of the same stat with an order
// less than or equal to theirs (Mobius Calculator.addFunc). If one of them is
// invalid, none is added.
func (ms *Modifiers) Add(mods ...Modifier) error {
	for i := range mods {
		if err := mods[i].check(); err != nil {
			return err
		}
	}
	if len(mods) > 0 && ms.byStat == nil {
		ms.byStat = make(map[Stat][]Modifier)
	}
	for _, m := range mods {
		list := ms.byStat[m.Stat]
		at := sort.Search(len(list), func(i int) bool { return list[i].Order > m.Order })
		ms.byStat[m.Stat] = slices.Insert(list, at, m)
		ms.n++
	}
	return nil
}

// RemoveOwner removes every modifier of owner, keeping the order of the
// others, and returns how many it removed.
func (ms *Modifiers) RemoveOwner(owner Owner) int {
	removed := 0
	for stat, list := range ms.byStat {
		kept := slices.DeleteFunc(list, func(m Modifier) bool { return m.Owner == owner })
		if len(kept) == len(list) {
			continue
		}
		removed += len(list) - len(kept)
		if len(kept) == 0 {
			delete(ms.byStat, stat)
		} else {
			ms.byStat[stat] = kept
		}
	}
	ms.n -= removed
	return removed
}

// Of returns the modifiers of stat in evaluation order. The slice belongs to
// ms and must not be changed.
func (ms *Modifiers) Of(stat Stat) []Modifier { return ms.byStat[stat] }

// Len returns the number of modifiers.
func (ms *Modifiers) Len() int { return ms.n }

// formulas is implemented by the creature kinds: the formula functions Mobius
// puts in their calculators when they are created (addFuncsToNewCharacter for
// players, the standard NPC calculator for NPCs).
type formulas interface {
	hasFormula(stat Stat) bool
	formula(stat Stat, v float64) float64
}

// calcStat runs the calculator of one stat, like CreatureStat.calcStat: the
// modifiers ordered before the formulas, the formulas, then the rest. A
// stat with neither formulas nor modifiers returns init unchanged; otherwise
// a result of 0 or less becomes 1 for the stats in keepsPositive.
func calcStat(f formulas, mods []Modifier, stat Stat, init float64, ctx Context) float64 {
	has := f.hasFormula(stat)
	v, i := init, 0
	if has {
		for ; i < len(mods) && mods[i].Order < OrderFormula; i++ {
			v = mods[i].apply(v, ctx)
		}
		v = f.formula(stat, v)
	}
	for ; i < len(mods); i++ {
		v = mods[i].apply(v, ctx)
	}
	if v <= 0 && (has || len(mods) > 0) && stat.Valid() && keepsPositive[stat] {
		v = 1
	}
	return v
}
