-- Translated by questport from quests/Q00661_MakingTheHarvestGroundsSafe (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(661, "Q00661_MakingTheHarvestGroundsSafe", "Making the Harvest Grounds Safe")

-- NPC
local NORMAN = 30210
-- Monsters
local GIANT_POISON_BEE = 21095
local CLOUDY_BEAST = 21096
local YOUNG_ARANEID = 21097
-- Items
local STING_OF_GIANT_POISON_BEE = 8283
local CLOUDY_GEM = 8284
local TALON_OF_YOUNG_ARANEID = 8285
-- Reward
local ADENA = 57

q:register_quest_items(STING_OF_GIANT_POISON_BEE, CLOUDY_GEM, TALON_OF_YOUNG_ARANEID)
q:add_start_npc(NORMAN)
q:add_talk_id(NORMAN)
q:add_kill_id(GIANT_POISON_BEE, CLOUDY_BEAST, YOUNG_ARANEID)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30210-02.htm" then
    st:start_quest()
  elseif event == "30210-04.htm" then
    local item1 = q:get_quest_items_count(player, STING_OF_GIANT_POISON_BEE)
    local item2 = q:get_quest_items_count(player, CLOUDY_GEM)
    local item3 = q:get_quest_items_count(player, TALON_OF_YOUNG_ARANEID)
    local sum = 0
    sum = item1 * 57 + item2 * 56 + item3 * 60
    if item1 + item2 + item3 >= 10 then
      sum = sum + 2871
    end
    q:take_items(player, STING_OF_GIANT_POISON_BEE, item1)
    q:take_items(player, CLOUDY_GEM, item2)
    q:take_items(player, TALON_OF_YOUNG_ARANEID, item3)
    q:reward_items(player, ADENA, sum)
  elseif event == "30210-06.htm" then
    st:exit_quest(true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 21 and "30210-01a.htm" or "30210-01.htm"
  elseif sw == State.STARTED then
    htmltext = q:has_at_least_one_quest_item(player, STING_OF_GIANT_POISON_BEE, CLOUDY_GEM, TALON_OF_YOUNG_ARANEID) and "30210-03.htm" or "30210-05.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if rnd.bool() then
    q:give_items(player, npc:id() - 12812, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
