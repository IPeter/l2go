-- Translated by questport from quests/Q00171_ActsOfEvil (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(171, "Q00171_ActsOfEvil", "Acts of Evil")

-- NPCs
local ALVAH = 30381
local ARODIN = 30207
local TYRA = 30420
local ROLENTO = 30437
local NETI = 30425
local BURAI = 30617
-- Items
local BLADE_MOLD = 4239
local TYRA_BILL = 4240
local RANGER_REPORT_1 = 4241
local RANGER_REPORT_2 = 4242
local RANGER_REPORT_3 = 4243
local RANGER_REPORT_4 = 4244
local WEAPON_TRADE_CONTRACT = 4245
local ATTACK_DIRECTIVES = 4246
local CERTIFICATE = 4247
local CARGO_BOX = 4248
local OL_MAHUM_HEAD = 4249
-- Turek Orcs drop chances
local CHANCES = {}

CHANCES[20496] = 530000
CHANCES[20497] = 550000
CHANCES[20498] = 510000
CHANCES[20499] = 500000

q:register_quest_items(BLADE_MOLD, TYRA_BILL, RANGER_REPORT_1, RANGER_REPORT_2, RANGER_REPORT_3, RANGER_REPORT_4, WEAPON_TRADE_CONTRACT, ATTACK_DIRECTIVES, CERTIFICATE, CARGO_BOX, OL_MAHUM_HEAD)
q:add_start_npc(ALVAH)
q:add_talk_id(ALVAH, ARODIN, TYRA, ROLENTO, NETI, BURAI)
q:add_kill_id(20496, 20497, 20498, 20499, 20062, 20064, 20066, 20438)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30381-02.htm" then
    st:start_quest()
  elseif event == "30207-02.htm" then
    st:set_cond(2, true)
  elseif event == "30381-04.htm" then
    st:set_cond(5, true)
  elseif event == "30381-07.htm" then
    st:set_cond(7, true)
    q:take_items(player, WEAPON_TRADE_CONTRACT, 1)
  elseif event == "30437-03.htm" then
    st:set_cond(9, true)
    q:give_items(player, CARGO_BOX, 1)
    q:give_items(player, CERTIFICATE, 1)
  elseif event == "30617-04.htm" then
    st:set_cond(10, true)
    q:take_items(player, ATTACK_DIRECTIVES, 1)
    q:take_items(player, CARGO_BOX, 1)
    q:take_items(player, CERTIFICATE, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 27 and "30381-01a.htm" or "30381-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ALVAH then
      if cond < 4 then
        htmltext = "30381-02a.htm"
      elseif cond == 4 then
        htmltext = "30381-03.htm"
      elseif cond == 5 then
        if q:has_quest_items(player, RANGER_REPORT_1, RANGER_REPORT_2, RANGER_REPORT_3, RANGER_REPORT_4) then
          htmltext = "30381-05.htm"
          st:set_cond(6, true)
          q:take_items(player, RANGER_REPORT_1, 1)
          q:take_items(player, RANGER_REPORT_2, 1)
          q:take_items(player, RANGER_REPORT_3, 1)
          q:take_items(player, RANGER_REPORT_4, 1)
        else
          htmltext = "30381-04a.htm"
        end
      elseif cond == 6 then
        if q:has_quest_items(player, WEAPON_TRADE_CONTRACT, ATTACK_DIRECTIVES) then
          htmltext = "30381-06.htm"
        else
          htmltext = "30381-05a.htm"
        end
      elseif cond > 6 and cond < 11 then
        htmltext = "30381-07a.htm"
      elseif cond == 11 then
        htmltext = "30381-08.htm"
        q:give_adena(player, 90000, true)
        st:exit_quest(false, true)
      end
    elseif sw2 == ARODIN then
      if cond == 1 then
        htmltext = "30207-01.htm"
      elseif cond == 2 then
        htmltext = "30207-01a.htm"
      elseif cond == 3 then
        if q:has_quest_items(player, TYRA_BILL) then
          htmltext = "30207-03.htm"
          st:set_cond(4, true)
          q:take_items(player, TYRA_BILL, 1)
        else
          htmltext = "30207-01a.htm"
        end
      elseif cond > 3 then
        htmltext = "30207-03a.htm"
      end
    elseif sw2 == TYRA then
      if cond == 2 then
        if q:get_quest_items_count(player, BLADE_MOLD) >= 20 then
          htmltext = "30420-01.htm"
          st:set_cond(3, true)
          q:take_items(player, BLADE_MOLD, -1)
          q:give_items(player, TYRA_BILL, 1)
        else
          htmltext = "30420-01b.htm"
        end
      elseif cond == 3 then
        htmltext = "30420-01a.htm"
      elseif cond > 3 then
        htmltext = "30420-02.htm"
      end
    elseif sw2 == NETI then
      if cond == 7 then
        htmltext = "30425-01.htm"
        st:set_cond(8, true)
      elseif cond > 7 then
        htmltext = "30425-02.htm"
      end
    elseif sw2 == ROLENTO then
      if cond == 8 then
        htmltext = "30437-01.htm"
      elseif cond > 8 then
        htmltext = "30437-03a.htm"
      end
    elseif sw2 == BURAI then
      if cond == 9 and q:has_quest_items(player, CERTIFICATE, CARGO_BOX, ATTACK_DIRECTIVES) then
        htmltext = "30617-01.htm"
      elseif cond == 10 then
        if q:get_quest_items_count(player, OL_MAHUM_HEAD) >= 30 then
          htmltext = "30617-05.htm"
          st:set_cond(11, true)
          q:take_items(player, OL_MAHUM_HEAD, -1)
          q:give_adena(player, 8000, true)
        else
          htmltext = "30617-04a.htm"
        end
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
  local npcId = npc:id()
  if npcId == 20496 or npcId == 20497 or npcId == 20498 or npcId == 20499 then
    if st:is_cond(2) and rnd(1000000) < CHANCES[npcId] then
      local count = q:get_quest_items_count(player, BLADE_MOLD)
      if count < 20 then
        q:give_items(player, BLADE_MOLD, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
      if count == 5 or count >= 10 and rnd(100) < 25 then
        q:add_spawn(27190, {x = player:x(), y = player:y(), z = player:z(), heading = player:heading()}, false, 0)
      end
    end
  elseif npcId == 20062 or npcId == 20064 then
    if st:is_cond(5) then
      if not q:has_quest_items(player, RANGER_REPORT_1) then
        q:give_items(player, RANGER_REPORT_1, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      elseif rnd(100) < 20 then
        if not q:has_quest_items(player, RANGER_REPORT_2) then
          q:give_items(player, RANGER_REPORT_2, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        elseif not q:has_quest_items(player, RANGER_REPORT_3) then
          q:give_items(player, RANGER_REPORT_3, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        elseif not q:has_quest_items(player, RANGER_REPORT_4) then
          q:give_items(player, RANGER_REPORT_4, 1)
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif npcId == 20438 then
    if st:is_cond(6) and rnd(100) < 10 and not q:has_quest_items(player, WEAPON_TRADE_CONTRACT, ATTACK_DIRECTIVES) then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      q:give_items(player, WEAPON_TRADE_CONTRACT, 1)
      q:give_items(player, ATTACK_DIRECTIVES, 1)
    end
  elseif npcId == 20066 then
    if st:is_cond(10) and q:get_quest_items_count(player, OL_MAHUM_HEAD) < 30 and rnd.bool() then
      q:give_items(player, OL_MAHUM_HEAD, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
