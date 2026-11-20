-- Translated by questport from quests/Q00652_AnAgedExAdventurer (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(652, "Q00652_AnAgedExAdventurer", "An Aged Ex-Adventurer")

-- NPCs
local TANTAN = 32012
local SARA = 30180
-- Item
local SOULSHOT_C = 1464
-- Reward
local ENCHANT_ARMOR_D = 956
-- Table of possible spawns
local SPAWNS = {
  {x = 78355, y = -1325, z = -3659, heading = 0},
  {x = 79890, y = -6132, z = -2922, heading = 0},
  {x = 90012, y = -7217, z = -3085, heading = 0},
  {x = 94500, y = -10129, z = -3290, heading = 0},
  {x = 96534, y = -1237, z = -3677, heading = 0},
}
-- Current position
local _currentPosition = 0

q:add_start_npc(TANTAN)
q:add_talk_id(TANTAN, SARA)

q:on_start(function()
  -- L2Go: on_start also runs after a reload, when Tantan is already out.
  if not world.npc_exists(TANTAN) then
    q:add_spawn(TANTAN, {x = 78355, y = -1325, z = -3659, heading = 0}, false, 0)
  end
end)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32012-02.htm" then
    if q:get_quest_items_count(player, SOULSHOT_C) >= 100 then
      st:start_quest()
      q:take_items(player, SOULSHOT_C, 100)
      npc:move_to({x = 85326, y = 7869, z = -3620})
      q:start_quest_timer("apparition_npc", 6000, npc, player, false)
    else
      htmltext = "32012-02a.htm"
      st:exit_quest(true)
    end
  elseif event == "apparition_npc" then
    local chance = rnd(5)
    -- Loop to avoid to spawn to the same place.
    while chance == _currentPosition do
      chance = rnd(5)
    end
    -- Register new position.
    _currentPosition = chance
    npc:delete()
    q:add_spawn(TANTAN, {x = SPAWNS[chance + 1].x, y = SPAWNS[chance + 1].y, z = SPAWNS[chance + 1].z, heading = SPAWNS[chance + 1].heading}, false, 0)
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 46 and "32012-00.htm" or "32012-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == SARA then
      if rnd(100) < 50 then
        htmltext = "30180-01.htm"
        q:give_adena(player, 5026, true)
        q:give_items(player, ENCHANT_ARMOR_D, 1)
      else
        htmltext = "30180-02.htm"
        q:give_adena(player, 10000, true)
      end
      st:exit_quest(true, true)
    elseif sw2 == TANTAN then
      htmltext = "32012-04a.htm"
    end
  end
  return htmltext
end)
