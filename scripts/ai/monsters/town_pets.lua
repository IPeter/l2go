-- Town pets, ported from Mobius ai/others/TownPets (Mobius).
--
-- The pets of the towns run about their spawn point: every 5 s, while
-- players are near, to a point up to 50 away (as far as geodata lets
-- them), unless that is within 20 of where they stand; every 15 s they
-- look again while nobody is near.

local s = script.new("TownPets")

local PETS = {
  31202, -- Maximus
  31203, -- Moon Dancer
  31204, -- Georgio
  31205, -- Katz
  31206, -- Ten Ten
  31207, -- Sardinia
  31208, -- La Grange
  31209, -- Misty Rain
  31266, -- Kaiser
  31593, -- Dorothy
  31606, -- Alice de Catrina
  31609, -- Aurora the Unicorn
  31625, -- Yeti
  31630, -- Kaleidos the Unicorn
  31703, -- Saint's Follower_a
  31704, -- Saint's Follower_b
  31758, -- Rafi
  31955, -- Ruby
}

local MOVE_INTERVAL = 5000
local MOVE_RANGE = 50
local MIN_MOVE = 20

s:add_spawn_id(PETS)

s:on_event(function(event, npc, player)
  if event ~= "move" or npc == nil or not npc:valid() then
    return nil
  end
  if not npc:is_in_active_region() then
    s:start_quest_timer("move", MOVE_INTERVAL * 3, npc, nil)
    return nil
  end
  local spawn = npc:spawn_location()
  if spawn ~= nil then
    local x = spawn.x + rnd(-MOVE_RANGE, MOVE_RANGE)
    local y = spawn.y + rnd(-MOVE_RANGE, MOVE_RANGE)
    local destination = world.valid_location(npc, { x = x, y = y, z = npc:z() })
    if npc:calculate_distance_3d(destination) > MIN_MOVE then
      npc:move_to(destination)
    end
  end
  s:start_quest_timer("move", MOVE_INTERVAL, npc, nil)
  return nil
end)

s:on_spawn(function(npc)
  npc:set_running()
  s:start_quest_timer("move", MOVE_INTERVAL, npc, nil)
end)

-- Timers do not survive a reload: start them again for the pets in the
-- world (a pet with a running timer keeps it).
s:on_start(function(ev)
  for _, id in ipairs(PETS) do
    for _, npc in ipairs(world.npcs_by_id(id)) do
      s:start_quest_timer("move", MOVE_INTERVAL, npc, nil)
    end
  end
end)
