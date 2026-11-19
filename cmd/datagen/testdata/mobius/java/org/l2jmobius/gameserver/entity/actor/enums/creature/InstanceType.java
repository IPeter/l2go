package org.l2jmobius.gameserver.entity.actor.enums.creature;

public enum InstanceType
{
	WorldObject(null),
	Creature(WorldObject),
	Npc(Creature),
	Playable(Creature),
	Summon(Playable),

	// Summons, Pets, Decoys and Traps
	Pet(Summon),

	// Attackable
	Attackable(Npc),
	Monster(Attackable),
	Chest(Monster);

	private final InstanceType _parent;

	private InstanceType(InstanceType parent)
	{
		_parent = parent;
	}
}
