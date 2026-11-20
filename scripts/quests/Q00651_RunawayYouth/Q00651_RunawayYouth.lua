-- Translated by questport from quests/Q00651_RunawayYouth (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(651, "Q00651_RunawayYouth", "Runaway Youth")

-- NPCs
local IVAN = 32014
local BATIDAE = 31989
-- Item
local SCROLL_OF_ESCAPE = 736
-- Table of possible spawns
local SPAWNS = {
  {x = 118600, y = -161235, z = -1119, heading = 0},
  {x = 108380, y = -150268, z = -2376, heading = 0},
  {x = 123254, y = -148126, z = -3425, heading = 0},
}
-- Current position
local _currentPosition = 0

q:add_start_npc(IVAN)
q:add_talk_id(IVAN, BATIDAE)

q:on_start(function()
  -- L2Go: on_start also runs after a reload, when Ivan is already out.
  if not world.npc_exists(IVAN) then
    q:add_spawn(IVAN, {x = 118600, y = -161235, z = -1119, heading = 0}, false, 0)
  end
end)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32014-04.htm" then
    if q:has_quest_items(player, SCROLL_OF_ESCAPE) then
      st:start_quest()
      q:take_items(player, SCROLL_OF_ESCAPE, 1)
      npc:magic_skill_use(npc, 2013, 1, 3500, 0)
      q:start_quest_timer("apparition_npc", 4000, npc, player, false)
      htmltext = "32014-03.htm"
    else
      st:exit_quest(true)
    end
  elseif event == "apparition_npc" then
    local chance = rnd(3)
    -- Loop to avoid to spawn to the same place.
    while chance == _currentPosition do
      chance = rnd(3)
    end
    -- Register new position.
    _currentPosition = chance
    npc:delete()
    q:add_spawn(IVAN, SPAWNS[chance + 1], false, 0)
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 26 and "32014-01.htm" or "32014-02.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == BATIDAE then
      htmltext = "31989-01.htm"
      q:give_adena(player, 2883, true)
      st:exit_quest(true, true)
    elseif sw2 == IVAN then
      htmltext = "32014-04a.htm"
    end
  end
  return htmltext
end)
