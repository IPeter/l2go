package skills

// Fisher is a World that can fish (Plan 14, M11g): the Fishing, Pumping
// and Reeling effects use it when the World implements it, and do nothing
// otherwise. The world does Mobius's checks and sends the messages.
type Fisher interface {
	// StartFishing is Mobius Fishing.onStart for a player effector: a cast
	// while fishing cancels it; otherwise the rod and lure checks, the bait
	// placement, one lure used and the fishing started.
	StartFishing(effector Creature)
	// Pump and Reel are Mobius Pumping.onStart and Reeling.onStart for a
	// player effector: without a fish to fight a message and ActionFailed;
	// otherwise the damage of the skill (with power, its level and the
	// fishing shot) goes to the fight.
	Pump(effector Creature, sk *Skill, power float64)
	Reel(effector Creature, sk *Skill, power float64)
}
