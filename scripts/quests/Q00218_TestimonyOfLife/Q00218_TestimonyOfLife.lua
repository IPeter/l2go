-- Translated by questport from quests/Q00218_TestimonyOfLife (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(218, "Q00218_TestimonyOfLife", "Testimony of Life")

-- NPCs
local ASTERIOS = 30154
local PUSHKIN = 30300
local THALIA = 30371
local ADONIUS = 30375
local ARKENIA = 30419
local CARDIEN = 30460
local ISAEL = 30655
-- Items
local TALINS_SPEAR = 3026
local CARDIEN_LETTER = 3141
local CAMOMILE_CHARM = 3142
local HIERARCH_LETTER = 3143
local MOONFLOWER_CHARM = 3144
local GRAIL_DIAGRAM = 3145
local THALIA_LETTER_1 = 3146
local THALIA_LETTER_2 = 3147
local THALIA_INSTRUCTIONS = 3148
local PUSHKIN_LIST = 3149
local PURE_MITHRIL_CUP = 3150
local ARKENIA_CONTRACT = 3151
local ARKENIA_INSTRUCTIONS = 3152
local ADONIUS_LIST = 3153
local ANDARIEL_SCRIPTURE_COPY = 3154
local STARDUST = 3155
local ISAEL_INSTRUCTIONS = 3156
local ISAEL_LETTER = 3157
local GRAIL_OF_PURITY = 3158
local TEARS_OF_UNICORN = 3159
local WATER_OF_LIFE = 3160
local PURE_MITHRIL_ORE = 3161
local ANT_SOLDIER_ACID = 3162
local WYRM_TALON = 3163
local SPIDER_ICHOR = 3164
local HARPY_DOWN = 3165
local TALINS_PIECES = {3166, 3167, 3168, 3169, 3170, 3171}
-- Rewards
local MARK_OF_LIFE = 3140
local DIMENSIONAL_DIAMOND = 7562

q:register_quest_items(TALINS_SPEAR, CARDIEN_LETTER, CAMOMILE_CHARM, HIERARCH_LETTER, MOONFLOWER_CHARM, GRAIL_DIAGRAM, THALIA_LETTER_1, THALIA_LETTER_2, THALIA_INSTRUCTIONS, PUSHKIN_LIST, PURE_MITHRIL_CUP, ARKENIA_CONTRACT, ARKENIA_INSTRUCTIONS, ADONIUS_LIST, ANDARIEL_SCRIPTURE_COPY, STARDUST, ISAEL_INSTRUCTIONS, ISAEL_LETTER, GRAIL_OF_PURITY, TEARS_OF_UNICORN, WATER_OF_LIFE, PURE_MITHRIL_ORE, ANT_SOLDIER_ACID, WYRM_TALON, SPIDER_ICHOR, HARPY_DOWN, 3166, 3167, 3168, 3169, 3170, 3171)
q:add_start_npc(CARDIEN)
q:add_talk_id(ASTERIOS, PUSHKIN, THALIA, ADONIUS, ARKENIA, CARDIEN, ISAEL)
q:add_kill_id(20145, 20176, 20233, 27077, 20550, 20581, 20582, 20082, 20084, 20086, 20087, 20088)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30460-04.htm" then
    st:start_quest()
    q:give_items(player, CARDIEN_LETTER, 1)
    if not player:var_bool("secondClassChange37", false) then
      htmltext = "30460-04a.htm"
      q:give_items(player, DIMENSIONAL_DIAMOND, quest.DF_REWARD_37[player:race()])
      player:set_var("secondClassChange37", true)
    end
  elseif event == "30154-07.htm" then
    st:set_cond(2, true)
    q:take_items(player, CARDIEN_LETTER, 1)
    q:give_items(player, HIERARCH_LETTER, 1)
    q:give_items(player, MOONFLOWER_CHARM, 1)
  elseif event == "30371-03.htm" then
    st:set_cond(3, true)
    q:take_items(player, HIERARCH_LETTER, 1)
    q:give_items(player, GRAIL_DIAGRAM, 1)
  elseif event == "30371-11.htm" then
    q:take_items(player, STARDUST, 1)
    if player:level() < 38 then
      htmltext = "30371-10.htm"
      st:set_cond(13, true)
      q:give_items(player, THALIA_INSTRUCTIONS, 1)
    else
      st:set_cond(14, true)
      q:give_items(player, THALIA_LETTER_2, 1)
    end
  elseif event == "30300-06.htm" then
    st:set_cond(4, true)
    q:take_items(player, GRAIL_DIAGRAM, 1)
    q:give_items(player, PUSHKIN_LIST, 1)
  elseif event == "30300-10.htm" then
    st:set_cond(6, true)
    q:take_items(player, PUSHKIN_LIST, 1)
    q:take_items(player, ANT_SOLDIER_ACID, -1)
    q:take_items(player, PURE_MITHRIL_ORE, -1)
    q:take_items(player, WYRM_TALON, -1)
    q:give_items(player, PURE_MITHRIL_CUP, 1)
  elseif event == "30419-04.htm" then
    st:set_cond(8, true)
    q:take_items(player, THALIA_LETTER_1, 1)
    q:give_items(player, ARKENIA_CONTRACT, 1)
    q:give_items(player, ARKENIA_INSTRUCTIONS, 1)
  elseif event == "30375-02.htm" then
    st:set_cond(9, true)
    q:take_items(player, ARKENIA_INSTRUCTIONS, 1)
    q:give_items(player, ADONIUS_LIST, 1)
  elseif event == "30655-02.htm" then
    st:set_cond(15, true)
    q:take_items(player, THALIA_LETTER_2, 1)
    q:give_items(player, ISAEL_INSTRUCTIONS, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "30460-01.htm"
    elseif player:level() < 37 or ClassId.level(player:class_id()) ~= 1 then
      htmltext = "30460-02.htm"
    else
      htmltext = "30460-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ASTERIOS then
      if cond == 1 then
        htmltext = "30154-01.htm"
      elseif cond == 2 then
        htmltext = "30154-08.htm"
      elseif cond == 20 then
        htmltext = "30154-09.htm"
        st:set_cond(21, true)
        q:take_items(player, MOONFLOWER_CHARM, 1)
        q:take_items(player, WATER_OF_LIFE, 1)
        q:give_items(player, CAMOMILE_CHARM, 1)
      elseif cond == 21 then
        htmltext = "30154-10.htm"
      end
    elseif sw2 == PUSHKIN then
      if cond == 3 then
        htmltext = "30300-01.htm"
      elseif cond == 4 then
        htmltext = "30300-07.htm"
      elseif cond == 5 then
        htmltext = "30300-08.htm"
      elseif cond == 6 then
        htmltext = "30300-11.htm"
      elseif cond > 6 then
        htmltext = "30300-12.htm"
      end
    elseif sw2 == THALIA then
      if cond == 2 then
        htmltext = "30371-01.htm"
      elseif cond == 3 then
        htmltext = "30371-04.htm"
      elseif cond > 3 and cond < 6 then
        htmltext = "30371-05.htm"
      elseif cond == 6 then
        htmltext = "30371-06.htm"
        st:set_cond(7, true)
        q:take_items(player, PURE_MITHRIL_CUP, 1)
        q:give_items(player, THALIA_LETTER_1, 1)
      elseif cond == 7 then
        htmltext = "30371-07.htm"
      elseif cond > 7 and cond < 12 then
        htmltext = "30371-08.htm"
      elseif cond == 12 then
        htmltext = "30371-09.htm"
      elseif cond == 13 then
        if player:level() < 38 then
          htmltext = "30371-12.htm"
        else
          htmltext = "30371-13.htm"
          st:set_cond(14, true)
          q:take_items(player, THALIA_INSTRUCTIONS, 1)
          q:give_items(player, THALIA_LETTER_2, 1)
        end
      elseif cond == 14 then
        htmltext = "30371-14.htm"
      elseif cond > 14 and cond < 17 then
        htmltext = "30371-15.htm"
      elseif cond == 17 then
        htmltext = "30371-16.htm"
        st:set_cond(18, true)
        q:take_items(player, ISAEL_LETTER, 1)
        q:give_items(player, GRAIL_OF_PURITY, 1)
      elseif cond == 18 then
        htmltext = "30371-17.htm"
      elseif cond == 19 then
        htmltext = "30371-18.htm"
        st:set_cond(20, true)
        q:take_items(player, TEARS_OF_UNICORN, 1)
        q:give_items(player, WATER_OF_LIFE, 1)
      elseif cond > 19 then
        htmltext = "30371-19.htm"
      end
    elseif sw2 == ADONIUS then
      if cond == 8 then
        htmltext = "30375-01.htm"
      elseif cond == 9 then
        htmltext = "30375-03.htm"
      elseif cond == 10 then
        htmltext = "30375-04.htm"
        st:set_cond(11, true)
        q:take_items(player, ADONIUS_LIST, 1)
        q:take_items(player, HARPY_DOWN, -1)
        q:take_items(player, SPIDER_ICHOR, -1)
        q:give_items(player, ANDARIEL_SCRIPTURE_COPY, 1)
      elseif cond == 11 then
        htmltext = "30375-05.htm"
      elseif cond > 11 then
        htmltext = "30375-06.htm"
      end
    elseif sw2 == ARKENIA then
      if cond == 7 then
        htmltext = "30419-01.htm"
      elseif cond > 7 and cond < 11 then
        htmltext = "30419-05.htm"
      elseif cond == 11 then
        htmltext = "30419-06.htm"
        st:set_cond(12, true)
        q:take_items(player, ANDARIEL_SCRIPTURE_COPY, 1)
        q:take_items(player, ARKENIA_CONTRACT, 1)
        q:give_items(player, STARDUST, 1)
      elseif cond == 12 then
        htmltext = "30419-07.htm"
      elseif cond > 12 then
        htmltext = "30419-08.htm"
      end
    elseif sw2 == CARDIEN then
      if cond == 1 then
        htmltext = "30460-05.htm"
      elseif cond > 1 and cond < 21 then
        htmltext = "30460-06.htm"
      elseif cond == 21 then
        htmltext = "30460-07.htm"
        q:take_items(player, CAMOMILE_CHARM, 1)
        q:give_items(player, MARK_OF_LIFE, 1)
        q:add_exp_and_sp(player, 104591, 11250)
        player:social_action(3)
        st:exit_quest(false, true)
      end
    elseif sw2 == ISAEL then
      if cond == 14 then
        htmltext = "30655-01.htm"
      elseif cond == 15 then
        htmltext = "30655-03.htm"
      elseif cond == 16 then
        if q:has_quest_items(player, unpack(TALINS_PIECES)) then
          htmltext = "30655-04.htm"
          st:set_cond(17, true)
          for _, itemId in ipairs(TALINS_PIECES) do
            q:take_items(player, itemId, 1)
          end
          q:take_items(player, ISAEL_INSTRUCTIONS, 1)
          q:give_items(player, ISAEL_LETTER, 1)
          q:give_items(player, TALINS_SPEAR, 1)
        else
          htmltext = "30655-03.htm"
        end
      elseif cond == 17 then
        htmltext = "30655-05.htm"
      elseif cond > 17 then
        htmltext = "30655-06.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == 20550 then
    if st:is_cond(4) and q:get_quest_items_count(player, PURE_MITHRIL_ORE) < 10 and rnd.bool() then
      q:give_items(player, PURE_MITHRIL_ORE, 1)
      if q:get_quest_items_count(player, PURE_MITHRIL_ORE) >= 10 and q:get_quest_items_count(player, WYRM_TALON) >= 20 and q:get_quest_items_count(player, ANT_SOLDIER_ACID) >= 20 then
        st:set_cond(5, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20176 then
    if st:is_cond(4) and q:get_quest_items_count(player, WYRM_TALON) < 20 and rnd.bool() then
      q:give_items(player, WYRM_TALON, 1)
      if q:get_quest_items_count(player, WYRM_TALON) >= 20 and q:get_quest_items_count(player, PURE_MITHRIL_ORE) >= 10 and q:get_quest_items_count(player, ANT_SOLDIER_ACID) >= 20 then
        st:set_cond(5, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20082 or sw == 20084 or sw == 20086 or sw == 20087 or sw == 20088 then
    if st:is_cond(4) and q:get_quest_items_count(player, ANT_SOLDIER_ACID) < 20 and rnd(10) < 8 then
      q:give_items(player, ANT_SOLDIER_ACID, 1)
      if q:get_quest_items_count(player, ANT_SOLDIER_ACID) >= 20 and q:get_quest_items_count(player, PURE_MITHRIL_ORE) >= 10 and q:get_quest_items_count(player, WYRM_TALON) >= 20 then
        st:set_cond(5, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20233 then
    if st:is_cond(9) and q:get_quest_items_count(player, SPIDER_ICHOR) < 20 and rnd.bool() then
      q:give_items(player, SPIDER_ICHOR, 1)
      if q:get_quest_items_count(player, SPIDER_ICHOR) >= 20 and q:get_quest_items_count(player, HARPY_DOWN) >= 20 then
        st:set_cond(10, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 20145 then
    if st:is_cond(9) and q:get_quest_items_count(player, HARPY_DOWN) < 20 and rnd.bool() then
      q:give_items(player, HARPY_DOWN, 1)
      if q:get_quest_items_count(player, HARPY_DOWN) >= 20 and q:get_quest_items_count(player, SPIDER_ICHOR) >= 20 then
        st:set_cond(10, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  elseif sw == 27077 then
    if st:is_cond(18) and q:get_item_equipped(player, Paperdoll.RHAND) == TALINS_SPEAR then
      st:set_cond(19, true)
      q:take_items(player, GRAIL_OF_PURITY, 1)
      q:take_items(player, TALINS_SPEAR, 1)
      q:give_items(player, TEARS_OF_UNICORN, 1)
    end
  elseif sw == 20581 or sw == 20582 then
    if st:is_cond(15) and rnd.bool() then
      for _, itemId in ipairs(TALINS_PIECES) do
        if not q:has_quest_items(player, itemId) then
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
          q:give_items(player, itemId, 1)
          return
        end
      end
      st:set_cond(16, true)
    end
  end
end)
