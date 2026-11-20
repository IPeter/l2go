-- Translated by questport from quests/Q00109_InSearchOfTheNest (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(109, "Q00109_InSearchOfTheNest", "In Search of the Nest")

-- NPCs
local PIERCE = 31553
local KAHMAN = 31554
local SCOUT_CORPSE = 32015
-- Items
local SCOUT_MEMO = 8083
local RECRUIT_BADGE = 7246
local SOLDIER_BADGE = 7247

q:register_quest_items(SCOUT_MEMO)
q:add_start_npc(PIERCE)
q:add_talk_id(PIERCE, SCOUT_CORPSE, KAHMAN)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31553-01.htm" then
    st:start_quest()
  elseif event == "32015-02.htm" then
    st:set_cond(2, true)
    q:give_items(player, SCOUT_MEMO, 1)
  elseif event == "31553-03.htm" then
    st:set_cond(3, true)
    q:take_items(player, SCOUT_MEMO, 1)
  elseif event == "31554-02.htm" then
    q:give_adena(player, 5168, true)
    st:exit_quest(false, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    -- Must worn one or other Golden Ram Badge in order to be accepted.
    if player:level() >= 66 and q:has_at_least_one_quest_item(player, RECRUIT_BADGE, SOLDIER_BADGE) then
      htmltext = "31553-00.htm"
    else
      htmltext = "31553-00a.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == PIERCE then
      if cond == 1 then
        htmltext = "31553-01a.htm"
      elseif cond == 2 then
        htmltext = "31553-02.htm"
      elseif cond == 3 then
        htmltext = "31553-03.htm"
      end
    elseif sw2 == SCOUT_CORPSE then
      if cond == 1 then
        htmltext = "32015-01.htm"
      elseif cond == 2 then
        htmltext = "32015-02.htm"
      end
    elseif sw2 == KAHMAN then
      if cond == 3 then
        htmltext = "31554-01.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
