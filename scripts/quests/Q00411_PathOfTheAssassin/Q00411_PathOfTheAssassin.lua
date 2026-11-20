-- Translated by questport from quests/Q00411_PathOfTheAssassin (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(411, "Q00411_PathOfTheAssassin", "Path to an Assassin")

-- NPCs
local TRISKEL = 30416
local ARKENIA = 30419
local LEIKAN = 30382
-- Items
local SHILEN_CALL = 1245
local ARKENIA_LETTER = 1246
local LEIKAN_NOTE = 1247
local MOONSTONE_BEAST_MOLAR = 1248
local SHILEN_TEARS = 1250
local ARKENIA_RECOMMENDATION = 1251
local IRON_HEART = 1252

q:register_quest_items(SHILEN_CALL, ARKENIA_LETTER, LEIKAN_NOTE, MOONSTONE_BEAST_MOLAR, SHILEN_TEARS, ARKENIA_RECOMMENDATION)
q:add_start_npc(TRISKEL)
q:add_talk_id(TRISKEL, ARKENIA, LEIKAN)
q:add_kill_id(27036, 20369)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30416-05.htm" then
    if player:class_id() ~= ClassId.DARK_FIGHTER then
      htmltext = player:class_id() == ClassId.ASSASSIN and "30416-02a.htm" or "30416-02.htm"
    elseif player:level() < 19 then
      htmltext = "30416-03.htm"
    elseif q:has_quest_items(player, IRON_HEART) then
      htmltext = "30416-04.htm"
    else
      st:start_quest()
      q:give_items(player, SHILEN_CALL, 1)
    end
  elseif event == "30419-05.htm" then
    st:set_cond(2, true)
    q:take_items(player, SHILEN_CALL, 1)
    q:give_items(player, ARKENIA_LETTER, 1)
  elseif event == "30382-03.htm" then
    st:set_cond(3, true)
    q:take_items(player, ARKENIA_LETTER, 1)
    q:give_items(player, LEIKAN_NOTE, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30416-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == TRISKEL then
      if cond == 1 then
        htmltext = "30416-11.htm"
      elseif cond == 2 then
        htmltext = "30416-07.htm"
      elseif cond == 3 or cond == 4 then
        htmltext = "30416-08.htm"
      elseif cond == 5 then
        htmltext = "30416-09.htm"
      elseif cond == 6 then
        htmltext = "30416-10.htm"
      elseif cond == 7 then
        htmltext = "30416-06.htm"
        q:take_items(player, ARKENIA_RECOMMENDATION, 1)
        q:give_items(player, IRON_HEART, 1)
        q:add_exp_and_sp(player, 3200, 3930)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == ARKENIA then
      if cond == 1 then
        htmltext = "30419-01.htm"
      elseif cond == 2 then
        htmltext = "30419-07.htm"
      elseif cond == 3 or cond == 4 then
        htmltext = "30419-10.htm"
      elseif cond == 5 then
        htmltext = "30419-11.htm"
      elseif cond == 6 then
        htmltext = "30419-08.htm"
        st:set_cond(7, true)
        q:take_items(player, SHILEN_TEARS, -1)
        q:give_items(player, ARKENIA_RECOMMENDATION, 1)
      elseif cond == 7 then
        htmltext = "30419-09.htm"
      end
    elseif sw2 == LEIKAN then
      if cond == 2 then
        htmltext = "30382-01.htm"
      elseif cond == 3 then
        htmltext = not q:has_quest_items(player, MOONSTONE_BEAST_MOLAR) and "30382-05.htm" or "30382-06.htm"
      elseif cond == 4 then
        htmltext = "30382-07.htm"
        st:set_cond(5, true)
        q:take_items(player, MOONSTONE_BEAST_MOLAR, -1)
        q:take_items(player, LEIKAN_NOTE, -1)
      elseif cond == 5 then
        htmltext = "30382-09.htm"
      elseif cond > 5 then
        htmltext = "30382-08.htm"
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
  if npc:id() == 20369 then
    if st:is_cond(3) then
      if q:get_quest_items_count(player, MOONSTONE_BEAST_MOLAR) < 10 then
        q:give_items(player, MOONSTONE_BEAST_MOLAR, 1)
        if q:get_quest_items_count(player, MOONSTONE_BEAST_MOLAR) >= 10 then
          st:set_cond(4, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif st:is_cond(5) then
    st:set_cond(6, true)
    q:give_items(player, SHILEN_TEARS, 1)
  end
end)
