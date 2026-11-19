package org.l2jmobius.gameserver.data.enums;

public enum CategoryType
{
	FIGHTER_GROUP,
	MAGE_GROUP,
	WIZARD_GROUP;

	public static CategoryType findByName(String categoryName)
	{
		for (CategoryType type : values())
		{
			if (type.name().equalsIgnoreCase(categoryName))
			{
				return type;
			}
		}

		return null;
	}
}
