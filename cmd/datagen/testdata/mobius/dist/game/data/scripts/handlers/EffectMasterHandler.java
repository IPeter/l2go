package handlers;

import java.util.logging.Logger;

import handlers.skill.effects.*;

/**
 * Effect Master handler (cut down for the datagen tests).
 */
public class EffectMasterHandler
{
	private static final Logger LOGGER = Logger.getLogger(EffectMasterHandler.class.getName());

	private static final Class<?>[] EFFECTS =
	{
		Buff.class,
		FocusEnergy.class,
		// A disabled handler:
		null,
		MagicalDamage.class,
		PhysicalDamage.class,
		RestorationRandom.class,
	};

	public static void main(String[] args)
	{
		for (Class<?> c : EFFECTS)
		{
			if (c == null)
			{
				continue; // Disabled handler.
			}
		}
	}
}
