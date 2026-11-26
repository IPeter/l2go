package residence

import "slices"

// ClanType is a clan's side in a siege (Mobius SiegeClanType).
type ClanType int8

// The sides, in Mobius's order.
const (
	Owner ClanType = iota
	Defender
	Attacker
	DefenderPending
)

// Stored registration types (Mobius siege_clans.type and the Siege
// constants OWNER -1, DEFENDER 0, ATTACKER 1, DEFENDER_NOT_APPROVED 2).
const (
	RegOwner           = -1
	RegDefender        = 0
	RegAttacker        = 1
	RegDefenderPending = 2
)

// SiegeClan is a registered clan of a siege (Mobius SiegeClan): its side and
// the object IDs of its headquarters (siege flags).
type SiegeClan struct {
	ClanID int32
	Type   ClanType
	Flags  []int32
}

// RemoveFlag forgets a flag; it reports whether the clan had it.
func (c *SiegeClan) RemoveFlag(id int32) bool {
	i := slices.Index(c.Flags, id)
	if i < 0 {
		return false
	}
	c.Flags = slices.Delete(c.Flags, i, i+1)
	return true
}

// Registration is one stored registration (a Mobius siege_clans row).
type Registration struct {
	ClanID int32
	Type   int // RegDefender, RegAttacker or RegDefenderPending
}

// Registry is the clans of one siege (the lists of Mobius Siege and
// ClanHallSiegeEngine): attackers, defenders (the owner first) and the
// defenders waiting for the owner's approval. Lists keep the order clans
// were added.
type Registry struct {
	attackers, defenders, waiting []*SiegeClan
}

// Attackers returns the attacker clans. The slice must not be modified.
func (r *Registry) Attackers() []*SiegeClan { return r.attackers }

// Defenders returns the defender clans, the owner among them.
func (r *Registry) Defenders() []*SiegeClan { return r.defenders }

// Waiting returns the defenders waiting for approval.
func (r *Registry) Waiting() []*SiegeClan { return r.waiting }

func find(list []*SiegeClan, id int32) *SiegeClan {
	for _, c := range list {
		if c.ClanID == id {
			return c
		}
	}
	return nil
}

func remove(list []*SiegeClan, c *SiegeClan) []*SiegeClan {
	if i := slices.Index(list, c); i >= 0 {
		return slices.Delete(list, i, i+1)
	}
	return list
}

// Attacker returns the attacker clan with the ID, or nil.
func (r *Registry) Attacker(id int32) *SiegeClan { return find(r.attackers, id) }

// Defender returns the defender clan with the ID, or nil.
func (r *Registry) Defender(id int32) *SiegeClan { return find(r.defenders, id) }

// WaitingClan returns the waiting defender with the ID, or nil.
func (r *Registry) WaitingClan(id int32) *SiegeClan { return find(r.waiting, id) }

// IsAttacker is Siege.checkIsAttacker.
func (r *Registry) IsAttacker(id int32) bool { return id != 0 && r.Attacker(id) != nil }

// IsDefender is Siege.checkIsDefender.
func (r *Registry) IsDefender(id int32) bool { return id != 0 && r.Defender(id) != nil }

// IsWaiting is Siege.checkIsDefenderWaiting.
func (r *Registry) IsWaiting(id int32) bool { return id != 0 && r.WaitingClan(id) != nil }

// Registered reports whether the clan is on any list.
func (r *Registry) Registered(id int32) bool {
	return r.IsAttacker(id) || r.IsDefender(id) || r.IsWaiting(id)
}

// AddAttacker adds a clan to the attackers.
func (r *Registry) AddAttacker(id int32) {
	r.attackers = append(r.attackers, &SiegeClan{ClanID: id, Type: Attacker})
}

// AddDefender adds a clan to the defenders with type t (Owner or Defender).
func (r *Registry) AddDefender(id int32, t ClanType) {
	r.defenders = append(r.defenders, &SiegeClan{ClanID: id, Type: t})
}

// AddWaiting adds a clan to the defenders waiting for approval.
func (r *Registry) AddWaiting(id int32) {
	r.waiting = append(r.waiting, &SiegeClan{ClanID: id, Type: DefenderPending})
}

// Remove takes the clan off every list (Siege.removeSiegeClan reloads the
// lists without its row).
func (r *Registry) Remove(id int32) {
	if c := r.Attacker(id); c != nil {
		r.attackers = remove(r.attackers, c)
	}
	if c := r.Defender(id); c != nil {
		r.defenders = remove(r.defenders, c)
	}
	if c := r.WaitingClan(id); c != nil {
		r.waiting = remove(r.waiting, c)
	}
}

// Approve moves a waiting clan to the defenders (approveSiegeDefenderClan).
func (r *Registry) Approve(id int32) bool {
	c := r.WaitingClan(id)
	if c == nil {
		return false
	}
	r.waiting = remove(r.waiting, c)
	r.AddDefender(id, Defender)
	return true
}

// ClearWaiting forgets the waiting defenders (clearSiegeWaitingClan).
func (r *Registry) ClearWaiting() { r.waiting = nil }

// Clear forgets every clan (clearSiegeClan).
func (r *Registry) Clear() { r.attackers, r.defenders, r.waiting = nil, nil, nil }

// Load is Mobius loadSiegeClan: the lists are rebuilt from the stored rows,
// the owner (if any) first among the defenders.
func (r *Registry) Load(owner int32, rows []Registration) {
	r.Clear()
	if owner > 0 {
		r.AddDefender(owner, Owner)
	}
	for _, row := range rows {
		switch row.Type {
		case RegDefender:
			r.AddDefender(row.ClanID, Defender)
		case RegAttacker:
			r.AddAttacker(row.ClanID)
		case RegDefenderPending:
			r.AddWaiting(row.ClanID)
		}
	}
}

