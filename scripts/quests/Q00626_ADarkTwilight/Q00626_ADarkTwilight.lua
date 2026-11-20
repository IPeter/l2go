-- Translated by questport from quests/Q00626_ADarkTwilight (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(626, "Q00626_ADarkTwilight", "A Dark Twilight")

-- NPC
local HIERARCH = 31517
-- Items
local BLOOD_OF_SAINT = 7169
-- Drop chances
local CHANCES = {}

CHANCES[21520] = 533000
CHANCES[21523] = 566000
CHANCES[21524] = 603000
CHANCES[21525] = 603000
CHANCES[21526] = 587000
CHANCES[21529] = 606000
CHANCES[21530] = 560000
CHANCES[21531] = 669000
CHANCES[21532] = 651000
CHANCES[21535] = 672000
CHANCES[21536] = 597000
CHANCES[21539] = 739000
CHANCES[21540] = 739000
CHANCES[21658] = 669000

q:register_quest_items(BLOOD_OF_SAINT)
q:add_start_npc(HIERARCH)
q:add_talk_id(HIERARCH)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31517-03.htm" then
    st:start_quest()
  elseif event == "reward1" then
    if q:get_quest_items_count(player, BLOOD_OF_SAINT) == 300 then
      htmltext = "31517-07.htm"
      q:take_items(player, BLOOD_OF_SAINT, 300)
      q:add_exp_and_sp(player, 162773, 12500)
      st:exit_quest(false, true)
    else
      htmltext = "31517-08.htm"
    end
  elseif event == "reward2" then
    if q:get_quest_items_count(player, BLOOD_OF_SAINT) == 300 then
      htmltext = "31517-07.htm"
      q:take_items(player, BLOOD_OF_SAINT, 300)
      q:give_adena(player, 100000, true)
      st:exit_quest(false, true)
    else
      htmltext = "31517-08.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 60 and "31517-02.htm" or "31517-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "31517-05.htm"
    else
      htmltext = "31517-04.htm"
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
  if rnd(1000000) < CHANCES[npc:id()] then
    q:give_items(player, BLOOD_OF_SAINT, 1)
    if q:get_quest_items_count(player, BLOOD_OF_SAINT) < 300 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
