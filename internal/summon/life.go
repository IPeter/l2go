package summon

import "time"

// LifeTick is the period of a servitor's life task (Mobius Servitor.run,
// scheduled every 5,000 ms from its spawn).
const LifeTick = 5 * time.Second

// lifeTickMillis is LifeTick in milliseconds, what each run takes off.
const lifeTickMillis = 5000

// Default upkeep intervals of a servitor without its own (Mobius Summon
// effect): 240 s, 60 s for a siege weapon.
const (
	DefaultConsumeInterval       = 240
	DefaultSiegeConsumeInterval  = 60
	DefaultLifeTime              = 3600
	DefaultServitorConsumeAmount = 1
)

// Life is a servitor's life and upkeep (Mobius Servitor's _lifeTime,
// _lifeTimeRemaining, _itemConsume, _consumeItemInterval and
// _consumeItemIntervalRemaining), in milliseconds.
type Life struct {
	// Total and Remaining are the life time and what is left of it.
	Total, Remaining int
	// ItemID and Count are the upkeep, every Interval; IntervalLeft counts
	// down to the next one.
	ItemID, Count          int
	Interval, IntervalLeft int
}

// NewLife returns the life of a servitor made by a Summon effect: lifeTime
// and interval in seconds, an interval of 0 taking Mobius's default for the
// race (siege weapons 60 s, others 240 s).
func NewLife(lifeTime, itemID, count, interval int, siegeWeapon bool) Life {
	if interval <= 0 {
		interval = DefaultConsumeInterval
		if siegeWeapon {
			interval = DefaultSiegeConsumeInterval
		}
	}
	l := Life{Total: lifeTime * 1000, ItemID: itemID, Count: count, Interval: interval * 1000}
	l.Remaining, l.IntervalLeft = l.Total, l.Interval
	return l
}

// LifeStep is what one run of the life task decided.
type LifeStep uint8

// The outcomes of a life task run.
const (
	// LifeGoesOn: nothing happens but the remaining time (SetSummonRemainTime).
	LifeGoesOn LifeStep = iota
	// LifeExpired: the life time ran out; the servitor passes away.
	LifeExpired
	// LifeConsume: the upkeep is due; the caller takes the items and calls
	// Consumed.
	LifeConsume
)

// Tick is one run of Mobius Servitor.run on a living, spawned servitor:
// 5 s go off the life time, which may run out; otherwise, with an upkeep,
// 5 s go off its interval, and the upkeep is due once that reaches 0 (with
// a positive item and count).
func (l *Life) Tick() LifeStep {
	l.Remaining -= lifeTickMillis
	if l.Remaining < 0 {
		return LifeExpired
	}
	if l.Interval > 0 {
		l.IntervalLeft -= lifeTickMillis
		if l.IntervalLeft <= 0 && l.Count > 0 && l.ItemID > 0 {
			return LifeConsume
		}
	}
	return LifeGoesOn
}

// Consumed ends a due upkeep: when the items were taken the interval starts
// over and the servitor stays (true); otherwise it leaves (false).
func (l *Life) Consumed(ok bool) bool {
	if ok {
		l.IntervalLeft = l.Interval
	}
	return ok
}
