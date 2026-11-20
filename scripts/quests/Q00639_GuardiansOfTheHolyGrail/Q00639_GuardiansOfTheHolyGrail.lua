-- Translated by questport from quests/Q00639_GuardiansOfTheHolyGrail (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(639, "Q00639_GuardiansOfTheHolyGrail", "Guardians of the Holy Grail")

-- NPCs
local DOMINIC = 31350
local GREMORY = 32008
local HOLY_GRAIL = 32028
-- Items
local SCRIPTURE = 8069
local WATER_BOTTLE = 8070
local HOLY_WATER_BOTTLE = 8071
local CHANCES = {}

CHANCES[22122] = 760000
CHANCES[22123] = 750000
CHANCES[22124] = 590000
CHANCES[22125] = 580000
CHANCES[22126] = 590000
CHANCES[22127] = 580000
CHANCES[22128] = 170000
CHANCES[22129] = 590000
CHANCES[22130] = 850000
CHANCES[22131] = 920000
CHANCES[22132] = 580000
CHANCES[22133] = 930000
CHANCES[22134] = 230000
CHANCES[22135] = 580000

q:register_quest_items(SCRIPTURE, WATER_BOTTLE, HOLY_WATER_BOTTLE)
q:add_start_npc(DOMINIC)
q:add_talk_id(DOMINIC, GREMORY, HOLY_GRAIL)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  local htmltext = nil
  if qs == nil then
    return htmltext
  end
  if event == "31350-03.htm" or event == "31350-07.htm" or event == "32008-02.htm" or event == "32008-03.htm" or event == "32008-04.htm" then
    htmltext = event
  elseif event == "31350-04.htm" then
    qs:start_quest()
    htmltext = event
  elseif event == "31350-08.htm" then
    local count = q:get_quest_items_count(player, SCRIPTURE)
    q:take_items(player, SCRIPTURE, -1)
    q:give_adena(player, 1625 * count + (count >= 10 and 33940 or 0), true)
    htmltext = event
  elseif event == "31350-09.htm" then
    qs:exit_quest(true, true)
    htmltext = event
  elseif event == "32008-05.htm" then
    qs:set_cond(2, true)
    q:give_items(player, WATER_BOTTLE, 1)
    htmltext = event
  elseif event == "32008-09.htm" then
    qs:set_cond(4, true)
    q:take_items(player, HOLY_WATER_BOTTLE, 1)
    htmltext = event
  elseif event == "32008-12.htm" then
    if q:get_quest_items_count(player, SCRIPTURE) >= 4000 then
      q:take_items(player, SCRIPTURE, 4000)
      q:reward_items(player, 959, 1)
      htmltext = "32008-11.htm"
    else
      htmltext = event
    end
  elseif event == "32008-14.htm" then
    if q:get_quest_items_count(player, SCRIPTURE) >= 400 then
      q:take_items(player, SCRIPTURE, 400)
      q:reward_items(player, 960, 1)
      htmltext = "32008-13.htm"
    else
      htmltext = event
    end
  elseif event == "32028-02.htm" then
    qs:set_cond(3, true)
    q:take_items(player, WATER_BOTTLE, 1)
    q:give_items(player, HOLY_WATER_BOTTLE, 1)
    htmltext = event
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = qs:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 73 and "31350-02.htm" or "31350-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == DOMINIC then
      htmltext = q:has_quest_items(player, SCRIPTURE) and "31350-05.htm" or "31350-06.htm"
    elseif sw2 == GREMORY then
      local sw3 = qs:get_cond()
      if sw3 == 1 then
        htmltext = "32008-01.htm"
      elseif sw3 == 2 then
        htmltext = "32008-06.htm"
      elseif sw3 == 3 then
        htmltext = "32008-08.htm"
      elseif sw3 == 4 then
        htmltext = "32008-10.htm"
      end
    elseif sw2 == HOLY_GRAIL then
      if qs:is_cond(2) then
        htmltext = "32028-01.htm"
      elseif qs:get_cond() > 2 then
        htmltext = "32028-03.htm"
      end
    end
  end
  return htmltext
end)

-- As in Mobius, scriptures drop only at cond 1.
q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, 1, 3, npc)
  if qs ~= nil and rnd(1000000) < CHANCES[npc:id()] then
    q:give_item_randomly(qs:get_player(), npc, SCRIPTURE, 1, 1, 0, 1, true)
  end
end)
