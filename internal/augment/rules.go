package augment

import "github.com/IPeter/l2go/internal/data"

// System messages of augmentation (Mobius SystemMessageId).
const (
	MsgNotSuitable         = 1960 // "This is not a suitable item."
	MsgGemstoneCount       = 1961 // "Gemstone quantity is incorrect."
	MsgRemoveNotAugmented  = 1964 // "Augmentation removal can only be done on an augmented item."
	MsgAlreadyAugmented    = 1970 // "Once an item is augmented, it cannot be augmented again."
	MsgStore               = 1972 // "You cannot augment items while a private store or private workshop is in operation."
	MsgDead                = 1974 // "You cannot augment items while dead."
	MsgTrade               = 1975 // "You cannot augment items while engaged in trade activities."
	MsgParalyzed           = 1976 // "You cannot augment items while paralyzed."
	MsgFishing             = 1977 // "You cannot augment items while fishing."
	MsgSitting             = 1978 // "You cannot augment items while sitting down."
	MsgInappropriateRefine = 2001 // "Augmentation failed due to inappropriate conditions."
	MsgAugmentedNoDiscard  = 2011 // "The augmented item cannot be discarded."
)

// Player is the state of a player that augmentation checks
// (AbstractRefinePacket.isValid(Player)).
type Player struct {
	Level int
	// StoreMode is set while a private store or workshop runs.
	StoreMode bool
	// Trading is set while a trade window is open (getActiveTradeList).
	Trading               bool
	Dead                  bool
	Paralyzed             bool
	Fishing               bool
	Sitting               bool
	CursedWeapon          bool
	Enchanting            bool
	ProcessingTransaction bool
}

// CheckPlayer is AbstractRefinePacket.isValid(Player): it returns false
// when the player may not augment now, with the system message Mobius
// sends, or 0 for the refusals it sends none for (a cursed weapon,
// enchanting, a request or trade in progress).
func CheckPlayer(p Player) (msg int, ok bool) {
	switch {
	case p.StoreMode:
		return MsgStore, false
	case p.Trading:
		return MsgTrade, false
	case p.Dead:
		return MsgDead, false
	case p.Paralyzed:
		return MsgParalyzed, false
	case p.Fishing:
		return MsgFishing, false
	case p.Sitting:
		return MsgSitting, false
	case p.CursedWeapon, p.Enchanting, p.ProcessingTransaction:
		return 0, false
	}
	return 0, true
}

// Item is an item instance as augmentation checks it.
type Item struct {
	Template *data.Item
	// Owned is set when the player owns the item (Mobius getOwnerId).
	Owned bool
	// Carried and Worn are its location: the inventory or the paperdoll.
	Carried, Worn bool
	// Augmentation is the item's augmentation, 0 for none.
	Augmentation Augmentation
	// Shadow is set for an item with mana (isShadowItem).
	Shadow bool
}

// Augmented reports whether the item has an augmentation.
func (it Item) Augmented() bool { return it.Augmentation != 0 }

// TargetValid is AbstractRefinePacket.isValid(Player, Item) without the
// player part: an owned weapon (but a fishing rod or one without a type)
// or ring, earring or necklace of grade C or better, carried or worn, not
// augmented, hero, shadow, common, time-limited, a PvP item (unless
// AllowPvP) or blacklisted.
func (c Config) TargetValid(it Item) bool {
	t := it.Template
	switch {
	case t == nil || !it.Owned:
		return false
	case it.Augmented(), t.Hero, it.Shadow, t.Common, t.Type == data.ItemTypeEtc, t.Time > 0:
		return false
	case t.PvP && !c.AllowPvP:
		return false
	case t.Grade < data.CrystalC:
		return false
	case !it.Carried && !it.Worn:
		return false
	}
	switch {
	case t.Weapon != nil:
		if t.Weapon.Type == data.WeaponNone || t.Weapon.Type == data.WeaponFishingrod {
			return false
		}
	case t.Armor != nil:
		switch t.Part() {
		case data.BodyPartLRFinger, data.BodyPartLREar, data.BodyPartNeck:
		default:
			return false
		}
	default:
		return false
	}
	return !c.Blacklisted(t.ID)
}

// RefinerValid is the life stone part of AbstractRefinePacket.isValid(Player,
// Item, Item): an owned life stone in the inventory (not worn) whose player
// level the player has. It returns the life stone.
func (t *Tables) RefinerValid(level int, refiner Item) (data.LifeStone, bool) {
	if refiner.Template == nil || !refiner.Owned || !refiner.Carried || refiner.Worn {
		return data.LifeStone{}, false
	}
	ls, ok := t.LifeStone(refiner.Template.ID)
	if !ok || level < ls.PlayerLevel {
		return data.LifeStone{}, false
	}
	return ls, true
}

