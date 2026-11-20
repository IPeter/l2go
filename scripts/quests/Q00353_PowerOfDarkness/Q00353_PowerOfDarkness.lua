-- Translated by questport from quests/Q00353_PowerOfDarkness (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(353, "Q00353_PowerOfDarkness", "Power of Darkness")

-- Item
local STONE = 5862

q:register_quest_items(STONE)
q:add_start_npc(31044) -- Galman
q:add_talk_id(31044)
q:add_kill_id(20244, 20245, 20283, 20284)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31044-04.htm" then
    st:start_quest()
  elseif event == "31044-08.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 55 and "31044-01.htm" or "31044-02.htm"
  elseif sw == State.STARTED then
    local stones = q:get_quest_items_count(player, STONE)
    if stones == 0 then
      htmltext = "31044-05.htm"
    else
      htmltext = "31044-06.htm"
      q:take_items(player, STONE, -1)
      q:give_adena(player, 2500 + 230 * stones, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isPet)
  local st = q:get_quest_state(killer, false)
  if st == nil then
    return
  end
  if st:is_cond(1) and rnd(100) < ((npc:id() == 20244 or npc:id() == 20283) and 48 or 50) then
    q:give_items(killer, STONE, 1)
  end
end)
