package org.l2jmobius.gameserver.mechanics.skill;

/**
 * Abnormal Visual Effect enumerated (cut down for the datagen tests).
 */
public enum AbnormalVisualEffect
{
	NONE(0x0000000, 0),
	DOT_BLEEDING(0x00000001, 0),
	TURN_FLEE(0x80000000, 0),
	INVINCIBILITY(0x000008, 1),
	BR_SOUL_AVATAR(0x000100, 2); // High Five

	private final int _mask;
	private final int _type;

	private AbnormalVisualEffect(int mask, int type)
	{
		_mask = mask;
		_type = type;
	}
}
