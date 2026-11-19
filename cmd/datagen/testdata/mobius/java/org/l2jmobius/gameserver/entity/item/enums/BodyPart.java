// Small sample of the L2J Mobius BodyPart enum, for the datagen golden tests.
package org.l2jmobius.gameserver.entity.item.enums;

import org.l2jmobius.gameserver.entity.itemcontainer.Inventory;

public enum BodyPart
{
	// Items with no body part.
	NONE(0x0000, Inventory.PAPERDOLL_TOTALSLOTS, "none"),

	UNDERWEAR(0x0001, Inventory.PAPERDOLL_UNDER, "underwear", "shirt"),
	R_EAR(0x0002, Inventory.PAPERDOLL_REAR, "rear", "rbracelet"),
	L_EAR(0x0004, Inventory.PAPERDOLL_LEAR, "lear"),
	LR_EAR(0x0006, -1, "rear;lear"), // Special case, no direct paperdoll slot.
	R_HAND(0x0080, Inventory.PAPERDOLL_RHAND, "rhand"),
	L_HAND(0x0100, Inventory.PAPERDOLL_LHAND, "lhand"),
	CHEST(0x0400, Inventory.PAPERDOLL_CHEST, "chest"),
	LR_HAND(0x4000, -1, "lrhand"), // Maps to RHAND slot.
	FULL_ARMOR(0x8000, -1, "fullarmor", "onepiece"), // Maps to CHEST slot.

	// Special slots for pets.
	WOLF(-100, -1, "wolf");

	private final int _mask;

	private BodyPart(int mask, int paperdollSlot, String... names)
	{
		_mask = mask;
	}
}
