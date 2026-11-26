-- Corpses of the dead, ported from Mobius ai/others/WastedCorpse (Altur).
--
-- The retail wasted_corpse class kills the NPC as it is made: the bodies
-- are scenery, not living monsters.

local s = script.new("WastedCorpse")

local CORPSE_OF_DEADMAN = 18119

s:add_spawn_id(CORPSE_OF_DEADMAN)

s:on_spawn(function(npc)
  npc:die() -- Mobius doDie(null): no killer
end)
