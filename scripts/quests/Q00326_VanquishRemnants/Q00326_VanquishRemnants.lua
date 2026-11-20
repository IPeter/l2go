-- Translated by questport from quests/Q00326_VanquishRemnants (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(326, "Q00326_VanquishRemnants", "Vanquish Remnants")

-- Items
local RED_CROSS_BADGE = 1359
local BLUE_CROSS_BADGE = 1360
local BLACK_CROSS_BADGE = 1361
-- Reward
local BLACK_LION_MARK = 1369

q:register_quest_items(RED_CROSS_BADGE, BLUE_CROSS_BADGE, BLACK_CROSS_BADGE)
q:add_start_npc(30435) -- Leopold
q:add_talk_id(30435)
q:add_kill_id(20053, 20437, 20058, 20436, 20061, 20439, 20063, 20066, 20438)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30435-03.htm" then
    st:start_quest()
  elseif event == "30435-07.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 21 and "30435-01.htm" or "30435-02.htm"
  elseif sw == State.STARTED then
    local redBadges = q:get_quest_items_count(player, RED_CROSS_BADGE)
    local blueBadges = q:get_quest_items_count(player, BLUE_CROSS_BADGE)
    local blackBadges = q:get_quest_items_count(player, BLACK_CROSS_BADGE)
    local badgesSum = redBadges + blueBadges + blackBadges
    if badgesSum > 0 then
      q:take_items(player, RED_CROSS_BADGE, -1)
      q:take_items(player, BLUE_CROSS_BADGE, -1)
      q:take_items(player, BLACK_CROSS_BADGE, -1)
      q:give_adena(player, redBadges * 60 + blueBadges * 65 + blackBadges * 70 + (badgesSum >= 10 and 4320 or 0), true)
      if badgesSum >= 100 then
        if not q:has_quest_items(player, BLACK_LION_MARK) then
          htmltext = "30435-06.htm"
          q:give_items(player, BLACK_LION_MARK, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        else
          htmltext = "30435-09.htm"
        end
      else
        htmltext = "30435-05.htm"
      end
    else
      htmltext = "30435-04.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == 20053 or sw == 20437 or sw == 20058 then
    if rnd(100) < 33 then
      q:give_items(player, RED_CROSS_BADGE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20436 or sw == 20061 or sw == 20439 or sw == 20063 then
    if rnd(100) < 16 then
      q:give_items(player, BLUE_CROSS_BADGE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == 20066 or sw == 20438 then
    if rnd(100) < 12 then
      q:give_items(player, BLACK_CROSS_BADGE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
