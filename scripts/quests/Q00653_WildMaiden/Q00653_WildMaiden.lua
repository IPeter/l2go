-- Translated by questport from quests/Q00653_WildMaiden (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(653, "Q00653_WildMaiden", "Wild Maiden")

-- NPCs
local SUKI = 32013
local GALIBREDO = 30181
-- Item
local SCROLL_OF_ESCAPE = 736
-- Table of possible spawns
local SPAWNS = {
  {x = 66578, y = 72351, z = -3731, heading = 0},
  {x = 77189, y = 73610, z = -3708, heading = 2555},
  {x = 71809, y = 67377, z = -3675, heading = 29130},
  {x = 69166, y = 88825, z = -3447, heading = 43886},
}
-- Current position
local _currentPosition = 0

q:add_start_npc(SUKI)
q:add_talk_id(SUKI, GALIBREDO)

q:on_start(function()
  -- L2Go: on_start also runs after a reload, when Suki is already out.
  if not world.npc_exists(SUKI) then
    q:add_spawn(SUKI, {x = 66578, y = 72351, z = -3731, heading = 0}, false, 0)
  end
end)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32013-03.htm" then
    if q:has_quest_items(player, SCROLL_OF_ESCAPE) then
      st:start_quest()
      q:take_items(player, SCROLL_OF_ESCAPE, 1)
      npc:magic_skill_use(npc, 2013, 1, 3500, 0)
      q:start_quest_timer("apparition_npc", 4000, npc, player, false)
    else
      htmltext = "32013-03a.htm"
      st:exit_quest(true)
    end
  elseif event == "apparition_npc" then
    local chance = rnd(4)
    -- Loop to avoid to spawn to the same place.
    while chance == _currentPosition do
      chance = rnd(4)
    end
    -- Register new position.
    _currentPosition = chance
    npc:delete()
    q:add_spawn(SUKI, SPAWNS[chance + 1], false, 0)
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 36 and "32013-01.htm" or "32013-02.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == GALIBREDO then
      htmltext = "30181-01.htm"
      q:give_adena(player, 2883, true)
      st:exit_quest(true, true)
    elseif sw2 == SUKI then
      htmltext = "32013-04a.htm"
    end
  end
  return htmltext
end)