// Rows returns the stored registrations of the lists: every clan but the
// owner, attackers first.
func (r *Registry) Rows() []Registration {
	var rows []Registration
	for _, c := range r.attackers {
		if c.Type != Owner {
			rows = append(rows, Registration{ClanID: c.ClanID, Type: RegAttacker})
		}
	}
	for _, c := range r.defenders {
		if c.Type != Owner {
			rows = append(rows, Registration{ClanID: c.ClanID, Type: RegDefender})
		}
	}
	for _, c := range r.waiting {
		rows = append(rows, Registration{ClanID: c.ClanID, Type: RegDefenderPending})
	}
	return rows
}

// FlagOwner returns the attacker or defender clan that owns a flag, or nil.
func (r *Registry) FlagOwner(flag int32) *SiegeClan {
	for _, list := range [][]*SiegeClan{r.attackers, r.defenders} {
		for _, c := range list {
			if slices.Contains(c.Flags, flag) {
				return c
			}
		}
	}
	return nil
}

// RegisterCheck is what Mobius Siege.checkIfCanRegister looks at.
type RegisterCheck struct {
	RegistrationOver bool
	InProgress       bool
	ClanID           int32 // 0 without a clan
	ClanLevel        int
	ClanCastleID     int32
	OwnerID          int32
	// Registered is SiegeManager.checkIsRegistered: the clan owns a castle or
	// has a row for this castle.
	Registered bool
	// SameDay is checkIfAlreadyRegisteredForSameDay: the clan is on a list
	// of another siege on the same weekday.
	SameDay      bool
	Attacker     bool // registering as attacker, else as defender
	MinLevel     int
	MaxAttackers int
	MaxDefenders int
}

// CheckRegister is Mobius Siege.checkIfCanRegister: 0 and true when the clan
// may register, else the message the player gets.
func (r *Registry) CheckRegister(c RegisterCheck) (msg int, ok bool) {
	switch {
	case c.RegistrationOver:
		return MsgRegistrationDeadline, false
	case c.InProgress:
		return MsgNotTimeToRegister, false
	case c.ClanID == 0 || c.ClanLevel < c.MinLevel:
		return MsgClanLevelTooLow, false
	case c.ClanID == c.OwnerID:
		return MsgOwnerIsDefender, false
	case c.ClanCastleID > 0:
		return MsgOwnsAnotherCastle, false
	case c.Registered:
		return MsgAlreadyRequested, false
	case c.SameDay:
		return MsgAnotherSiegeSameDay, false
	case c.Attacker && len(r.attackers) >= c.MaxAttackers:
		return MsgNoMoreAttackers, false
	case !c.Attacker && len(r.defenders)+len(r.waiting) >= c.MaxDefenders:
		return MsgNoMoreDefenders, false
	}
	return 0, true
}

// CanSave is the capacity check of Mobius saveSiegeClan: a defender side
// below MaxDefenders (defenders and waiting) or an attacker side below
// MaxAttackers.
func (r *Registry) CanSave(attacker bool, maxAttackers, maxDefenders int) bool {
	if attacker {
		return len(r.attackers) < maxAttackers
	}
	return len(r.defenders)+len(r.waiting) < maxDefenders
}

// MidVictory is the registry part of Mobius Siege.midVictory, after newOwner
// engraved the ruler. allyOf gives a clan's alliance ID; allies lists the
// clans of the new owner's alliance (ClanTable.getClanAllies, empty without
// one). It reports whether the siege ends at once: without defenders and with
// a single attacker, or with only the new owner's alliance attacking. Else
// every defender becomes an attacker, the new owner the owner, and the
// attackers of its alliance defenders.
func (r *Registry) MidVictory(newOwner int32, allyOf func(int32) int32, allies []int32) (endNow bool) {
	makeOwner := func() {
		if sc := r.Attacker(newOwner); sc != nil {
			r.attackers = remove(r.attackers, sc)
			sc.Type = Owner
			r.defenders = append(r.defenders, sc)
		}
	}
	if len(r.defenders) == 0 && len(r.attackers) == 1 {
		makeOwner()
		return true
	}
	if newOwner <= 0 {
		return false
	}
	allyID := allyOf(newOwner)
	if len(r.defenders) == 0 && allyID != 0 {
		same := true
		for _, sc := range r.attackers {
			if allyOf(sc.ClanID) != allyID {
				same = false
			}
		}
		if same {
			makeOwner()
			return true
		}
	}
	for _, sc := range slices.Clone(r.defenders) {
		r.defenders = remove(r.defenders, sc)
		sc.Type = Attacker
		r.attackers = append(r.attackers, sc)
	}
	makeOwner()
	for _, id := range allies {
		if sc := r.Attacker(id); sc != nil {
			r.attackers = remove(r.attackers, sc)
			sc.Type = Defender
			r.defenders = append(r.defenders, sc)
		}
	}
	return false
}

// RemoveDefenderFlags forgets the defenders' flags and returns them
// (removeDefenderFlags).
func (r *Registry) RemoveDefenderFlags() []int32 {
	var out []int32
	for _, sc := range r.defenders {
		out = append(out, sc.Flags...)
		sc.Flags = nil
	}
	return out
}

// RemoveFlags forgets every flag and returns them (removeFlags).
func (r *Registry) RemoveFlags() []int32 {
	out := r.RemoveDefenderFlags()
	for _, sc := range r.attackers {
		out = append(out, sc.Flags...)
		sc.Flags = nil
	}
	return out
}
