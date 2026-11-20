-- Translated by questport from quests/Q00412_PathOfTheDarkWizard (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(412, "Q00412_PathOfTheDarkWizard", "Path to a Dark Wizard")

-- NPCs
local VARIKA = 30421
local CHARKEREN = 30415
local ANNIKA = 30418
local ARKENIA = 30419
-- Items
local SEED_OF_ANGER = 1253
local SEED_OF_DESPAIR = 1254
local SEED_OF_HORROR = 1255
local SEED_OF_LUNACY = 1256
local FAMILY_REMAINS = 1257
local VARIKA_LIQUOR = 1258
local KNEE_BONE = 1259
local HEART_OF_LUNACY = 1260
local JEWEL_OF_DARKNESS = 1261
local LUCKY_KEY = 1277
local CANDLE = 1278
local HUB_SCENT = 1279

q:register_quest_items(SEED_OF_ANGER, SEED_OF_DESPAIR, SEED_OF_HORROR, SEED_OF_LUNACY, FAMILY_REMAINS, VARIKA_LIQUOR, KNEE_BONE, HEART_OF_LUNACY, LUCKY_KEY, CANDLE, HUB_SCENT)
q:add_start_npc(VARIKA)
q:add_talk_id(VARIKA, CHARKEREN, ANNIKA, ARKENIA)
q:add_kill_id(20015, 20022, 20045, 20517, 20518)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30421-05.htm" then
    if player:class_id() ~= ClassId.DARK_MAGE then
      htmltext = player:class_id() == ClassId.DARK_WIZARD and "30421-02a.htm" or "30421-03.htm"
    elseif player:level() < 19 then
      htmltext = "30421-02.htm"
    elseif q:has_quest_items(player, JEWEL_OF_DARKNESS) then
      htmltext = "30421-04.htm"
    else
      st:start_quest()
      q:give_items(player, SEED_OF_DESPAIR, 1)
    end
  elseif event == "30421-07.htm" then
    if q:has_quest_items(player, SEED_OF_ANGER) then
      htmltext = "30421-06.htm"
    elseif q:has_quest_items(player, LUCKY_KEY) then
      htmltext = "30421-08.htm"
    elseif q:get_quest_items_count(player, FAMILY_REMAINS) == 3 then
      htmltext = "30421-18.htm"
    end
  elseif event == "30421-10.htm" then
    if q:has_quest_items(player, SEED_OF_HORROR) then
      htmltext = "30421-09.htm"
    elseif q:get_quest_items_count(player, KNEE_BONE) == 2 then
      htmltext = "30421-19.htm"
    end
  elseif event == "30421-13.htm" then
    if q:has_quest_items(player, SEED_OF_LUNACY) then
      htmltext = "30421-12.htm"
    end
  elseif event == "30415-03.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:give_items(player, LUCKY_KEY, 1)
  elseif event == "30418-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:give_items(player, CANDLE, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30421-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == VARIKA then
      if q:has_quest_items(player, SEED_OF_ANGER, SEED_OF_HORROR, SEED_OF_LUNACY) then
        htmltext = "30421-16.htm"
        q:take_items(player, SEED_OF_ANGER, 1)
        q:take_items(player, SEED_OF_DESPAIR, 1)
        q:take_items(player, SEED_OF_HORROR, 1)
        q:take_items(player, SEED_OF_LUNACY, 1)
        q:give_items(player, JEWEL_OF_DARKNESS, 1)
        q:add_exp_and_sp(player, 3200, 1650)
        player:social_action(3)
        st:exit_quest(true, true)
      else
        htmltext = "30421-17.htm"
      end
    elseif sw2 == CHARKEREN then
      if q:has_quest_items(player, SEED_OF_ANGER) then
        htmltext = "30415-06.htm"
      elseif not q:has_quest_items(player, LUCKY_KEY) then
        htmltext = "30415-01.htm"
      elseif q:get_quest_items_count(player, FAMILY_REMAINS) == 3 then
        htmltext = "30415-05.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:take_items(player, FAMILY_REMAINS, -1)
        q:take_items(player, LUCKY_KEY, 1)
        q:give_items(player, SEED_OF_ANGER, 1)
      else
        htmltext = "30415-04.htm"
      end
    elseif sw2 == ANNIKA then
      if q:has_quest_items(player, SEED_OF_HORROR) then
        htmltext = "30418-04.htm"
      elseif not q:has_quest_items(player, CANDLE) then
        htmltext = "30418-01.htm"
      elseif q:get_quest_items_count(player, KNEE_BONE) == 2 then
        htmltext = "30418-04.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:take_items(player, CANDLE, 1)
        q:take_items(player, KNEE_BONE, -1)
        q:give_items(player, SEED_OF_HORROR, 1)
      else
        htmltext = "30418-03.htm"
      end
    elseif sw2 == ARKENIA then
      if q:has_quest_items(player, SEED_OF_LUNACY) then
        htmltext = "30419-03.htm"
      elseif not q:has_quest_items(player, HUB_SCENT) then
        htmltext = "30419-01.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:give_items(player, HUB_SCENT, 1)
      elseif q:get_quest_items_count(player, HEART_OF_LUNACY) == 3 then
        htmltext = "30419-03.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:take_items(player, HEART_OF_LUNACY, -1)
        q:take_items(player, HUB_SCENT, 1)
        q:give_items(player, SEED_OF_LUNACY, 1)
      else
        htmltext = "30419-02.htm"
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
  local sw = npc:id()
  if sw == 20015 then
    if q:has_quest_items(player, LUCKY_KEY) and q:get_quest_items_count(player, FAMILY_REMAINS) < 3 and rnd(10) < 5 then
      q:give_items(player, FAMILY_REMAINS, 1)
      if q:get_quest_items_count(player, FAMILY_REMAINS) < 3 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20022 or sw == 20517 or sw == 20518 then
    if q:has_quest_items(player, CANDLE) and q:get_quest_items_count(player, KNEE_BONE) < 2 and rnd(10) < 5 then
      q:give_items(player, KNEE_BONE, 1)
      if q:get_quest_items_count(player, KNEE_BONE) < 2 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif sw == 20045 then
    if q:has_quest_items(player, HUB_SCENT) and q:get_quest_items_count(player, HEART_OF_LUNACY) < 3 and rnd(10) < 5 then
      q:give_items(player, HEART_OF_LUNACY, 1)
      if q:get_quest_items_count(player, HEART_OF_LUNACY) < 3 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  end
end)
