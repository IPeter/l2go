package org.l2jmobius.gameserver.util;

/**
 * Cut down for the datagen tests: the condition parsing of DocumentBase.
 */
public abstract class DocumentBase
{
	protected Condition parseCondition(Node node, Object template)
	{
		Node n = node;
		while (n != null)
		{
			if ((n.getNodeType() == Node.ELEMENT_NODE) && !"value".equalsIgnoreCase(n.getNodeName()))
			{
				break;
			}
			n = n.getNextSibling();
		}

		switch (n.getNodeName().toLowerCase())
		{
			case "and":
			{
				return parseLogicAnd(n, template);
			}
			case "or":
			{
				return parseLogicOr(n, template);
			}
			case "not":
			{
				return parseLogicNot(n, template);
			}
			case "player":
			{
				return parsePlayerCondition(n, template);
			}
			case "target":
			{
				return parseTargetCondition(n, template);
			}
			case "using":
			{
				return parseUsingCondition(n);
			}
			case "game":
			{
				return parseGameCondition(n);
			}
		}

		return null;
	}

	protected Condition parsePlayerCondition(Node n, Object template)
	{
		Condition cond = null;
		final NamedNodeMap attrs = n.getAttributes();
		for (int i = 0; i < attrs.getLength(); i++)
		{
			final Node a = attrs.item(i);
			switch (a.getNodeName().toLowerCase())
			{
				case "level":
				{
					final int lvl = Integer.decode(getValue(a.getNodeValue(), template));
					cond = joinAnd(cond, new ConditionPlayerLevel(lvl));
					break;
				}
				case "charges":
				{
					final int value = Integer.decode(getValue(a.getNodeValue(), template));
					cond = joinAnd(cond, new ConditionPlayerCharges(value));
					break;
				}
				case "active_skill_id_lvl":
				{
					final String val = getValue(a.getNodeValue(), template);
					cond = joinAnd(cond, new ConditionPlayerActiveSkillId(0, 0));
					break;
				}
			}
		}

		return cond;
	}

	protected Condition parseTargetCondition(Node n, Object template)
	{
		Condition cond = null;
		final NamedNodeMap attrs = n.getAttributes();
		for (int i = 0; i < attrs.getLength(); i++)
		{
			final Node a = attrs.item(i);
			switch (a.getNodeName().toLowerCase())
			{
				case "race":
				{
					cond = joinAnd(cond, new ConditionTargetRace(Race.valueOf(a.getNodeValue())));
					break;
				}
			}
		}

		return cond;
	}

	protected Condition parseUsingCondition(Node n)
	{
		Condition cond = null;
		final NamedNodeMap attrs = n.getAttributes();
		for (int i = 0; i < attrs.getLength(); i++)
		{
			final Node a = attrs.item(i);
			switch (a.getNodeName().toLowerCase())
			{
				case "kind":
				{
					cond = joinAnd(cond, new ConditionUsingItemType(0));
					break;
				}
				case "slot":
				{
					cond = joinAnd(cond, new ConditionUsingSlotType(0));
					break;
				}
			}
		}

		return cond;
	}

	protected Condition parseGameCondition(Node n)
	{
		Condition cond = null;
		final NamedNodeMap attrs = n.getAttributes();
		for (int i = 0; i < attrs.getLength(); i++)
		{
			final Node a = attrs.item(i);
			if ("night".equalsIgnoreCase(a.getNodeName()))
			{
				final boolean val = Boolean.parseBoolean(a.getNodeValue());
				cond = joinAnd(cond, new ConditionGameTime(val));
			}

			if ("chance".equalsIgnoreCase(a.getNodeName()))
			{
				final int val = Integer.decode(getValue(a.getNodeValue(), null));
				cond = joinAnd(cond, new ConditionGameChance(val));
			}
		}

		return cond;
	}
}
