// Small sample of the L2J Mobius CrystalType enum, for the datagen golden tests.
package org.l2jmobius.gameserver.entity.item.type;

public enum CrystalType
{
	NONE(0, 0, 0, 0),
	D(1, 1458, 11, 90),
	C(2, 1459, 6, 45),
	S(5, 1462, 25, 250);

	CrystalType(int level, int crystalId, int crystalEnchantBonusArmor, int crystalEnchantBonusWeapon)
	{
	}
}
