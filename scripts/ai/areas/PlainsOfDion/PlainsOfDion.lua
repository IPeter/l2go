-- The Delu lizardmen of the Plains of Dion (near Floran), ported from
-- Mobius ai/areas/PlainsOfDion (Gladicek).
--
-- The first hit on a lizardman interrupts its "duel": it shouts, and the
-- lizardmen within its clan help range that can see it, are not dead and
-- not swinging, attack the player and shout too.

local J = require("lib.java")

local s = script.new("PlainsOfDion")

local DELU_LIZARDMEN = {
  21104, -- Delu Lizardman Supplier
  21105, -- Delu Lizardman Special Agent
  21107, -- Delu Lizardman Commander
}

local MONSTERS_MSG = {
  "$s1! How dare you interrupt our fight! Hey guys, help!",
  "$s1! Hey! We're having a duel here!",
  "The duel is over! Attack!",
  "Foul! Kill the coward!",
  "How dare you interrupt a sacred duel! You must be taught a lesson!",
}

local MONSTERS_ASSIST_MSG = {
  "Die, you coward!",
  "Kill the coward!",
  "What are you looking at?",
}

s:on_attack(DELU_LIZARDMEN, function(npc, player, damage, is_summon)
  if not npc:is_script_value(0) then
    return
  end
  npc:broadcast_say(ChatType.NPC_GENERAL, J.replace(MONSTERS_MSG[rnd(5) + 1], "$s1", player:name()))

  for _, obj in ipairs(world.visible_objects(npc, "Monster", npc:clan_help_range())) do
    if J.contains(DELU_LIZARDMEN, obj:id()) and not obj:is_attacking_now() and not obj:is_dead() and world.can_see_target(npc, obj) then
      -- Mobius addAttackDesire.
      obj:add_damage_hate(player, 0, 999)
      obj:set_running()
      obj:attack(player)
      obj:broadcast_say(ChatType.NPC_GENERAL, MONSTERS_ASSIST_MSG[rnd(3) + 1])
    end
  end
  npc:set_script_value(1)
end)
