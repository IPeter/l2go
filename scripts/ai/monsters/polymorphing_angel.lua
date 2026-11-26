-- Angels that come back in another shape, ported from Mobius
-- ai/others/PolymorphingAngel.
--
-- When one of these angels dies, its other form appears where it died and
-- runs.

local s = script.new("PolymorphingAngel")

local ANGEL_SPAWNS = {
  [20830] = 20859,
  [21067] = 21068,
  [21062] = 21063,
  [20831] = 20860,
  [21070] = 21071,
}

for id in pairs(ANGEL_SPAWNS) do
  s:add_kill_id(id)
end

s:on_kill(function(npc, killer, is_summon)
  local angel = s:add_spawn(ANGEL_SPAWNS[npc:id()], npc)
  if angel ~= nil then
    angel:set_running()
  end
end)
