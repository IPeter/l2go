package org.l2jmobius.gameserver.mechanics.skill;

/**
 * This enum class holds the skill operative types.
 */
public enum SkillOperateType
{
	/**
	 * Active Skill with "Instant Effect".
	 */
	A1,
	A2,
	A3,
	A4,
	CA1,
	CA5,
	DA1,
	DA2,

	/**
	 * Passive Skill.
	 */
	P,

	/**
	 * Toggle Skill.
	 */
	T;

	public boolean isPassive()
	{
		return this == P;
	}
}
