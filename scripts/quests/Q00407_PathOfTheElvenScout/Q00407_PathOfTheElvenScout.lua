-- Translated by questport from quests/Q00407_PathOfTheElvenScout (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(407, "Q00407_PathOfTheElvenScout", "Path to an Elven Scout")

-- NPCs
local REISA = 30328
local BABENCO = 30334
local MORETTI = 30337
local PRIAS = 30426
-- Items
local REISA_LETTER = 1207
local PRIAS_TORN_LETTER_1 = 1208
local PRIAS_TORN_LETTER_2 = 1209
local PRIAS_TORN_LETTER_3 = 1210
local PRIAS_TORN_LETTER_4 = 1211
local MORETTI_HERB = 1212
local MORETTI_LETTER = 1214
local PRIAS_LETTER = 1215
local HONORARY_GUARD = 1216
local REISA_RECOMMENDATION = 1217
local RUSTED_KEY = 1293

q:register_quest_items(REISA_LETTER, PRIAS_TORN_LETTER_1, PRIAS_TORN_LETTER_2, PRIAS_TORN_LETTER_3, PRIAS_TORN_LETTER_4, MORETTI_HERB, MORETTI_LETTER, PRIAS_LETTER, HONORARY_GUARD, RUSTED_KEY)
q:add_start_npc(REISA)
q:add_talk_id(REISA, MORETTI, BABENCO, PRIAS)
q:add_kill_id(20053, 27031)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30328-05.htm" then
    if player:class_id() ~= ClassId.ELVEN_FIGHTER then
      htmltext = player:class_id() == ClassId.ELVEN_SCOUT and "30328-02a.htm" or "30328-02.htm"
    elseif player:level() < 19 then
      htmltext = "30328-03.htm"
    elseif q:has_quest_items(player, REISA_RECOMMENDATION) then
      htmltext = "30328-04.htm"
    else
      st:start_quest()
      q:give_items(player, REISA_LETTER, 1)
    end
  elseif event == "30337-03.htm" then
    st:set_cond(2, true)
    q:take_items(player, REISA_LETTER, -1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = "30328-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == REISA then
      if cond == 1 then
        htmltext = "30328-06.htm"
      elseif cond > 1 and cond < 8 then
        htmltext = "30328-08.htm"
      elseif cond == 8 then
        htmltext = "30328-07.htm"
        q:take_items(player, HONORARY_GUARD, -1)
        q:give_items(player, REISA_RECOMMENDATION, 1)
        q:add_exp_and_sp(player, 3200, 1000)
        player:social_action(3)
        st:exit_quest(true, true)
      end
    elseif sw2 == MORETTI then
      if cond == 1 then
        htmltext = "30337-01.htm"
      elseif cond == 2 then
        htmltext = not q:has_quest_items(player, PRIAS_TORN_LETTER_1) and "30337-04.htm" or "30337-05.htm"
      elseif cond == 3 then
        htmltext = "30337-06.htm"
        st:set_cond(4, true)
        q:take_items(player, PRIAS_TORN_LETTER_1, -1)
        q:take_items(player, PRIAS_TORN_LETTER_2, -1)
        q:take_items(player, PRIAS_TORN_LETTER_3, -1)
        q:take_items(player, PRIAS_TORN_LETTER_4, -1)
        q:give_items(player, MORETTI_HERB, 1)
        q:give_items(player, MORETTI_LETTER, 1)
      elseif cond > 3 and cond < 7 then
        htmltext = "30337-09.htm"
      elseif cond == 7 and q:has_quest_items(player, PRIAS_LETTER) then
        htmltext = "30337-07.htm"
        st:set_cond(8, true)
        q:take_items(player, PRIAS_LETTER, -1)
        q:give_items(player, HONORARY_GUARD, 1)
      elseif cond == 8 then
        htmltext = "30337-08.htm"
      end
    elseif sw2 == BABENCO then
      if cond == 2 then
        htmltext = "30334-01.htm"
      end
    elseif sw2 == PRIAS then
      if cond == 4 then
        htmltext = "30426-01.htm"
        st:set_cond(5, true)
      elseif cond == 5 then
        htmltext = "30426-01.htm"
      elseif cond == 6 then
        htmltext = "30426-02.htm"
        st:set_cond(7, true)
        q:take_items(player, RUSTED_KEY, -1)
        q:take_items(player, MORETTI_HERB, -1)
        q:take_items(player, MORETTI_LETTER, -1)
        q:give_items(player, PRIAS_LETTER, 1)
      elseif cond == 7 then
        htmltext = "30426-04.htm"
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
  local cond = st:get_cond()
  if npc:id() == 20053 then
    if cond == 2 then
      if not q:has_quest_items(player, PRIAS_TORN_LETTER_1) then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, PRIAS_TORN_LETTER_1, 1)
      elseif not q:has_quest_items(player, PRIAS_TORN_LETTER_2) then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, PRIAS_TORN_LETTER_2, 1)
      elseif not q:has_quest_items(player, PRIAS_TORN_LETTER_3) then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        q:give_items(player, PRIAS_TORN_LETTER_3, 1)
      elseif not q:has_quest_items(player, PRIAS_TORN_LETTER_4) then
        st:set_cond(3, true)
        q:give_items(player, PRIAS_TORN_LETTER_4, 1)
      end
    end
  elseif (cond == 4 or cond == 5) and rnd(10) < 6 then
    q:give_items(player, RUSTED_KEY, 1)
    st:set_cond(6, true)
  end
end)
