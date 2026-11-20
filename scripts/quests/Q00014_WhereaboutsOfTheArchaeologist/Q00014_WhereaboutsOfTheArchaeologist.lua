-- Translated by questport from quests/Q00014_WhereaboutsOfTheArchaeologist (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(14, "Q00014_WhereaboutsOfTheArchaeologist", "Whereabouts of the Archaeologist")

-- NPCs
local LIESEL = 31263
local GHOST_OF_ADVENTURER = 31538
-- Items
local LETTER = 7253

q:register_quest_items(LETTER)
q:add_start_npc(LIESEL)
q:add_talk_id(LIESEL, GHOST_OF_ADVENTURER)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31263-2.htm" then
    st:start_quest()
    q:give_items(player, LETTER, 1)
  elseif event == "31538-1.htm" then
    q:take_items(player, LETTER, 1)
    q:give_adena(player, 113228, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 74 and "31263-1.htm" or "31263-0.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == LIESEL then
      htmltext = "31263-2.htm"
    elseif sw2 == GHOST_OF_ADVENTURER then
      htmltext = "31538-0.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
