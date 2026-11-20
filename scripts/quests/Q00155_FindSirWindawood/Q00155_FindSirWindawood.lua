-- Translated by questport from quests/Q00155_FindSirWindawood (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(155, "Q00155_FindSirWindawood", "Find Sir Windawood")

-- NPCs
local ABELLOS = 30042
local WINDAWOOD = 30311
-- Items
local OFFICIAL_LETTER = 1019
local HASTE_POTION = 734

q:register_quest_items(OFFICIAL_LETTER)
q:add_start_npc(ABELLOS)
q:add_talk_id(WINDAWOOD, ABELLOS)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30042-02.htm" then
    st:start_quest()
    q:give_items(player, OFFICIAL_LETTER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 3 and "30042-01a.htm" or "30042-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == ABELLOS then
      htmltext = "30042-03.htm"
    elseif sw2 == WINDAWOOD then
      if q:has_quest_items(player, OFFICIAL_LETTER) then
        htmltext = "30311-01.htm"
        q:take_items(player, OFFICIAL_LETTER, 1)
        q:reward_items(player, HASTE_POTION, 1)
        st:exit_quest(false, true)
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
