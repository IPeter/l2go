-- Translated by questport from quests/Q00110_ToThePrimevalIsle (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(110, "Q00110_ToThePrimevalIsle", "To the Primeval Isle")

-- NPCs
local ANTON = 31338
local MARQUEZ = 32113
-- Item
local ANCIENT_BOOK = 8777

q:register_quest_items(ANCIENT_BOOK)
q:add_start_npc(ANTON)
q:add_talk_id(ANTON, MARQUEZ)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31338-02.htm" then
    st:start_quest()
    q:give_items(player, ANCIENT_BOOK, 1)
  elseif event == "32113-03.htm" and q:has_quest_items(player, ANCIENT_BOOK) then
    q:take_items(player, ANCIENT_BOOK, 1)
    q:give_adena(player, 169380, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 75 and "31338-00.htm" or "31338-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == ANTON then
      htmltext = "31338-01c.htm"
    elseif sw2 == MARQUEZ then
      htmltext = "32113-01.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
