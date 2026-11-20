-- Translated by questport from quests/Q00413_PathOfTheShillienOracle (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(413, "Q00413_PathOfTheShillienOracle", "Path to a Shillien Oracle")

-- NPCs
local SIDRA = 30330
local ADONIUS = 30375
local TALBOT = 30377
-- Items
local SIDRA_LETTER = 1262
local BLANK_SHEET = 1263
local BLOODY_RUNE = 1264
local GARMIEL_BOOK = 1265
local PRAYER_OF_ADONIUS = 1266
local PENITENT_MARK = 1267
local ASHEN_BONES = 1268
local ANDARIEL_BOOK = 1269
local ORB_OF_ABYSS = 1270

q:register_quest_items(SIDRA_LETTER, BLANK_SHEET, BLOODY_RUNE, GARMIEL_BOOK, PRAYER_OF_ADONIUS, PENITENT_MARK, ASHEN_BONES, ANDARIEL_BOOK)
q:add_start_npc(SIDRA)
q:add_talk_id(SIDRA, ADONIUS, TALBOT)
q:add_kill_id(20776, 20457, 20458, 20514, 20515)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30330-05.htm" then
    if player:class_id() ~= ClassId.DARK_MAGE then
      htmltext = player:class_id() == ClassId.SHILLIEN_ORACLE and "30330-02a.htm" or "30330-03.htm"
    elseif player:level() < 19 then
      htmltext = "30330-02.htm"
    elseif q:has_quest_items(player, ORB_OF_ABYSS) then
      htmltext = "30330-04.htm"
    end
  elseif event == "30330-06.htm" then
    st:start_quest()
    q:give_items(player, SIDRA_LETTER, 1)
  elseif event == "30377-02.htm" then
    st:set_cond(2, true)
    q:take_items(player, SIDRA_LETTER, 1)
    q:give_items(player, BLANK_SHEET, 5)
  elseif event == "30375-04.htm" then
    st:set_cond(5, true)
    q:take_items(player, PRAYER_OF_ADONIUS, 1)
    q:give_items(player, PENITENT_MARK, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30330-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == SIDRA then
      if cond == 1 then
        htmltext = "30330-07.htm"
      elseif cond > 1 and cond < 4 then
        htmltext = "30330-08.htm"
      elseif cond > 3 and cond < 7 then
        htmltext = "30330-09.htm"
      elseif cond == 7 then
        htmltext = "30330-10.htm"
        q:take_items(player, ANDARIEL_BOOK, 1)
        q:take_items(player, GARMIEL_BOOK, 1)
        q:give_items(player, ORB_OF_ABYSS, 1)
        q:add_exp_and_sp(player, 3200, 3120)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == TALBOT then
      if cond == 1 then
        htmltext = "30377-01.htm"
      elseif cond == 2 then
        htmltext = q:has_quest_items(player, BLOODY_RUNE) and "30377-04.htm" or "30377-03.htm"
      elseif cond == 3 then
        htmltext = "30377-05.htm"
        st:set_cond(4, true)
        q:take_items(player, BLOODY_RUNE, -1)
        q:give_items(player, GARMIEL_BOOK, 1)
        q:give_items(player, PRAYER_OF_ADONIUS, 1)
      elseif cond > 3 and cond < 7 then
        htmltext = "30377-06.htm"
      elseif cond == 7 then
        htmltext = "30377-07.htm"
      end
    elseif sw2 == ADONIUS then
      if cond == 4 then
        htmltext = "30375-01.htm"
      elseif cond == 5 then
        htmltext = q:has_quest_items(player, ASHEN_BONES) and "30375-05.htm" or "30375-06.htm"
      elseif cond == 6 then
        htmltext = "30375-07.htm"
        st:set_cond(7, true)
        q:take_items(player, ASHEN_BONES, -1)
        q:take_items(player, PENITENT_MARK, -1)
        q:give_items(player, ANDARIEL_BOOK, 1)
      elseif cond == 7 then
        htmltext = "30375-08.htm"
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
  if npc:id() == 20776 then
    if st:is_cond(2) then
      if q:get_quest_items_count(player, BLOODY_RUNE) < 5 then
        q:take_items(player, BLANK_SHEET, 1)
        q:give_items(player, BLOODY_RUNE, 1)
        if q:get_quest_items_count(player, BLOODY_RUNE) >= 5 then
          st:set_cond(3, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif st:is_cond(5) then
    if q:get_quest_items_count(player, ASHEN_BONES) < 10 then
      q:give_items(player, ASHEN_BONES, 1)
      if q:get_quest_items_count(player, ASHEN_BONES) >= 10 then
        st:set_cond(6, true)
      else
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    end
  end
end)
