-- Translated by questport from quests/Q00654_JourneyToASettlement (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author Skache
local J = require("lib.java")

local q = quest.new(654, "Q00654_JourneyToASettlement", "Journey to a Settlement")

-- Items
local ANTELOPE_SKIN = 8072
local FORCE_FIELD_REMOVAL_SCROLL = 8073
-- NPCs
local NAMELESS_SPIRIT = 31453
local CANYON_ANTELOPE = 21294
local CANYON_ANTELOPE_SLAVE = 21295
-- Drop
local DROP_CHANCE = 30 -- Retail 30%
local DROP_AMOUNT = 1 -- Retail 1

q:register_quest_items(ANTELOPE_SKIN)
q:add_start_npc(NAMELESS_SPIRIT)
q:add_talk_id(NAMELESS_SPIRIT)
q:add_kill_id(CANYON_ANTELOPE, CANYON_ANTELOPE_SLAVE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31453-02.htm" then
    st:start_quest()
  elseif event == "31453-03.htm" then
    st:set_cond(2, true)
  elseif event == "31453-06.htm" then
    q:take_items(player, ANTELOPE_SKIN, -1)
    q:give_items(player, FORCE_FIELD_REMOVAL_SCROLL, 1)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    local prevSt = player:get_quest_state("Q00119_LastImperialPrince")
    htmltext = (prevSt == nil or not prevSt:is_completed() or player:level() < 74) and "31453-00.htm" or "31453-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "31453-02.htm"
    elseif cond == 2 then
      htmltext = "31453-04.htm"
    elseif cond == 3 then
      htmltext = "31453-05.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local qs = q:get_quest_state(player, false)
  if qs == nil or not qs:is_cond(2) then
    return
  end
  if rnd(100) < DROP_CHANCE then
    q:give_items(player, ANTELOPE_SKIN, DROP_AMOUNT)
    qs:set_cond(3, true)
  end
end)
