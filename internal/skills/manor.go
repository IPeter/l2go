package skills

// ManorWorld is a World with the castle manor (Plan 14 M11k). The Sow and
// Harvesting effects use it when the World implements it, and do nothing
// otherwise.
type ManorWorld interface {
	// Sow is Sow.onStart: the player effector sows the seed it chose
	// (handlers/items/Seed) on the monster effected.
	Sow(effector, effected Creature)
	// Harvest is Harvesting.onStart: the player effector harvests the dead
	// monster effected.
	Harvest(effector, effected Creature)
}
