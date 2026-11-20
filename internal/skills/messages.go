package skills

import (
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// The system messages of skills and effects (Mobius SystemMessageId). The
// comment is the client text.
const (
	MsgNotEnoughHP                 sp.SystemMessageID = 23   // Not enough HP.
	MsgNotEnoughMP                 sp.SystemMessageID = 24   // Not enough MP.
	MsgRejuvenatingHP              sp.SystemMessageID = 25   // Rejuvenating HP.
	MsgUseS1                       sp.SystemMessageID = 46   // You use $s1.
	MsgCannotUseOnYourself         sp.SystemMessageID = 51   // You cannot use this on yourself.
	MsgS1HasWornOff                sp.SystemMessageID = 92   // $s1 has worn off.
	MsgEffectsOfS1FlowThroughYou   sp.SystemMessageID = 110  // The effects of $s1 flow through you.
	MsgS1CannotBeUsedUnsuitable    sp.SystemMessageID = 113  // $s1 cannot be used due to unsuitable terms.
	MsgS1HasResistedYourS2         sp.SystemMessageID = 139  // $s1 has resisted your $s2.
	MsgIncorrectTarget             sp.SystemMessageID = 144  // That is the incorrect target.
	MsgDrainHalfSuccessful         sp.SystemMessageID = 156  // Drain was only 50 percent successful.
	MsgYouResistedS1sDrain         sp.SystemMessageID = 157  // You resisted $s1's drain.
	MsgYourAttackHasFailed         sp.SystemMessageID = 158  // Your attack has failed.
	MsgYouResistedS1sMagic         sp.SystemMessageID = 159  // You have resisted $s1's magic.
	MsgCannotSeeTarget             sp.SystemMessageID = 181  // Cannot see target.
	MsgS1HasBeenAborted            sp.SystemMessageID = 335  // $s1 has been aborted.
	MsgDistanceTooFarCastStopped   sp.SystemMessageID = 748  // The distance is too far and so the casting has been stopped.
	MsgEffectOfS1Removed           sp.SystemMessageID = 749  // The effect of $s1 has been removed.
	MsgCannotResurrectInSiege      sp.SystemMessageID = 1053 // It is not possible to resurrect in battlefields where a siege war is taking place.
	MsgS1HPRestored                sp.SystemMessageID = 1066 // $s1 HP has been restored.
	MsgS2HPRestoredByS1            sp.SystemMessageID = 1067 // $s2 HP has been restored by $s1.
	MsgS1MPRestored                sp.SystemMessageID = 1068 // $s1 MP has been restored.
	MsgS2MPRestoredByS1            sp.SystemMessageID = 1069 // $s2 MP has been restored by $s1.
	MsgMagicCriticalHit            sp.SystemMessageID = 1280 // Magic Critical Hit!
	MsgS1CPRestored                sp.SystemMessageID = 1405 // $s1 CPs have been restored.
	MsgS1RestoresS2CP              sp.SystemMessageID = 1406 // $s1 restores $s2 CP.
	MsgS1IsPerformingCounterAttack sp.SystemMessageID = 1997 // $s1 is performing a counter-attack.
	MsgYouCounterAttackS1sAttack   sp.SystemMessageID = 1998 // You counter-attack $s1's attack.
	MsgS1DodgesTheAttack           sp.SystemMessageID = 1999 // $s1 dodges the attack.
	MsgYouDodgeS1sAttack           sp.SystemMessageID = 2000 // You dodge $s1's attack.
	MsgSkillReadyToBeUsedAgain     sp.SystemMessageID = 2015 // A skill is ready to be used again.
)

// The texts Mobius sends with sendMessage.
const (
	TextTargetTypeNotHandled = "Target type of skill is not currently handled."
	TextNoFestivalResurrect  = "You may not resurrect participants in a festival."
)

// message builds a system message.
func message(id sp.SystemMessageID, params ...sp.SystemMessageParam) sp.SystemMessage {
	return sp.SystemMessage{ID: id, Params: params}
}

// SkillName is the skill name parameter of a system message, as Mobius
// SystemMessage.addSkillName(Skill) writes it: a text with the name when
// the skill shows another skill's ID, else the skill ID and level.
func SkillName(sk *Skill) sp.SystemMessageParam {
	if sk.ID != sk.DisplayID {
		return sp.TextParam(sk.Name)
	}
	return sp.SkillNameParam(int32(sk.ID), int32(sk.Level)) //nolint:gosec // skill IDs and levels are small
}

// skillNameLevel1 is addSkillName(int id): the skill at level 1.
func skillNameLevel1(id int) sp.SystemMessageParam {
	return sp.SkillNameParam(int32(id), 1) //nolint:gosec // skill IDs are small
}

// send sends m to c if c is not nil.
func send(c Creature, id sp.SystemMessageID, params ...sp.SystemMessageParam) {
	if c != nil {
		c.SendMessage(message(id, params...))
	}
}

// Send sends the system message id with params to c; nothing for a nil
// creature. NPCs ignore messages.
func Send(c Creature, id sp.SystemMessageID, params ...sp.SystemMessageParam) {
	send(c, id, params...)
}
