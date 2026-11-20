-- Translated by questport from quests/Q00032_AnObviousLie (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(32, "Q00032_AnObviousLie", "An Obvious Lie")

-- NPCs
local GENTLER = 30094
local MAXIMILIAN = 30120
local MIKI_THE_CAT = 31706
-- Items
local SUEDE = 1866
local THREAD = 1868
local SPIRIT_ORE = 3031
local MAP = 7165
local MEDICINAL_HERB = 7166
-- Rewards
local CAT_EARS = 6843
local RACOON_EARS = 7680
local RABBIT_EARS = 7683

q:register_quest_items(MAP, MEDICINAL_HERB)
q:add_start_npc(MAXIMILIAN)
q:add_talk_id(MAXIMILIAN, GENTLER, MIKI_THE_CAT)
q:add_kill_id(20135) -- Alligator

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30120-1.htm" then
    st:start_quest()
  elseif event == "30094-1.htm" then
    st:set_cond(2, true)
    q:give_items(player, MAP, 1)
  elseif event == "31706-1.htm" then
    st:set_cond(3, true)
    q:take_items(player, MAP, 1)
  elseif event == "30094-4.htm" then
    st:set_cond(5, true)
    q:take_items(player, MEDICINAL_HERB, 20)
  elseif event == "30094-7.htm" then
    if q:get_quest_items_count(player, SPIRIT_ORE) < 500 then
      htmltext = "30094-5.htm"
    else
      st:set_cond(6, true)
      q:take_items(player, SPIRIT_ORE, 500)
    end
  elseif event == "31706-4.htm" then
    st:set_cond(7, true)
  elseif event == "30094-10.htm" then
    st:set_cond(8, true)
  elseif event == "30094-13.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
  elseif event == "cat" then
    if q:get_quest_items_count(player, THREAD) < 1000 or q:get_quest_items_count(player, SUEDE) < 500 then
      htmltext = "30094-11.htm"
    else
      htmltext = "30094-14.htm"
      q:take_items(player, SUEDE, 500)
      q:take_items(player, THREAD, 1000)
      q:give_items(player, CAT_EARS, 1)
      st:exit_quest(false, true)
    end
  elseif event == "racoon" then
    if q:get_quest_items_count(player, THREAD) < 1000 or q:get_quest_items_count(player, SUEDE) < 500 then
      htmltext = "30094-11.htm"
    else
      htmltext = "30094-14.htm"
      q:take_items(player, SUEDE, 500)
      q:take_items(player, THREAD, 1000)
      q:give_items(player, RACOON_EARS, 1)
      st:exit_quest(false, true)
    end
  elseif event == "rabbit" then
    if q:get_quest_items_count(player, THREAD) < 1000 or q:get_quest_items_count(player, SUEDE) < 500 then
      htmltext = "30094-11.htm"
    else
      htmltext = "30094-14.htm"
      q:take_items(player, SUEDE, 500)
      q:take_items(player, THREAD, 1000)
      q:give_items(player, RABBIT_EARS, 1)
      st:exit_quest(false, true)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 45 and "30120-0a.htm" or "30120-0.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == MAXIMILIAN then
      htmltext = "30120-2.htm"
    elseif sw2 == GENTLER then
      if cond == 1 then
        htmltext = "30094-0.htm"
      elseif cond == 2 or cond == 3 then
        htmltext = "30094-2.htm"
      elseif cond == 4 then
        htmltext = "30094-3.htm"
      elseif cond == 5 then
        htmltext = q:get_quest_items_count(player, SPIRIT_ORE) < 500 and "30094-5.htm" or "30094-6.htm"
      elseif cond == 6 then
        htmltext = "30094-8.htm"
      elseif cond == 7 then
        htmltext = "30094-9.htm"
      elseif cond == 8 then
        htmltext = (q:get_quest_items_count(player, THREAD) < 1000 or q:get_quest_items_count(player, SUEDE) < 500) and "30094-11.htm" or "30094-12.htm"
      end
    elseif sw2 == MIKI_THE_CAT then
      if cond == 2 then
        htmltext = "31706-0.htm"
      elseif cond > 2 and cond < 6 then
        htmltext = "31706-2.htm"
      elseif cond == 6 then
        htmltext = "31706-3.htm"
      elseif cond > 6 then
        htmltext = "31706-5.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, 3, 3, npc)
  if qs ~= nil and q:give_item_randomly(qs:get_player(), npc, MEDICINAL_HERB, 1, 1, 20, 1, true) then
    qs:set_cond(4, true)
  end
end)
