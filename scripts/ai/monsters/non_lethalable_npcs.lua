-- NPCs lethal strikes do not work on, ported from Mobius
-- ai/others/NonLethalableNpcs (UnAfraid).

local s = script.new("NonLethalableNpcs")

local NPCS = {
  35062, -- Headquarters
  -- Conquerable clan hall bosses.
  35410, -- Gustav
  35375, -- Nurka
  35629, -- Lidia
  35630, -- Alfred
  35631, -- Giselle
}

s:add_spawn_id(NPCS)

s:on_spawn(function(npc)
  npc:set_lethalable(false)
end)
