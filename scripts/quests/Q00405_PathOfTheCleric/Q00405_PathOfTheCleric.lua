-- Translated by questport from quests/Q00405_PathOfTheCleric (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(405, "Q00405_PathOfTheCleric", "Path to a Cleric")

-- NPCs
local GALLINT = 30017
local ZIGAUNT = 30022
local VIVYAN = 30030
local PRAGA = 30333
local SIMPLON = 30253
local LIONEL = 30408
-- Items
local LETTER_OF_ORDER_1 = 1191
local LETTER_OF_ORDER_2 = 1192
local LIONEL_BOOK = 1193
local BOOK_OF_VIVYAN = 1194
local BOOK_OF_SIMPLON = 1195
local BOOK_OF_PRAGA = 1196
local CERTIFICATE_OF_GALLINT = 1197
local PENDANT_OF_MOTHER = 1198
local NECKLACE_OF_MOTHER = 1199
local LIONEL_COVENANT = 1200
-- Reward
local MARK_OF_FATE = 1201

q:register_quest_items(LETTER_OF_ORDER_1, BOOK_OF_SIMPLON, BOOK_OF_PRAGA, BOOK_OF_VIVYAN, NECKLACE_OF_MOTHER, PENDANT_OF_MOTHER, LETTER_OF_ORDER_2, LIONEL_BOOK, CERTIFICATE_OF_GALLINT, LIONEL_COVENANT)
q:add_start_npc(ZIGAUNT)
q:add_talk_id(ZIGAUNT, SIMPLON, PRAGA, VIVYAN, LIONEL, GALLINT)
q:add_kill_id(20029, 20026)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30022-05.htm" then
    st:start_quest()
    q:give_items(player, LETTER_OF_ORDER_1, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:class_id() ~= ClassId.MAGE then
      htmltext = player:class_id() == ClassId.CLERIC and "30022-02a.htm" or "30022-02.htm"
    elseif player:level() < 19 then
      htmltext = "30022-03.htm"
    elseif q:has_quest_items(player, MARK_OF_FATE) then
      htmltext = "30022-04.htm"
    else
      htmltext = "30022-01.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ZIGAUNT then
      if cond == 1 then
        htmltext = "30022-06.htm"
      elseif cond == 2 then
        htmltext = "30022-08.htm"
        st:set_cond(3, true)
        q:take_items(player, BOOK_OF_PRAGA, 1)
        q:take_items(player, BOOK_OF_VIVYAN, 1)
        q:take_items(player, BOOK_OF_SIMPLON, 3)
        q:take_items(player, LETTER_OF_ORDER_1, 1)
        q:give_items(player, LETTER_OF_ORDER_2, 1)
      elseif cond > 2 and cond < 6 then
        htmltext = "30022-07.htm"
      elseif cond == 6 then
        htmltext = "30022-09.htm"
        q:take_items(player, LETTER_OF_ORDER_2, 1)
        q:take_items(player, LIONEL_COVENANT, 1)
        q:give_items(player, MARK_OF_FATE, 1)
        q:add_exp_and_sp(player, 3200, 5610)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == SIMPLON then
      if cond == 1 and not q:has_quest_items(player, BOOK_OF_SIMPLON) then
        htmltext = "30253-01.htm"
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, BOOK_OF_SIMPLON, 3)
      elseif cond > 1 or q:has_quest_items(player, BOOK_OF_SIMPLON) then
        htmltext = "30253-02.htm"
      end
    elseif sw2 == PRAGA then
      if cond == 1 then
        if not q:has_quest_items(player, BOOK_OF_PRAGA) and not q:has_quest_items(player, NECKLACE_OF_MOTHER) and q:has_quest_items(player, BOOK_OF_SIMPLON) then
          htmltext = "30333-01.htm"
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:give_items(player, NECKLACE_OF_MOTHER, 1)
        elseif not q:has_quest_items(player, PENDANT_OF_MOTHER) then
          htmltext = "30333-02.htm"
        elseif q:has_quest_items(player, PENDANT_OF_MOTHER) then
          htmltext = "30333-03.htm"
          q:take_items(player, NECKLACE_OF_MOTHER, 1)
          q:take_items(player, PENDANT_OF_MOTHER, 1)
          q:give_items(player, BOOK_OF_PRAGA, 1)
          if q:has_quest_items(player, BOOK_OF_VIVYAN) then
            st:set_cond(2, true)
          else
            q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          end
        end
      elseif cond > 1 or q:has_quest_items(player, BOOK_OF_PRAGA) then
        htmltext = "30333-04.htm"
      end
    elseif sw2 == VIVYAN then
      if cond == 1 and not q:has_quest_items(player, BOOK_OF_VIVYAN) and q:has_quest_items(player, BOOK_OF_SIMPLON) then
        htmltext = "30030-01.htm"
        q:give_items(player, BOOK_OF_VIVYAN, 1)
        if q:has_quest_items(player, BOOK_OF_PRAGA) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      elseif cond > 1 or q:has_quest_items(player, BOOK_OF_VIVYAN) then
        htmltext = "30030-02.htm"
      end
    elseif sw2 == LIONEL then
      if cond < 3 then
        htmltext = "30408-02.htm"
      elseif cond == 3 then
        htmltext = "30408-01.htm"
        st:set_cond(4, true)
        q:give_items(player, LIONEL_BOOK, 1)
      elseif cond == 4 then
        htmltext = "30408-03.htm"
      elseif cond == 5 then
        htmltext = "30408-04.htm"
        st:set_cond(6, true)
        q:take_items(player, CERTIFICATE_OF_GALLINT, 1)
        q:give_items(player, LIONEL_COVENANT, 1)
      elseif cond == 6 then
        htmltext = "30408-05.htm"
      end
    elseif sw2 == GALLINT then
      if cond == 4 then
        htmltext = "30017-01.htm"
        st:set_cond(5, true)
        q:take_items(player, LIONEL_BOOK, 1)
        q:give_items(player, CERTIFICATE_OF_GALLINT, 1)
      elseif cond > 4 then
        htmltext = "30017-02.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if q:has_quest_items(player, NECKLACE_OF_MOTHER) and not q:has_quest_items(player, PENDANT_OF_MOTHER) then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    q:give_items(player, PENDANT_OF_MOTHER, 1)
  end
end)
