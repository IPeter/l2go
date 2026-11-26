-- Monsters Silent Move does not fool, ported from Mobius
-- ai/others/SeeThroughSilentMove (Gigiikun).

local s = script.new("SeeThroughSilentMove")

local MONSTERS = {
  18001, 18002, 22199, 22215, 22216, 22217,
  29009,
  29010, 29011, 29012, 29013,
}

s:add_spawn_id(MONSTERS)

s:on_spawn(function(npc)
  if npc:is_type("Attackable") then
    npc:set_see_through_silent_move(true)
  end
end)