// GemstoneValid is the gemstone part of AbstractRefinePacket.isValid(Player,
// Item, Item, Item): an owned stack in the inventory of the gemstone the
// target's grade needs, with at least as many as it needs.
func (t *Tables) GemstoneValid(target *data.Item, ls data.LifeStone, gem Item, count int64) bool {
	if gem.Template == nil || !gem.Owned || !gem.Carried || gem.Worn {
		return false
	}
	if t.GemstoneID(target.Grade) != gem.Template.ID {
		return false
	}
	return int64(t.GemstoneCount(target.Grade, ls.Grade)) <= count
}

// Request is a step of the augmentation window: the target alone
// (RequestConfirmTargetItem), with a life stone (RequestConfirmRefinerItem)
// and with gemstones (RequestConfirmGemStone, RequestRefine).
type Request struct {
	Player Player
	Target Item
	// Refiner and Gemstone are nil for the steps without them; GemCount is
	// the size of the gemstone stack.
	Refiner, Gemstone *Item
	GemCount          int64
}

// Result is the outcome of Check.
type Result struct {
	// Valid is set when every check passed.
	Valid bool
	// PlayerMsg is the message of a refused player state, 0 for none
	// (Mobius sends it from isValid(Player)).
	PlayerMsg int
	// LifeStone is the refiner's life stone when it passed.
	LifeStone data.LifeStone
}

// Check runs the checks of AbstractRefinePacket.isValid for the parts the
// request names, in Mobius's order: the player, the target, the life stone
// and the gemstones.
func (t *Tables) Check(cfg Config, req Request) Result {
	msg, ok := CheckPlayer(req.Player)
	if !ok {
		return Result{PlayerMsg: msg}
	}
	if !cfg.TargetValid(req.Target) {
		return Result{}
	}
	if req.Refiner == nil {
		return Result{Valid: true}
	}
	ls, ok := t.RefinerValid(req.Player.Level, *req.Refiner)
	if !ok {
		return Result{}
	}
	if req.Gemstone != nil && !t.GemstoneValid(req.Target.Template, ls, *req.Gemstone, req.GemCount) {
		return Result{}
	}
	return Result{Valid: true, LifeStone: ls}
}

// CancelCheck is the outcome of CheckCancel.
type CancelCheck struct {
	// Valid is set when the augmentation can be removed for Fee.
	Valid bool
	// PlayerMsg is the message of a refused player state; Msg the message
	// of a refused item; both 0 for none.
	PlayerMsg int
	Msg       int
	Fee       int64
}

// CheckCancel runs the checks of RequestConfirmCancelItem and
// RequestRefineCancel in Mobius's order: the player state, the item is
// augmented, not a PvP item (unless AllowPvP), and has a removal fee.
func (t *Tables) CheckCancel(cfg Config, p Player, it Item) CancelCheck {
	if msg, ok := CheckPlayer(p); !ok {
		return CancelCheck{PlayerMsg: msg}
	}
	if !it.Augmented() {
		return CancelCheck{Msg: MsgRemoveNotAugmented}
	}
	if it.Template.PvP && !cfg.AllowPvP {
		return CancelCheck{Msg: MsgNotSuitable}
	}
	fee, ok := t.CancelFee(it.Template.Grade, it.Template.CrystalCount)
	if !ok {
		return CancelCheck{}
	}
	return CancelCheck{Valid: true, Fee: fee}
}

// CheckRefineCancel runs the checks of RequestRefineCancel, which come in
// another order than those of the confirmation: the item is augmented
// (else Msg), then the player state (PlayerMsg) and the PvP rule (no
// message), then the fee.
func (t *Tables) CheckRefineCancel(cfg Config, p Player, it Item) CancelCheck {
	if !it.Augmented() {
		return CancelCheck{Msg: MsgRemoveNotAugmented}
	}
	if msg, ok := CheckPlayer(p); !ok {
		return CancelCheck{PlayerMsg: msg}
	}
	if it.Template.PvP && !cfg.AllowPvP {
		return CancelCheck{}
	}
	fee, ok := t.CancelFee(it.Template.Grade, it.Template.CrystalCount)
	if !ok {
		return CancelCheck{}
	}
	return CancelCheck{Valid: true, Fee: fee}
}

// Tradable is Item.isTradeable, isDropable and isSellable for an augmented
// item: AllowTrade decides.
func (c Config) Tradable(it Item) bool { return !it.Augmented() || c.AllowTrade }

// Destroyable is Item.isDestroyable for an augmented item: AllowDestroy
// decides.
func (c Config) Destroyable(it Item) bool { return !it.Augmented() || c.AllowDestroy }
