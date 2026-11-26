package game

import (
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// Riding a strider pet (Mobius Player.mountPlayer, action 38, and
// mount(Summon)); the rest of riding is in mount.go.

// mountPlayer is action 38 (Mount/Dismount, Mobius Player.mountPlayer): next
// to a strider pet a player on foot gets on it, unless dead, the strider is
// dead, in battle or rooted, the player is in battle, sits, fishes, holds a
// cursed weapon (no message), the strider is hungry or more than 200 away
// (each with ActionFailed and its message). A rider gets off, except from a
// wyvern over a no-landing zone (GMs may) or a hungry mount.
func (s *Server) mountPlayer(p *Player, sm *Summon) bool {
	if sm != nil && s.summonMountable(sm) && !s.mounted(p) && !p.skill.el.IsAffected(skills.FlagBetrayed) {
		refuse := func(msg ...serverpackets.SystemMessageID) bool {
			p.SendActionFailed()
			for _, m := range msg {
				p.SendSystemMessage(m)
			}
			return false
		}
		n := sm.npc
		switch {
		case p.IsDead():
			return refuse(mountMsgStriderDeadRider)
		case n.Dead:
			return refuse(mountMsgDeadStrider)
		case s.summonInCombat(sm) || (n.skill != nil && n.skill.el.IsAffected(skills.FlagRooted)):
			return refuse(mountMsgStriderInBattle)
		case s.InCombat(p):
			return refuse(mountMsgRiderInBattle)
		case p.Sitting:
			return refuse(mountMsgOnlyStanding)
		case s.Fishing(p):
			return refuse(mountMsgFishing)
		case p.CursedWeaponEquipped():
			return refuse()
		case s.petHungry(sm):
			return refuse(mountMsgHungryStrider)
		case !s.mountInRange(p, sm):
			return refuse(mountMsgTooFarFromStrider)
		}
		if s.mountPet(p, sm) {
			p.SendSkillList()
		}
		return true
	}
	if s.mounted(p) {
		switch {
		case p.mount.typ == serverpackets.MountWyvern && p.Zones.Inside(ZoneNoLanding) && !p.IsGM():
			p.SendActionFailed()
			p.SendSystemMessage(mountMsgNotAllowedToDismount)
			return false
		case s.mountHungry(p):
			p.SendActionFailed()
			p.SendSystemMessage(mountMsgHungryStrider)
			return false
		}
		s.dismount(p)
	}
	return true
}

// mountInRange is checkIfInRange(200, player, pet, true): within 200 beyond
// both template collision radii, in 3D.
func (s *Server) mountInRange(p *Player, sm *Summon) bool {
	r := float64(mountRange + creatureCollision(p) + creatureCollision(sm.npc))
	return p.Position().Distance3D(sm.npc.Position()) <= r
}

// mountPet is Mobius mount(Summon): in a siege zone only the castle's lord
// gets on a strider ("You cannot mount during a siege." otherwise); the
// weapon and shield come off (false when they cannot), the toggles end, the
// player rides the pet at its level with its feed and collar, everyone sees
// it, and the pet goes back into its collar.
func (s *Server) mountPet(p *Player, sm *Summon) bool {
	npcID := sm.npc.Template.ID
	if !mountAllowDuringSiege && p.Zones.Inside(ZoneSiege) {
		c := s.CastleAt(p.Position())
		lord := petMountType(npcID) == serverpackets.MountStrider && c != nil && p.IsClanLeader() && p.ClanID() == c.OwnerID()
		if !lord {
			p.SendMessage("You cannot mount during a siege.")
			return false
		}
	}
	if !s.mountDisarm(p) {
		return false
	}
	p.skill.el.StopAllToggles(true)
	s.setMount(p, npcID, sm.level())
	p.mount.control = sm.pet.control
	s.mountStartFeed(p, npcID, sm)
	s.world.Broadcast(p, ridePacket(p), true)
	p.BroadcastUserInfo()
	s.unsummonNow(sm)
	return true
}
