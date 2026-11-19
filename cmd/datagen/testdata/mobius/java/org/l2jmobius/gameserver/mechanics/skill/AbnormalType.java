package org.l2jmobius.gameserver.mechanics.skill;

/**
 * Abnormal type enumerate (cut down for the datagen tests).
 */
public enum AbnormalType
{
	ABNORMAL_INVINCIBILITY,
	INVINCIBILITY,
	PA_UP,
	SPEED_UP_SPECIAL,
	NONE;

	public static AbnormalType getAbnormalType(String type)
	{
		try
		{
			return Enum.valueOf(AbnormalType.class, type);
		}
		catch (Exception e)
		{
			return NONE;
		}
	}
}
