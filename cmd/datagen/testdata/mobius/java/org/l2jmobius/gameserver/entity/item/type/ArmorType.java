// Small sample of the L2J Mobius ArmorType enum, for the datagen golden tests.
package org.l2jmobius.gameserver.entity.item.type;

public enum ArmorType implements ItemType
{
	NONE,
	LIGHT,
	HEAVY,
	MAGIC,
	SIGIL,

	// L2J CUSTOM
	SHIELD;

	final int _mask;

	ArmorType()
	{
		_mask = 1 << (ordinal() + WeaponType.values().length);
	}
}
