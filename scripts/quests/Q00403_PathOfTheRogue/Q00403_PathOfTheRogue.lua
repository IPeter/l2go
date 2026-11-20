-- Translated by questport from quests/Q00403_PathOfTheRogue (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(403, "Q00403_PathOfTheRogue", "Path to a Rogue")

-- NPCs
local BEZIQUE = 30379
local NETI = 30425
-- Items
local BEZIQUE_LETTER = 1180
local NETI_BOW = 1181
local NETI_DAGGER = 1182
local SPARTOI_BONES = 1183
local HORSESHOE_OF_LIGHT = 1184
local MOST_WANTED_LIST = 1185
local STOLEN_JEWELRY = 1186
local STOLEN_TOMES = 1187
local STOLEN_RING = 1188
local STOLEN_NECKLACE = 1189
local BEZIQUE_RECOMMENDATION = 1190

q:register_quest_items(BEZIQUE_LETTER, NETI_BOW, NETI_DAGGER, SPARTOI_BONES, HORSESHOE_OF_LIGHT, MOST_WANTED_LIST, STOLEN_JEWELRY, STOLEN_TOMES, STOLEN_RING, STOLEN_NECKLACE)
q:add_start_npc(BEZIQUE)
q:add_talk_id(BEZIQUE, NETI)
q:add_kill_id(20035, 20042, 20045, 20051, 20054, 20060, 27038)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30379-05.htm" then
    if player:class_id() ~= ClassId.FIGHTER then
      htmltext = player:class_id() == ClassId.ROGUE and "30379-02a.htm" or "30379-02.htm"
    elseif player:level() < 19 then
      htmltext = "30379-02.htm"
    elseif q:has_quest_items(player, BEZIQUE_RECOMMENDATION) then
      htmltext = "30379-04.htm"
    end
  elseif event == "30379-06.htm" then
    st:start_quest()
    q:give_items(player, BEZIQUE_LETTER, 1)
  elseif event == "30425-05.htm" then
    st:set_cond(2, true)
    q:give_items(player, NETI_BOW, 1)
    q:give_items(player, NETI_DAGGER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30379-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == BEZIQUE then
      if cond == 1 then
        htmltext = "30379-07.htm"
      elseif cond == 2 or cond == 3 then
        htmltext = "30379-10.htm"
      elseif cond == 4 then
        htmltext = "30379-08.htm"
        st:set_cond(5, true)
        q:take_items(player, HORSESHOE_OF_LIGHT, 1)
        q:give_items(player, MOST_WANTED_LIST, 1)
      elseif cond == 5 then
        htmltext = "30379-11.htm"
      elseif cond == 6 then
        htmltext = "30379-09.htm"
        q:take_items(player, NETI_BOW, 1)
        q:take_items(player, NETI_DAGGER, 1)
        q:take_items(player, STOLEN_JEWELRY, 1)
        q:take_items(player, STOLEN_NECKLACE, 1)
        q:take_items(player, STOLEN_RING, 1)
        q:take_items(player, STOLEN_TOMES, 1)
        q:give_items(player, BEZIQUE_RECOMMENDATION, 1)
        q:add_exp_and_sp(player, 3200, 1500)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == NETI then
      if cond == 1 then
        htmltext = "30425-01.htm"
      elseif cond == 2 then
        htmltext = "30425-06.htm"
      elseif cond == 3 then
        htmltext = "30425-07.htm"
        st:set_cond(4, true)
        q:take_items(player, SPARTOI_BONES, 10)
        q:give_items(player, HORSESHOE_OF_LIGHT, 1)
      elseif cond > 3 then
        htmltext = "30425-08.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local equippedItemId = q:get_item_equipped(player, Paperdoll.RHAND)
  if equippedItemId ~= NETI_BOW and equippedItemId ~= NETI_DAGGER then
    return
  end
  local sw = npc:id()
  if sw == 20035 or sw == 20045 or sw == 20051 then
    if st:is_cond(2) and rnd(10) < 2 then
      if q:get_quest_items_count(player, SPARTOI_BONES) < 10 then
        q:give_items(player, SPARTOI_BONES, 1)
        if q:get_quest_items_count(player, SPARTOI_BONES) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == 20042 then
    if st:is_cond(2) and rnd(10) < 3 then
      if q:get_quest_items_count(player, SPARTOI_BONES) < 10 then
        q:give_items(player, SPARTOI_BONES, 1)
        if q:get_quest_items_count(player, SPARTOI_BONES) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == 20054 or sw == 20060 then
    if st:is_cond(2) and rnd(10) < 8 then
      if q:get_quest_items_count(player, SPARTOI_BONES) < 10 then
        q:give_items(player, SPARTOI_BONES, 1)
        if q:get_quest_items_count(player, SPARTOI_BONES) >= 10 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == 27038 then
    if st:is_cond(5) then
      local randomItem = rnd(STOLEN_JEWELRY, STOLEN_NECKLACE)
      if not q:has_quest_items(player, randomItem) then
        q:give_items(player, randomItem, 1)
        if q:has_quest_items(player, STOLEN_JEWELRY, STOLEN_TOMES, STOLEN_RING, STOLEN_NECKLACE) then
          st:set_cond(6, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)
