package game

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
)

// SkillList, SkillCoolTime, the reuse stamps and ExRegenMax.

// skillListDelay is how long SkillList waits for more changes (Mobius
// sendSkillList schedules it 300 ms later).
const skillListDelay = 300 * time.Millisecond

// skillReuse is a reuse stamp (Mobius TimeStamp): the skill level, its full
// reuse delay and when the reuse ends.
type skillReuse struct {
	id, level int
	delay     time.Duration
	end       time.Time
}

// reuseStamps are a creature's reuse stamps by reuse hash, in the order of
// Mobius's map. The zero value is empty.
type reuseStamps struct {
	byHash map[int]skillReuse
	order  chmOrder
}

// SendSkillList sends p its SkillList 300 ms from now, once however often
// it is called meanwhile (Mobius sendSkillList). Loop-only.
func (p *Player) SendSkillList() {
	a := p.skill
	if a.listTimer != 0 || p.srv == nil {
		return
	}
	a.listTimer = p.srv.loop.After(skillListDelay, func() {
		a.listTimer = 0
		if p.state == stateInGame {
			p.Send(skillListPacket(p))
		}
	})
}

// skillListPacket builds SkillList (Mobius sendSkillList's task): every
// known skill in map order with its display ID and level; an active skill
// is disabled while p wears formal wear, and a rider's but its mount's
// (mount.go). Cursed weapons do not disable skills here yet.
func skillListPacket(p *Player) []byte {
	known := p.skill.known.all()
	formal := p.wearsFormalWear()
	entries := make([]serverpackets.SkillListEntry, len(known))
	for i, sk := range known {
		entries[i] = serverpackets.SkillListEntry{
			ID:       int32(sk.DisplayID),    //nolint:gosec // skill IDs are small
			Level:    int32(sk.DisplayLevel), //nolint:gosec // skill levels are small
			Passive:  sk.IsPassive(),
			Disabled: p.clanSkillDisabled(sk) || (!sk.IsPassive() && (formal || !mountSkillAllowed(p, sk))),
		}
	}
	return serverpackets.NewSkillListPacket(entries)
}

// wearsFormalWear is Mobius isWearingFormalWear: the chest item is an
// all-dress.
func (p *Player) wearsFormalWear() bool {
	if p.inv.Inventory == nil {
		return false
	}
	it := p.inv.Paperdoll(storage.PaperdollChest)
	return it != nil && it.Template().Part() == data.BodyPartAlldress
}

// handleRequestSkillList answers RequestSkillList (0x3F) with SkillList
// (Mobius RequestSkillList: sendSkillList).
func (s *Server) handleRequestSkillList(p *Player, _ []byte) { p.SendSkillList() }

// addReuse records a reuse stamp of sk ending at end.
func (b *skillBase) addReuse(sk *skills.Skill, delay time.Duration, end time.Time) {
	r := &b.reuse
	if r.byHash == nil {
		r.byHash = make(map[int]skillReuse)
	}
	h := sk.ReuseHash()
	if _, ok := r.byHash[h]; !ok {
		r.order.put(h)
	}
	r.byHash[h] = skillReuse{id: sk.ID, level: sk.Level, delay: delay, end: end}
}

// reuseLeft returns the time left of sk's reuse stamp, 0 when it has none or
// it has passed.
func (b *skillBase) reuseLeft(sk *skills.Skill, now time.Time) time.Duration {
	r, ok := b.reuse.byHash[sk.ReuseHash()]
	if !ok || !r.end.After(now) {
		return 0
	}
	return r.end.Sub(now)
}

// activeReuse returns the stamps that have not passed, in map order.
func (b *skillBase) activeReuse(now time.Time) []skillReuse {
	var out []skillReuse
	for _, h := range b.reuse.order.keys() {
		if r := b.reuse.byHash[h]; r.end.After(now) {
			out = append(out, r)
		}
	}
	return out
}

// AddSkillReuse puts sk under reuse for delay (Mobius addTimeStamp): until
// then SkillReuseLeft is positive, SkillCoolTime lists it and a save keeps
// it. Reuse is kept per skill ID and level, so a new level is ready at once.
// Casting (Phase 7.3) calls it for reuses over one second. Loop-only.
func (s *Server) AddSkillReuse(c Creature, sk *skills.Skill, delay time.Duration) {
	if b := s.skillBaseOf(c); b != nil && sk != nil && delay > 0 {
		b.addReuse(sk, delay, s.Now().Add(delay))
	}
}

// SkillReuseLeft returns how long sk stays under reuse for c, 0 when it is
// ready (Mobius isSkillDisabled by the time stamp). Loop-only.
func (s *Server) SkillReuseLeft(c Creature, sk *skills.Skill) time.Duration {
	b := s.skillBaseOf(c)
	if b == nil || sk == nil {
		return 0
	}
	return b.reuseLeft(sk, s.Now())
}

// ClearSkillReuse ends every reuse of c (Mobius resetTimeStamps, for
// //removereuse); the caller sends SkillCoolTime. Loop-only.
func (s *Server) ClearSkillReuse(c Creature) {
	if b := s.skillBaseOf(c); b != nil {
		b.reuse = reuseStamps{}
	}
}

// skillBaseOf returns the adapter base of a player or NPC.
func (s *Server) skillBaseOf(c Creature) *skillBase {
	switch v := c.(type) {
	case *Player:
		if v != nil && v.skill != nil {
			return &v.skill.skillBase
		}
	case *Npc:
		if v != nil && v.skill != nil {
			return &v.skill.skillBase
		}
	}
	return nil
}

// SkillCoolTimePacket builds p's SkillCoolTime (Mobius SkillCoolTime): the
// reuse stamps that have not passed. Loop-only.
func (s *Server) SkillCoolTimePacket(p *Player) []byte {
	now := s.Now()
	stamps := p.skill.activeReuse(now)
	entries := make([]serverpackets.SkillCoolTimeEntry, len(stamps))
	for i, r := range stamps {
		entries[i] = serverpackets.SkillCoolTimeEntry{
			SkillID:    int32(r.id),    //nolint:gosec // skill IDs are small
			SkillLevel: int32(r.level), //nolint:gosec // skill levels are small
			Reuse:      r.delay,
			Remaining:  r.end.Sub(now),
		}
	}
	return serverpackets.NewSkillCoolTimePacket(entries)
}

// exRegenMaxSubOpcode is the sub-opcode of ExRegenMax (FE 01).
const exRegenMaxSubOpcode = 0x01

// exRegenMaxPacket builds ExRegenMax (Mobius ExRegenMax(count, time,
// hpRegen)): 1, the duration, the tick period and 90 % of the amount per
// tick. The serverpackets package has no codec for it yet.
func exRegenMaxPacket(duration, ticks int, amount float64) []byte {
	w := packets.NewWriter(23)
	w.WriteUint8(serverpackets.OpExPacket)
	w.WriteUint16(exRegenMaxSubOpcode)
	w.WriteInt32(1)
	w.WriteInt32(int32(duration)) //nolint:gosec // seconds of an effect
	w.WriteInt32(int32(ticks))    //nolint:gosec // a few ticks
	w.WriteFloat64(amount * 0.90)
	return w.Bytes()
}
