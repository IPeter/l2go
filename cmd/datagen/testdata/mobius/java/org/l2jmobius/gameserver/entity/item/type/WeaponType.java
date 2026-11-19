// Small sample of the L2J Mobius WeaponType enum, for the datagen golden tests.
package org.l2jmobius.gameserver.entity.item.type;

import org.l2jmobius.gameserver.mechanics.stats.TraitType;

public enum WeaponType implements ItemType
{
	SWORD(TraitType.SWORD),
	BLUNT(TraitType.BLUNT),
	DAGGER(TraitType.DAGGER),
	BOW(TraitType.BOW),
	NONE(TraitType.NONE),
	FIST(TraitType.FIST),
	DUALFIST(TraitType.DUALFIST);

	private final int _mask;

	WeaponType(TraitType traitType)
	{
		_mask = 1 << ordinal();
	}
}
