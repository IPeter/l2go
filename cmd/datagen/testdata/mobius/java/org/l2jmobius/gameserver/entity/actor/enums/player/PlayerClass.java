// Small sample of the L2J Mobius PlayerClass enum, for the datagen golden
// tests. Only the constants matter to datagen; they must match the real
// enum, which internal/data was generated from.
package org.l2jmobius.gameserver.entity.actor.enums.player;

import org.l2jmobius.gameserver.entity.actor.enums.creature.Race;

public enum PlayerClass
{
	FIGHTER(0, false, Race.HUMAN, null),

	WARRIOR(1, false, Race.HUMAN, FIGHTER), // first class transfer

	ORC_MAGE(49, true, Race.ORC, null);

	private PlayerClass(int id, boolean isMage, Race race, PlayerClass parent)
	{
	}

	private PlayerClass(int id, boolean isMage, boolean isSummoner, Race race, PlayerClass parent)
	{
	}
}
