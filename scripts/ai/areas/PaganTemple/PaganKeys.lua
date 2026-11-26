-- The Pagan Temple keys, ported from Mobius ai/areas/PaganTemple/PaganKeys.
--
-- Three temple monsters drop a key in one kill of ten: into the killer's
-- inventory with auto loot, else on the ground for the killer.

local s = script.new("PaganKeys")

-- The key each monster drops, one kill in ten.
local KEYS = {
  [22140] = 8273, -- Zombie Worker: Anteroom Key
  [22142] = 8274, -- Triol's Layperson: Chapel Key
  [22168] = 8275, -- Triol's Priest: Key of Darkness
}
local KEY_CHANCE = 10

s:on_kill({ 22140, 22142, 22168 }, function(npc, killer, is_summon)
  local key = KEYS[npc:id()]
  if key == nil or rnd(100) >= KEY_CHANCE then
    return
  end
  if Config.AUTO_LOOT then
    s:give_items(killer, key, 1)
  else
    npc:drop_item(killer, key, 1)
  end
end)
