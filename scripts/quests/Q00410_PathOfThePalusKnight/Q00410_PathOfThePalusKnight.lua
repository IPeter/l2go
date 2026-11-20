-- Translated by questport from quests/Q00410_PathOfThePalusKnight (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(410, "Q00410_PathOfThePalusKnight", "Path to a Palus Knight")

-- NPCs
local KALINTA = 30422
local VIRGIL = 30329
-- Monsters
local POISON_SPIDER = 20038
local ARACHNID_TRACKER = 20043
local LYCANTHROPE = 20049
-- Items
local PALUS_TALISMAN = 1237
local LYCANTHROPE_SKULL = 1238
local VIRGIL_LETTER = 1239
local MORTE_TALISMAN = 1240
local PREDATOR_CARAPACE = 1241
local ARACHNID_TRACKER_SILK = 1242
local COFFIN_OF_ETERNAL_REST = 1243
local GAZE_OF_ABYSS = 1244

q:register_quest_items(PALUS_TALISMAN, LYCANTHROPE_SKULL, VIRGIL_LETTER, MORTE_TALISMAN, PREDATOR_CARAPACE, ARACHNID_TRACKER_SILK, COFFIN_OF_ETERNAL_REST)
q:add_start_npc(VIRGIL)
q:add_talk_id(VIRGIL, KALINTA)
q:add_kill_id(POISON_SPIDER, ARACHNID_TRACKER, LYCANTHROPE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30329-05.htm" then
    if player:class_id() ~= ClassId.DARK_FIGHTER then
      htmltext = player:class_id() == ClassId.PALUS_KNIGHT and "30329-02a.htm" or "30329-03.htm"
    elseif player:level() < 19 then
      htmltext = "30329-02.htm"
    elseif q:has_quest_items(player, GAZE_OF_ABYSS) then
      htmltext = "30329-04.htm"
    end
  elseif event == "30329-06.htm" then
    st:start_quest()
    q:give_items(player, PALUS_TALISMAN, 1)
  elseif event == "30329-10.htm" then
    st:set_cond(3, true)
    q:take_items(player, LYCANTHROPE_SKULL, -1)
    q:take_items(player, PALUS_TALISMAN, 1)
    q:give_items(player, VIRGIL_LETTER, 1)
  elseif event == "30422-02.htm" then
    st:set_cond(4, true)
    q:take_items(player, VIRGIL_LETTER, 1)
    q:give_items(player, MORTE_TALISMAN, 1)
  elseif event == "30422-06.htm" then
    st:set_cond(6, true)
    q:take_items(player, ARACHNID_TRACKER_SILK, -1)
    q:take_items(player, MORTE_TALISMAN, 1)
    q:take_items(player, PREDATOR_CARAPACE, -1)
    q:give_items(player, COFFIN_OF_ETERNAL_REST, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30329-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == VIRGIL then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, LYCANTHROPE_SKULL) and "30329-07.htm" or "30329-08.htm"
      elseif cond == 2 then
        htmltext = "30329-09.htm"
      elseif cond > 2 and cond < 6 then
        htmltext = "30329-12.htm"
      elseif cond == 6 then
        htmltext = "30329-11.htm"
        q:take_items(player, COFFIN_OF_ETERNAL_REST, 1)
        q:give_items(player, GAZE_OF_ABYSS, 1)
        q:add_exp_and_sp(player, 3200, 1500)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == KALINTA then
      if cond == 3 then
        htmltext = "30422-01.htm"
      elseif cond == 4 then
        if not q:has_quest_items(player, ARACHNID_TRACKER_SILK) or not q:has_quest_items(player, PREDATOR_CARAPACE) then
          htmltext = "30422-03.htm"
        else
          htmltext = "30422-04.htm"
        end
      elseif cond == 5 then
        htmltext = "30422-05.htm"
      elseif cond == 6 then
        htmltext = "30422-06.htm"
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
  if sw == LYCANTHROPE then
    if st:is_cond(1) then
      if q:get_quest_items_count(player, LYCANTHROPE_SKULL) < 13 then
        q:give_items(player, LYCANTHROPE_SKULL, 1)
        if q:get_quest_items_count(player, LYCANTHROPE_SKULL) >= 13 then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == ARACHNID_TRACKER then
    if st:is_cond(4) then
      if q:get_quest_items_count(player, ARACHNID_TRACKER_SILK) < 5 then
        q:give_items(player, ARACHNID_TRACKER_SILK, 1)
        if q:get_quest_items_count(player, ARACHNID_TRACKER_SILK) >= 5 then
          if q:has_quest_items(player, PREDATOR_CARAPACE) then
            st:set_cond(5, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == POISON_SPIDER then
    if st:is_cond(4) then
      if not q:has_quest_items(player, PREDATOR_CARAPACE) then
        q:give_items(player, PREDATOR_CARAPACE, 1)
        if q:get_quest_items_count(player, ARACHNID_TRACKER_SILK) >= 5 then
          st:set_cond(5, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)
