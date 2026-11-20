-- Translated by questport from quests/Q00406_PathOfTheElvenKnight (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(406, "Q00406_PathOfTheElvenKnight", "Path to an Elven Knight")

-- NPCs
local SORIUS = 30327
local KLUTO = 30317
-- Items
local SORIUS_LETTER = 1202
local KLUTO_BOX = 1203
local ELVEN_KNIGHT_BROOCH = 1204
local TOPAZ_PIECE = 1205
local EMERALD_PIECE = 1206
local KLUTO_MEMO = 1276

q:register_quest_items(SORIUS_LETTER, KLUTO_BOX, TOPAZ_PIECE, EMERALD_PIECE, KLUTO_MEMO)
q:add_start_npc(SORIUS)
q:add_talk_id(SORIUS, KLUTO)
q:add_kill_id(20035, 20042, 20045, 20051, 20054, 20060, 20782)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30327-05.htm" then
    if player:class_id() ~= ClassId.ELVEN_FIGHTER then
      htmltext = player:class_id() == ClassId.ELVEN_KNIGHT and "30327-02a.htm" or "30327-02.htm"
    elseif player:level() < 19 then
      htmltext = "30327-03.htm"
    elseif q:has_quest_items(player, ELVEN_KNIGHT_BROOCH) then
      htmltext = "30327-04.htm"
    end
  elseif event == "30327-06.htm" then
    st:start_quest()
  elseif event == "30317-02.htm" then
    st:set_cond(4, true)
    q:take_items(player, SORIUS_LETTER, 1)
    q:give_items(player, KLUTO_MEMO, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30327-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == SORIUS then
      if cond == 1 then
        htmltext = not q:has_quest_items(player, TOPAZ_PIECE) and "30327-07.htm" or "30327-08.htm"
      elseif cond == 2 then
        htmltext = "30327-09.htm"
        st:set_cond(3, true)
        q:give_items(player, SORIUS_LETTER, 1)
      elseif cond > 2 and cond < 6 then
        htmltext = "30327-11.htm"
      elseif cond == 6 then
        htmltext = "30327-10.htm"
        q:take_items(player, KLUTO_BOX, 1)
        q:take_items(player, KLUTO_MEMO, 1)
        q:give_items(player, ELVEN_KNIGHT_BROOCH, 1)
        q:add_exp_and_sp(player, 3200, 2280)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == KLUTO then
      if cond == 3 then
        htmltext = "30317-01.htm"
      elseif cond == 4 then
        htmltext = not q:has_quest_items(player, EMERALD_PIECE) and "30317-03.htm" or "30317-04.htm"
      elseif cond == 5 then
        htmltext = "30317-05.htm"
        st:set_cond(6, true)
        q:take_items(player, EMERALD_PIECE, -1)
        q:take_items(player, TOPAZ_PIECE, -1)
        q:give_items(player, KLUTO_BOX, 1)
      elseif cond == 6 then
        htmltext = "30317-06.htm"
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
  if sw == 20035 or sw == 20042 or sw == 20045 or sw == 20051 or sw == 20054 or sw == 20060 then
    if st:is_cond(1) and rnd(10) < 7 then
      if q:get_quest_items_count(player, TOPAZ_PIECE) < 20 then
        q:give_items(player, TOPAZ_PIECE, 1)
        if q:get_quest_items_count(player, TOPAZ_PIECE) >= 20 then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == 20782 then
    if st:is_cond(4) and rnd(10) < 5 then
      if q:get_quest_items_count(player, EMERALD_PIECE) < 20 then
        q:give_items(player, EMERALD_PIECE, 1)
        if q:get_quest_items_count(player, EMERALD_PIECE) >= 20 then
          st:set_cond(5, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)
