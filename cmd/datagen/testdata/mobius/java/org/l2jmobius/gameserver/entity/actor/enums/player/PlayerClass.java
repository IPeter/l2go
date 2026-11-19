// Small sample of the L2J Mobius PlayerClass enum, for the datagen golden
// test. Only the constants matter to datagen.
package org.l2jmobius.gameserver.entity.actor.enums.player;

import org.l2jmobius.gameserver.entity.actor.enums.creature.Race;

public enum PlayerClass
{
	FIGHTER(0, false, Race.HUMAN, null),

	WARRIOR(1, false, Race.HUMAN, FIGHTER),
	WARLOCK(14, true, true, Race.HUMAN, WARRIOR),

	ORC_MAGE(49, true, Race.ORC, null),
	ORC_SHAMAN(50, true, Race.ORC, ORC_MAGE);

	private PlayerClass(int id, boolean isMage, Race race, PlayerClass parent)
	{
	}

	private PlayerClass(int id, boolean isMage, boolean isSummoner, Race race, PlayerClass parent)
	{
	}
}
