// Small sample of the L2J Mobius EtcItemType enum, for the datagen golden tests.
package org.l2jmobius.gameserver.entity.item.type;

public enum EtcItemType implements ItemType
{
	NONE,
	ARROW,
	POTION,
	SCROLL,

	// L2J CUSTOM, BACKWARD COMPATIBILITY
	SHOT;

	@Override
	public int mask()
	{
		return 0;
	}
}
