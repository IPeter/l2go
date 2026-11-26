package game

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/olympiad"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The heroes (Mobius Hero): at a period's end the best noble of each third
// class becomes an unclaimed hero; at a Monument of Heroes it claims the
// status (the hero aura, the five hero skills, the hero items it may wear,
// hero voice, 1,000 reputation for a clan of level 5 or more). The status
// lasts until the next period's end, over relogs: the claimed heroes are
// the hero flag of their characters at selection. The player's flag itself
// is Mobius _hero (adminCharState.hero, which //sethero toggles).

// heroWordsMax is the longest hero words (RequestWriteHeroWords).
const heroWordsMax = 300

// publishHeroes copies the claimed heroes for reads off the loop.
func (f *olympiadFeature) publishHeroes() {
	ids := map[int32]bool{}
	if f.heroes != nil {
		for _, id := range f.heroes.IDs() {
			ids[id] = true
		}
	}
	f.heroIDs.Store(&ids)
}

// heroID reports whether the character is a claimed hero (Mobius
// Hero.isHero). Safe on any goroutine.
func (s *Server) heroID(charID int32) bool {
	m := s.olympiad.heroIDs.Load()
	return m != nil && (*m)[charID]
}

// loadHero is Mobius Player.restore's setHero(Hero.isHero(objectId)) at
// character selection, without packets: a claimed hero gets the flag and,
// on its base class, the hero skills.
func (s *Server) loadHero(p *Player) {
	if !s.heroID(p.ObjectID()) {
		return
	}
	s.adminState(p).hero = true
	if p.BaseClassID() == p.ClassID() {
		s.giveHeroSkills(p)
	}
}

// heroSkills returns the hero skills (Mobius getHeroSkillTree), in the
// data's order.
func (s *Server) heroSkills() []*skills.Skill {
	d, c := s.skills.data, s.skills.catalog
	if d == nil || c == nil {
		return nil
	}
	var out []*skills.Skill
	for _, l := range d.OtherSkillTree(data.TreeHero) {
		if sk, ok := c.Skill(l.SkillID, l.Level); ok {
			out = append(out, sk)
		}
	}
	return out
}

// giveHeroSkills adds the hero skills, not stored, without packets.
func (s *Server) giveHeroSkills(p *Player) {
	for _, sk := range s.heroSkills() {
		s.AddSkill(p, sk, false)
	}
}

// takeHeroSkills removes the hero skills, without packets.
func (s *Server) takeHeroSkills(p *Player) {
	for _, sk := range s.heroSkills() {
		s.RemoveSkill(p, sk.ID, false, true)
	}
}

// computeNewHeroes is Mobius Hero.computeNewHeroes: the old heroes online
// lose the status, take off and lose their hero items, and everyone's
// stored hero items are deleted; the candidates become unclaimed heroes.
func (s *Server) computeNewHeroes(candidates []olympiad.Candidate) {
	f := &s.olympiad
	clear(f.fights)
	clear(f.diary)
	former := f.heroes.ComputeNewHeroes(candidates)
	for _, id := range former {
		p, ok := s.Player(world.ObjectID(id))
		if !ok {
			continue
		}
		s.setHero(p, false)
		s.dropHeroItems(p)
		p.BroadcastUserInfo()
	}
	f.publishHeroes()
	s.purgeHeroItems()
	s.loadHeroPages(f.heroes.Current())
}

// dropHeroItems takes off and destroys p's hero items (Mobius
// computeNewHeroes: unEquipItemInSlot, destroyItem, InventoryUpdate).
func (s *Server) dropHeroItems(p *Player) {
	if p.inv.Inventory == nil {
		return
	}
	var heroItems []*inventory.Item
	for _, it := range p.inv.Items() {
		if it.Template().Hero {
			heroItems = append(heroItems, it)
		}
	}
	for _, it := range heroItems {
		if it.Equipped() {
			p.inv.UnequipItem(it)
		}
		s.DestroyItem(p, it, it.Count(), false)
	}
	if len(heroItems) > 0 {
		s.RequestSave(p)
	}
}

