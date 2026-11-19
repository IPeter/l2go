// Small sample of the L2J Mobius Stat enum, for the datagen golden tests.
package org.l2jmobius.gameserver.mechanics.stats;

import java.util.NoSuchElementException;

/**
 * Enum of basic stats.
 */
public enum Stat
{
	// HP, MP & CP
	MAX_HP("maxHp"),
	MAX_MP("maxMp"),

	/* ATTACK & DEFENCE */
	POWER_DEFENCE("pDef"),
	MAGIC_DEFENCE("mDef"),
	POWER_ATTACK("pAtk"),
	MAGIC_ATTACK("mAtk"),
	POWER_ATTACK_SPEED("pAtkSpd"),
	SHIELD_DEFENCE("sDef"),
	EVASION_RATE("rEvas"),
	SHIELD_RATE("rShld"),
	CRITICAL_RATE("critRate"),
	ACCURACY_COMBAT("accCombat"),
	POWER_ATTACK_RANGE("pAtkRange"),
	STAT_STR("STR"),

	// Affects the random weapon damage.
	RANDOM_DAMAGE("randomDamage");

	public static final int NUM_STATS = values().length;

	private final String _value;

	private Stat(String s)
	{
		_value = s;
	}
}
