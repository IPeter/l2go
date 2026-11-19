package org.l2jmobius.gameserver.mechanics.stats;

/**
 * Trait type (cut down for the datagen tests).
 */
public enum TraitType
{
	NONE(0),
	SWORD(1),
	BLUNT(1),
	POISON(3),
	HOLD(3),
	SLEEP(3),
	BUG_WEAKNESS(2);

	private final int _type; // 1 = weapon, 2 = weakness, 3 = resistance

	TraitType(int type)
	{
		_type = type;
	}

	public int getType()
	{
		return _type;
	}
}