// purgeHeroItems is Mobius Hero.deleteItemsInDb: the hero items of every
// stored character are deleted, in the background. (Mobius spares GMs;
// the character store deletes by item only.)
func (s *Server) purgeHeroItems() {
	purger, ok := s.chars.(itemPurger)
	if !ok {
		s.log.Warn("game: the character store cannot delete the hero items")
		return
	}
	ids := make([]int32, len(olympiad.HeroItems))
	for i, id := range olympiad.HeroItems {
		ids[i] = int32(id) //nolint:gosec // item IDs
	}
	go func() {
		ctx, cancel := context.WithTimeout(s.background(), saveTimeout)
		defer cancel()
		n, err := purger.DeleteItemsByItemID(ctx, ids)
		if err != nil {
			s.log.Error("game: cannot delete the hero items", "err", err)
			return
		}
		s.log.Info("hero items deleted", "deleted", n)
	}()
}

// claimHero is Mobius Hero.claimHero: p becomes a claimed hero; a clan of
// level 5 or more gets the clan reputation and the news; p gets the hero
// status with its animation, a "hero gained" diary entry and empty words.
func (s *Server) claimHero(p *Player) {
	f := &s.olympiad
	f.heroes.Claim(p.ObjectID(), p.BaseClassID(), p.Name())
	if c := p.clan.clan; c != nil && c.Level >= 5 {
		pts := f.cfg.ClanHeroPoints
		s.AddClanReputation(c, pts)
		broadcastToClan(c, serverpackets.NewSystemMessageTextPacket("Clan member "+p.Name()+" was named a hero. "+
			strconv.Itoa(pts)+" points have been added to your clan's reputation score."))
	}
	s.setHero(p, true)
	s.refreshPledgeClass(p)
	s.world.Broadcast(p, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{ObjectID: p.ObjectID(), ActionID: olympiad.HeroSocialAction}), true)
	p.BroadcastUserInfo()
	f.publishHeroes()
	s.loadHeroPages([]olympiad.Hero{{CharID: p.ObjectID()}})
	s.addHeroDiary(p.ObjectID(), olympiad.DiaryHeroGained, 0)
	s.saveOlympiad()
}

// revokeHero takes the hero status from p (the admin page's Hero off; not
// in Mobius).
func (s *Server) revokeHero(p *Player) {
	f := &s.olympiad
	f.heroes.Revoke(p.ObjectID())
	s.setHero(p, false)
	s.refreshPledgeClass(p)
	p.BroadcastUserInfo()
	f.publishHeroes()
	s.saveOlympiad()
}

// heroListPacket is ExHeroList: the current heroes with their clan and
// alliance.
func (s *Server) heroListPacket() []byte {
	var list []serverpackets.HeroListEntry
	for _, h := range s.olympiad.heroes.Current() {
		e := serverpackets.HeroListEntry{Name: h.Name, ClassID: int32(h.ClassID), Count: int32(h.Count)} //nolint:gosec // small
		if c := s.clanOfCharacter(h.CharID); c != nil {
			e.ClanName, e.ClanCrestID = c.Name, c.CrestID
			if c.AllyID != 0 {
				e.AllyName, e.AllyCrestID = c.AllyName, c.AllyCrestID
			}
		}
		list = append(list, e)
	}
	return serverpackets.NewExHeroListPacket(list)
}

// clanOfCharacter returns the clan a character is a member of, online or
// not, nil for none.
func (s *Server) clanOfCharacter(charID int32) *Clan {
	if p, ok := s.Player(world.ObjectID(charID)); ok {
		return p.clan.clan
	}
	for _, c := range sortedClans(s.clan.clans) {
		if c.member(charID) != nil {
			return c
		}
	}
	return nil
}

