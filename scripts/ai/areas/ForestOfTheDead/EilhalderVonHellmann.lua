-- Eilhalder von Hellmann walks the Forest of the Dead at night, ported
-- from Mobius ai/areas/ForestOfTheDead/EilhalderVonHellmann.
--
-- At game midnight he appears; at 06:00 he leaves, or, while he fights,
-- tries again every 30 s.
--
-- Restart-safe: the boss is found in the world (Mobius keeps him in a
-- static field), and a reload during the day starts the leave timer again
-- for a boss still there.

local s = script.new("EilhalderVonHellmann")

local EILHALDER_VON_HELLMANN = 25328
local SPAWN_LOCATION = { x = 59090, y = -42188, z = -3003 }

-- The boss in the world, or nil.
local function instance()
  return world.npcs_by_id(EILHALDER_VON_HELLMANN)[1]
end

local function leave(npc)
  if npc:is_in_combat() then
    s:start_quest_timer("despawn", 30000, npc, nil)
  else
    npc:delete()
  end
end

s:on_start(function(ev)
  local boss = instance()
  if ev.reload and boss ~= nil and not time.is_night() then
    s:start_quest_timer("despawn", 30000, boss, nil)
  end
end)

s:on_event(function(event, npc, player)
  if npc ~= nil then
    leave(npc)
  end
  return nil
end)

s:on_day_night_change(function(is_night)
  if not is_night then
    local boss = instance()
    if boss ~= nil then
      leave(boss)
    end
  elseif instance() == nil then
    s:add_spawn(EILHALDER_VON_HELLMANN, SPAWN_LOCATION)
  end
end)
