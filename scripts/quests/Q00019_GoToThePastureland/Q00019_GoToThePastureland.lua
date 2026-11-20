-- Translated by questport from quests/Q00019_GoToThePastureland (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(19, "Q00019_GoToThePastureland", "Go to the Pastureland!")

-- NPCs
local VLADIMIR = 31302
local TUNATUN = 31537
-- Items
local YOUNG_WILD_BEAST_MEAT = 7547

q:register_quest_items(YOUNG_WILD_BEAST_MEAT)
q:add_start_npc(VLADIMIR)
q:add_talk_id(VLADIMIR, TUNATUN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31302-01.htm" then
    st:start_quest()
    q:give_items(player, YOUNG_WILD_BEAST_MEAT, 1)
  elseif event == "019_finish" then
    if q:has_quest_items(player, YOUNG_WILD_BEAST_MEAT) then
      htmltext = "31537-01.htm"
      q:take_items(player, YOUNG_WILD_BEAST_MEAT, 1)
      q:give_adena(player, 30000, true)
      st:exit_quest(false, true)
    else
      htmltext = "31537-02.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 63 and "31302-03.htm" or "31302-00.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == VLADIMIR then
      htmltext = "31302-02.htm"
    elseif sw2 == TUNATUN then
      htmltext = "31537-00.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
