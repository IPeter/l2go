-- Translated by questport from quests/Q00231_TestOfTheMaestro (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(231, "Q00231_TestOfTheMaestro", "Test of the Maestro")

-- NPCs
local LOCKIRIN = 30531
local SPIRON = 30532
local BALANKI = 30533
local KEEF = 30534
local FILAUR = 30535
local ARIN = 30536
local TOMA = 30556
local CROTO = 30671
local DUBABAH = 30672
local LORAIN = 30673
-- Monsters
local KING_BUGBEAR = 20150
local GIANT_MIST_LEECH = 20225
local STINGER_WASP = 20229
local MARSH_SPIDER = 20233
local EVIL_EYE_LORD = 27133
-- Items
local RECOMMENDATION_OF_BALANKI = 2864
local RECOMMENDATION_OF_FILAUR = 2865
local RECOMMENDATION_OF_ARIN = 2866
local LETTER_OF_SOLDER_DETACHMENT = 2868
local PAINT_OF_KAMURU = 2869
local NECKLACE_OF_KAMURU = 2870
local PAINT_OF_TELEPORT_DEVICE = 2871
local TELEPORT_DEVICE = 2872
local ARCHITECTURE_OF_KRUMA = 2873
local REPORT_OF_KRUMA = 2874
local INGREDIENTS_OF_ANTIDOTE = 2875
local STINGER_WASP_NEEDLE = 2876
local MARSH_SPIDER_WEB = 2877
local BLOOD_OF_LEECH = 2878
local BROKEN_TELEPORT_DEVICE = 2916
-- Rewards
local MARK_OF_MAESTRO = 2867
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(RECOMMENDATION_OF_BALANKI, RECOMMENDATION_OF_FILAUR, RECOMMENDATION_OF_ARIN, LETTER_OF_SOLDER_DETACHMENT, PAINT_OF_KAMURU, NECKLACE_OF_KAMURU, PAINT_OF_TELEPORT_DEVICE, TELEPORT_DEVICE, ARCHITECTURE_OF_KRUMA, REPORT_OF_KRUMA, INGREDIENTS_OF_ANTIDOTE, STINGER_WASP_NEEDLE, MARSH_SPIDER_WEB, BLOOD_OF_LEECH, BROKEN_TELEPORT_DEVICE)
q:add_start_npc(LOCKIRIN)
q:add_talk_id(LOCKIRIN, SPIRON, BALANKI, KEEF, FILAUR, ARIN, TOMA, CROTO, DUBABAH, LORAIN)
q:add_kill_id(GIANT_MIST_LEECH, STINGER_WASP, MARSH_SPIDER, EVIL_EYE_LORD)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30531-04.htm" then
    st:start_quest()
    if not player:var_bool("secondClassChange39", false) then
      htmltext = "30531-04a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_39[player:class_id()])
      player:set_var("secondClassChange39", true)
    end
  elseif event == "30533-02.htm" then
    st:set("bCond", "1")
  elseif event == "30671-02.htm" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:give_items(player, PAINT_OF_KAMURU, 1)
  elseif event == "TELEPORT" then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, PAINT_OF_TELEPORT_DEVICE, 1)
    q:give_items(player, BROKEN_TELEPORT_DEVICE, 1)
    player:teleport(140352, -194133, -3146)
    q:start_quest_timer("spawn_bugbears", 5000, nil, player, false)
  elseif event == "30673-04.htm" then
    st:set("fCond", "2")
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    q:take_items(player, BLOOD_OF_LEECH, -1)
    q:take_items(player, INGREDIENTS_OF_ANTIDOTE, 1)
    q:take_items(player, MARSH_SPIDER_WEB, -1)
    q:take_items(player, STINGER_WASP_NEEDLE, -1)
    q:give_items(player, REPORT_OF_KRUMA, 1)
  elseif event == "spawn_bugbears" then
    -- L2Go: Mobius makes each King Bugbear hate (addDamageHate 999) and
    -- attack the player. The script API has no NPC AI calls yet, so they
    -- are only spawned around the player (aggressive monsters still notice
    -- the player on their own).
    q:add_spawn(KING_BUGBEAR, {x = 140333, y = -194153, z = -3138, heading = 0}, false, 200000)
    q:add_spawn(KING_BUGBEAR, {x = 140395, y = -194147, z = -3146, heading = 0}, false, 200000)
    q:add_spawn(KING_BUGBEAR, {x = 140304, y = -194082, z = -3157, heading = 0}, false, 200000)
    return nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.ARTISAN then
      htmltext = "30531-01.htm"
    elseif player:level() < 39 then
      htmltext = "30531-02.htm"
    else
      htmltext = "30531-03.htm"
    end
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == LOCKIRIN then
      local cond = st:get_cond()
      if cond == 1 then
        htmltext = "30531-05.htm"
      elseif cond == 2 then
        htmltext = "30531-06.htm"
        q:take_items(player, RECOMMENDATION_OF_ARIN, 1)
        q:take_items(player, RECOMMENDATION_OF_BALANKI, 1)
        q:take_items(player, RECOMMENDATION_OF_FILAUR, 1)
        q:give_items(player, MARK_OF_MAESTRO, 1)
        q:add_exp_and_sp(player, 46000, 5900)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == SPIRON then
      htmltext = "30532-01.htm"
    elseif sw2 == KEEF then
      htmltext = "30534-01.htm"
    elseif sw2 == BALANKI then
      local bCond = st:get_int("bCond")
      if bCond == 0 then
        htmltext = "30533-01.htm"
      elseif bCond == 1 then
        htmltext = "30533-03.htm"
      elseif bCond == 2 then
        htmltext = "30533-04.htm"
        st:set("bCond", "3")
        q:take_items(player, LETTER_OF_SOLDER_DETACHMENT, 1)
        q:give_items(player, RECOMMENDATION_OF_BALANKI, 1)
        if q:has_quest_items(player, RECOMMENDATION_OF_ARIN, RECOMMENDATION_OF_FILAUR) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif bCond == 3 then
        htmltext = "30533-05.htm"
      end
    elseif sw2 == CROTO then
      local bCond = st:get_int("bCond")
      if bCond == 1 then
        if not q:has_quest_items(player, PAINT_OF_KAMURU) then
          htmltext = "30671-01.htm"
        elseif not q:has_quest_items(player, NECKLACE_OF_KAMURU) then
          htmltext = "30671-03.htm"
        else
          htmltext = "30671-04.htm"
          st:set("bCond", "2")
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, NECKLACE_OF_KAMURU, 1)
          q:take_items(player, PAINT_OF_KAMURU, 1)
          q:give_items(player, LETTER_OF_SOLDER_DETACHMENT, 1)
        end
      elseif bCond > 1 then
        htmltext = "30671-05.htm"
      end
    elseif sw2 == DUBABAH then
      htmltext = "30672-01.htm"
    elseif sw2 == ARIN then
      local aCond = st:get_int("aCond")
      if aCond == 0 then
        htmltext = "30536-01.htm"
        st:set("aCond", "1")
        q:give_items(player, PAINT_OF_TELEPORT_DEVICE, 1)
      elseif aCond == 1 then
        htmltext = "30536-02.htm"
      elseif aCond == 2 then
        htmltext = "30536-03.htm"
        st:set("aCond", "3")
        q:take_items(player, TELEPORT_DEVICE, -1)
        q:give_items(player, RECOMMENDATION_OF_ARIN, 1)
        if q:has_quest_items(player, RECOMMENDATION_OF_BALANKI, RECOMMENDATION_OF_FILAUR) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif aCond == 3 then
        htmltext = "30536-04.htm"
      end
    elseif sw2 == TOMA then
      local aCond = st:get_int("aCond")
      if aCond == 1 then
        if not q:has_quest_items(player, BROKEN_TELEPORT_DEVICE) then
          htmltext = "30556-01.htm"
        elseif not q:has_quest_items(player, TELEPORT_DEVICE) then
          htmltext = "30556-06.htm"
          st:set("aCond", "2")
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:take_items(player, BROKEN_TELEPORT_DEVICE, 1)
          q:give_items(player, TELEPORT_DEVICE, 5)
        end
      elseif aCond > 1 then
        htmltext = "30556-07.htm"
      end
    elseif sw2 == FILAUR then
      local fCond = st:get_int("fCond")
      if fCond == 0 then
        htmltext = "30535-01.htm"
        st:set("fCond", "1")
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, ARCHITECTURE_OF_KRUMA, 1)
      elseif fCond == 1 then
        htmltext = "30535-02.htm"
      elseif fCond == 2 then
        htmltext = "30535-03.htm"
        st:set("fCond", "3")
        q:take_items(player, REPORT_OF_KRUMA, 1)
        q:give_items(player, RECOMMENDATION_OF_FILAUR, 1)
        if q:has_quest_items(player, RECOMMENDATION_OF_BALANKI, RECOMMENDATION_OF_ARIN) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif fCond == 3 then
        htmltext = "30535-04.htm"
      end
    elseif sw2 == LORAIN then
      local fCond = st:get_int("fCond")
      if fCond == 1 then
        if not q:has_quest_items(player, REPORT_OF_KRUMA) then
          if not q:has_quest_items(player, INGREDIENTS_OF_ANTIDOTE) then
            htmltext = "30673-01.htm"
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
            q:take_items(player, ARCHITECTURE_OF_KRUMA, 1)
            q:give_items(player, INGREDIENTS_OF_ANTIDOTE, 1)
          elseif q:get_quest_items_count(player, STINGER_WASP_NEEDLE) < 10 or q:get_quest_items_count(player, MARSH_SPIDER_WEB) < 10 or q:get_quest_items_count(player, BLOOD_OF_LEECH) < 10 then
            htmltext = "30673-02.htm"
          else
            htmltext = "30673-03.htm"
          end
        end
      elseif fCond > 1 then
        htmltext = "30673-05.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  local sw = npc:id()
  if sw == GIANT_MIST_LEECH then
    if q:has_quest_items(player, INGREDIENTS_OF_ANTIDOTE) and q:get_quest_items_count(player, BLOOD_OF_LEECH) < 10 then
      q:give_items(player, BLOOD_OF_LEECH, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == STINGER_WASP then
    if q:has_quest_items(player, INGREDIENTS_OF_ANTIDOTE) and q:get_quest_items_count(player, STINGER_WASP_NEEDLE) < 10 then
      q:give_items(player, STINGER_WASP_NEEDLE, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == MARSH_SPIDER then
    if q:has_quest_items(player, INGREDIENTS_OF_ANTIDOTE) and q:get_quest_items_count(player, MARSH_SPIDER_WEB) < 10 then
      q:give_items(player, MARSH_SPIDER_WEB, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif sw == EVIL_EYE_LORD then
    if q:has_quest_items(player, PAINT_OF_KAMURU) and not q:has_quest_items(player, NECKLACE_OF_KAMURU) then
      q:give_items(player, NECKLACE_OF_KAMURU, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