// handleWriteHeroWords handles RequestWriteHeroWords (D0 0C): a hero's
// words, at most 300 letters.
func (s *Server) handleWriteHeroWords(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestWriteHeroWords(body)
	if err != nil || !p.IsHero() || len([]rune(req.Words)) > heroWordsMax {
		return
	}
	if s.olympiad.heroes.SetMessage(p.ObjectID(), req.Words) {
		s.saveOlympiad()
	}
}

// heroRaidKilled is Mobius Hero.setRBkilled for a noble of the killing
// party: the diary entry is stored, and shown when it is a current hero's.
func (s *Server) heroRaidKilled(p *Player, npcID int) {
	if !p.IsNoble() || !s.olympiad.cfg.Enabled {
		return
	}
	s.addHeroDiary(p.ObjectID(), olympiad.DiaryRaidKilled, npcID)
}

// HeroCastleTaken is Mobius Hero.setCastleTaken: a hero's clan leader took
// a castle; the sieges call it. Loop-only.
func (s *Server) HeroCastleTaken(p *Player, castleID int) {
	s.addHeroDiary(p.ObjectID(), olympiad.DiaryCastleTaken, castleID)
}

// addHeroDiary stores a diary entry and adds it to a current hero's page.
func (s *Server) addHeroDiary(charID int32, action, param int) {
	f := &s.olympiad
	e := storage.HeroDiaryEntry{CharID: charID, Time: s.Now().UnixMilli(), Action: int32(action), Param: int32(param)} //nolint:gosec // small
	if _, ok := f.diary[charID]; ok {
		f.diary[charID] = append(f.diary[charID], e)
	}
	if f.store != nil {
		f.writer.add(f.store, func(ctx context.Context, store storage.OlympiadStore) error {
			return store.AddHeroDiary(ctx, e)
		})
	}
}

// loadHeroPages loads the fights and the diary of heroes in the background
// (Mobius loadFights and loadDiary): the fights before the first of this
// month.
func (s *Server) loadHeroPages(heroes []olympiad.Hero) {
	f := &s.olympiad
	if len(heroes) == 0 {
		return
	}
	for _, h := range heroes {
		if _, ok := f.diary[h.CharID]; !ok {
			f.diary[h.CharID] = nil
		}
		if _, ok := f.fights[h.CharID]; !ok {
			f.fights[h.CharID] = nil
		}
	}
	store := f.store
	if store == nil {
		return
	}
	// Calendar.set(DAY_OF_MONTH, 1), HOUR_OF_DAY 0, MINUTE 0, MILLISECOND 0:
	// the seconds stay, as Mobius leaves them.
	t := s.calendar.In(s.Now())
	from := time.Date(t.Year(), t.Month(), 1, 0, 0, t.Second(), 0, t.Location())
	ids := make([]int32, len(heroes))
	for i, h := range heroes {
		ids[i] = h.CharID
	}
	go func() {
		ctx, cancel := context.WithTimeout(s.background(), olympiadLoadTimeout)
		defer cancel()
		fights := map[int32][]storage.OlympiadFight{}
		diary := map[int32][]storage.HeroDiaryEntry{}
		for _, id := range ids {
			fl, err := store.OlympiadFights(ctx, id, from.UnixMilli())
			if err != nil {
				s.log.Warn("game: cannot load a hero's fights", "charId", id, "err", err)
			}
			fights[id] = fl
			dl, err := store.HeroDiary(ctx, id)
			if err != nil {
				s.log.Warn("game: cannot load a hero's diary", "charId", id, "err", err)
			}
			diary[id] = dl
		}
		s.loop.Post(func() {
			for id, fl := range fights {
				if _, ok := f.fights[id]; ok {
					f.fights[id] = fl
				}
			}
			for id, dl := range diary {
				if cur, ok := f.diary[id]; ok {
					// Entries added since the load started stay.
					var newer []storage.HeroDiaryEntry
					for _, e := range cur {
						if !slices.Contains(dl, e) {
							newer = append(newer, e)
						}
					}
					f.diary[id] = append(dl, newer...)
				}
			}
		})
	}()
}
