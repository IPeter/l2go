-- Translated by questport from quests/Q00402_PathOfTheHumanKnight (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(402, "Q00402_PathOfTheHumanKnight", "Path to a Human Knight")

-- NPCs
local SIR_KLAUS_VASPER = 30417
local BATHIS = 30332
local RAYMOND = 30289
local BEZIQUE = 30379
local LEVIAN = 30037
local GILBERT = 30039
local BIOTIN = 30031
local SIR_AARON_TANFORD = 30653
local SIR_COLLIN_WINDAWOOD = 30311
-- Items
local SWORD_OF_RITUAL = 1161
local COIN_OF_LORDS_1 = 1162
local COIN_OF_LORDS_2 = 1163
local COIN_OF_LORDS_3 = 1164
local COIN_OF_LORDS_4 = 1165
local COIN_OF_LORDS_5 = 1166
local COIN_OF_LORDS_6 = 1167
local GLUDIO_GUARD_MARK_1 = 1168
local BUGBEAR_NECKLACE = 1169
local EINHASAD_CHURCH_MARK_1 = 1170
local EINHASAD_CRUCIFIX = 1171
local GLUDIO_GUARD_MARK_2 = 1172
local SPIDER_LEG = 1173
local EINHASAD_CHURCH_MARK_2 = 1174
local LIZARDMAN_TOTEM = 1175
local GLUDIO_GUARD_MARK_3 = 1176
local GIANT_SPIDER_HUSK = 1177
local EINHASAD_CHURCH_MARK_3 = 1178
local HORRIBLE_SKULL = 1179
local MARK_OF_ESQUIRE = 1271

q:register_quest_items(MARK_OF_ESQUIRE, COIN_OF_LORDS_1, COIN_OF_LORDS_2, COIN_OF_LORDS_3, COIN_OF_LORDS_4, COIN_OF_LORDS_5, COIN_OF_LORDS_6, GLUDIO_GUARD_MARK_1, BUGBEAR_NECKLACE, EINHASAD_CHURCH_MARK_1, EINHASAD_CRUCIFIX, GLUDIO_GUARD_MARK_2, SPIDER_LEG, EINHASAD_CHURCH_MARK_2, LIZARDMAN_TOTEM, GLUDIO_GUARD_MARK_3, GIANT_SPIDER_HUSK, EINHASAD_CHURCH_MARK_3, LIZARDMAN_TOTEM, GLUDIO_GUARD_MARK_3, GIANT_SPIDER_HUSK, EINHASAD_CHURCH_MARK_3, HORRIBLE_SKULL)
q:add_start_npc(SIR_KLAUS_VASPER)
q:add_talk_id(SIR_KLAUS_VASPER, BATHIS, RAYMOND, BEZIQUE, LEVIAN, GILBERT, BIOTIN, SIR_AARON_TANFORD, SIR_COLLIN_WINDAWOOD)
q:add_kill_id(20775, 27024, 20038, 20043, 20050, 20030, 20027, 20024, 20103, 20106, 20108, 20404)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30417-05.htm" then
    if player:class_id() ~= ClassId.FIGHTER then
      htmltext = player:class_id() == ClassId.KNIGHT and "30417-02a.htm" or "30417-03.htm"
    elseif player:level() < 19 then
      htmltext = "30417-02.htm"
    elseif q:has_quest_items(player, SWORD_OF_RITUAL) then
      htmltext = "30417-04.htm"
    end
  elseif event == "30417-08.htm" then
    st:start_quest()
    q:give_items(player, MARK_OF_ESQUIRE, 1)
  elseif event == "30332-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:give_items(player, GLUDIO_GUARD_MARK_1, 1)
  elseif event == "30289-03.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:give_items(player, EINHASAD_CHURCH_MARK_1, 1)
  elseif event == "30379-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:give_items(player, GLUDIO_GUARD_MARK_2, 1)
  elseif event == "30037-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:give_items(player, EINHASAD_CHURCH_MARK_2, 1)
  elseif event == "30039-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:give_items(player, GLUDIO_GUARD_MARK_3, 1)
  elseif event == "30031-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:give_items(player, EINHASAD_CHURCH_MARK_3, 1)
  elseif event == "30417-13.htm" or event == "30417-14.htm" then
    local coinCount = q:get_quest_items_count(player, COIN_OF_LORDS_1) + q:get_quest_items_count(player, COIN_OF_LORDS_2) + q:get_quest_items_count(player, COIN_OF_LORDS_3) + q:get_quest_items_count(player, COIN_OF_LORDS_4) + q:get_quest_items_count(player, COIN_OF_LORDS_5) + q:get_quest_items_count(player, COIN_OF_LORDS_6)
    q:take_items(player, COIN_OF_LORDS_1, -1)
    q:take_items(player, COIN_OF_LORDS_2, -1)
    q:take_items(player, COIN_OF_LORDS_3, -1)
    q:take_items(player, COIN_OF_LORDS_4, -1)
    q:take_items(player, COIN_OF_LORDS_5, -1)
    q:take_items(player, COIN_OF_LORDS_6, -1)
    q:take_items(player, MARK_OF_ESQUIRE, 1)
    q:give_items(player, SWORD_OF_RITUAL, 1)
    q:add_exp_and_sp(player, 3200, 1500 + 1920 * (coinCount - 3))
    player:social_action(3)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30417-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == SIR_KLAUS_VASPER then
      local coins = q:get_quest_items_count(player, COIN_OF_LORDS_1) + q:get_quest_items_count(player, COIN_OF_LORDS_2) + q:get_quest_items_count(player, COIN_OF_LORDS_3) + q:get_quest_items_count(player, COIN_OF_LORDS_4) + q:get_quest_items_count(player, COIN_OF_LORDS_5) + q:get_quest_items_count(player, COIN_OF_LORDS_6)
      if coins < 3 then
        htmltext = "30417-09.htm"
      elseif coins == 3 then
        htmltext = "30417-10.htm"
      elseif coins > 3 and coins < 6 then
        htmltext = "30417-11.htm"
      elseif coins == 6 then
        htmltext = "30417-12.htm"
        q:take_items(player, COIN_OF_LORDS_1, -1)
        q:take_items(player, COIN_OF_LORDS_2, -1)
        q:take_items(player, COIN_OF_LORDS_3, -1)
        q:take_items(player, COIN_OF_LORDS_4, -1)
        q:take_items(player, COIN_OF_LORDS_5, -1)
        q:take_items(player, COIN_OF_LORDS_6, -1)
        q:take_items(player, MARK_OF_ESQUIRE, 1)
        q:give_items(player, SWORD_OF_RITUAL, 1)
        q:add_exp_and_sp(player, 3200, 7260)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == BATHIS then
      if q:has_quest_items(player, COIN_OF_LORDS_1) then
        htmltext = "30332-05.htm"
      elseif q:has_quest_items(player, GLUDIO_GUARD_MARK_1) then
        if q:get_quest_items_count(player, BUGBEAR_NECKLACE) < 10 then
          htmltext = "30332-03.htm"
        else
          htmltext = "30332-04.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
          q:take_items(player, BUGBEAR_NECKLACE, -1)
          q:take_items(player, GLUDIO_GUARD_MARK_1, 1)
          q:give_items(player, COIN_OF_LORDS_1, 1)
        end
      else
        htmltext = "30332-01.htm"
      end
    elseif sw2 == RAYMOND then
      if q:has_quest_items(player, COIN_OF_LORDS_2) then
        htmltext = "30289-06.htm"
      elseif q:has_quest_items(player, EINHASAD_CHURCH_MARK_1) then
        if q:get_quest_items_count(player, EINHASAD_CRUCIFIX) < 12 then
          htmltext = "30289-04.htm"
        else
          htmltext = "30289-05.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
          q:take_items(player, EINHASAD_CRUCIFIX, -1)
          q:take_items(player, EINHASAD_CHURCH_MARK_1, 1)
          q:give_items(player, COIN_OF_LORDS_2, 1)
        end
      else
        htmltext = "30289-01.htm"
      end
    elseif sw2 == BEZIQUE then
      if q:has_quest_items(player, COIN_OF_LORDS_3) then
        htmltext = "30379-05.htm"
      elseif q:has_quest_items(player, GLUDIO_GUARD_MARK_2) then
        if q:get_quest_items_count(player, SPIDER_LEG) < 20 then
          htmltext = "30379-03.htm"
        else
          htmltext = "30379-04.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
          q:take_items(player, SPIDER_LEG, -1)
          q:take_items(player, GLUDIO_GUARD_MARK_2, 1)
          q:give_items(player, COIN_OF_LORDS_3, 1)
        end
      else
        htmltext = "30379-01.htm"
      end
    elseif sw2 == LEVIAN then
      if q:has_quest_items(player, COIN_OF_LORDS_4) then
        htmltext = "30037-05.htm"
      elseif q:has_quest_items(player, EINHASAD_CHURCH_MARK_2) then
        if q:get_quest_items_count(player, LIZARDMAN_TOTEM) < 20 then
          htmltext = "30037-03.htm"
        else
          htmltext = "30037-04.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
          q:take_items(player, LIZARDMAN_TOTEM, -1)
          q:take_items(player, EINHASAD_CHURCH_MARK_2, 1)
          q:give_items(player, COIN_OF_LORDS_4, 1)
        end
      else
        htmltext = "30037-01.htm"
      end
    elseif sw2 == GILBERT then
      if q:has_quest_items(player, COIN_OF_LORDS_5) then
        htmltext = "30039-05.htm"
      elseif q:has_quest_items(player, GLUDIO_GUARD_MARK_3) then
        if q:get_quest_items_count(player, GIANT_SPIDER_HUSK) < 20 then
          htmltext = "30039-03.htm"
        else
          htmltext = "30039-04.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
          q:take_items(player, GIANT_SPIDER_HUSK, -1)
          q:take_items(player, GLUDIO_GUARD_MARK_3, 1)
          q:give_items(player, COIN_OF_LORDS_5, 1)
        end
      else
        htmltext = "30039-01.htm"
      end
    elseif sw2 == BIOTIN then
      if q:has_quest_items(player, COIN_OF_LORDS_6) then
        htmltext = "30031-05.htm"
      elseif q:has_quest_items(player, EINHASAD_CHURCH_MARK_3) then
        if q:get_quest_items_count(player, HORRIBLE_SKULL) < 10 then
          htmltext = "30031-03.htm"
        else
          htmltext = "30031-04.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
          q:take_items(player, HORRIBLE_SKULL, -1)
          q:take_items(player, EINHASAD_CHURCH_MARK_3, 1)
          q:give_items(player, COIN_OF_LORDS_6, 1)
        end
      else
        htmltext = "30031-01.htm"
      end
    elseif sw2 == SIR_AARON_TANFORD then
      htmltext = "30653-01.htm"
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
  if sw == 20775 then
    if q:has_quest_items(player, GLUDIO_GUARD_MARK_1) and q:get_quest_items_count(player, BUGBEAR_NECKLACE) < 10 then
      q:give_items(player, BUGBEAR_NECKLACE, 1)
      if q:get_quest_items_count(player, BUGBEAR_NECKLACE) < 10 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 27024 then
    if q:has_quest_items(player, EINHASAD_CHURCH_MARK_1) and q:get_quest_items_count(player, EINHASAD_CRUCIFIX) < 12 and rnd(10) < 5 then
      q:give_items(player, EINHASAD_CRUCIFIX, 1)
      if q:get_quest_items_count(player, EINHASAD_CRUCIFIX) < 10 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20038 or sw == 20043 or sw == 20050 then
    if q:has_quest_items(player, GLUDIO_GUARD_MARK_2) and q:get_quest_items_count(player, SPIDER_LEG) < 20 then
      q:give_items(player, SPIDER_LEG, 1)
      if q:get_quest_items_count(player, SPIDER_LEG) < 20 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20030 or sw == 20027 or sw == 20024 then
    if q:has_quest_items(player, EINHASAD_CHURCH_MARK_2) and q:get_quest_items_count(player, LIZARDMAN_TOTEM) < 20 and rnd(10) < 5 then
      q:give_items(player, LIZARDMAN_TOTEM, 1)
      if q:get_quest_items_count(player, LIZARDMAN_TOTEM) < 20 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20103 or sw == 20106 or sw == 20108 then
    if q:has_quest_items(player, GLUDIO_GUARD_MARK_3) and q:get_quest_items_count(player, GIANT_SPIDER_HUSK) < 20 and rnd(10) < 4 then
      q:give_items(player, GIANT_SPIDER_HUSK, 1)
      if q:get_quest_items_count(player, GIANT_SPIDER_HUSK) < 20 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20404 then
    if q:has_quest_items(player, EINHASAD_CHURCH_MARK_3) and q:get_quest_items_count(player, HORRIBLE_SKULL) < 10 and rnd(10) < 4 then
      q:give_items(player, HORRIBLE_SKULL, 1)
      if q:get_quest_items_count(player, HORRIBLE_SKULL) < 10 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  end
end)
