-- Translated by questport from quests/Q00368_TrespassingIntoTheHolyGround (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(368, "Q00368_TrespassingIntoTheHolyGround", "Trespassing into the Sacred Area")

-- NPC
local RESTINA = 30926
-- Item
local FANG = 5881
-- Drop chances
local CHANCES = {}

CHANCES[20794] = 500000
CHANCES[20795] = 770000
CHANCES[20796] = 500000
CHANCES[20797] = 480000

q:register_quest_items(FANG)
q:add_start_npc(RESTINA)
q:add_talk_id(RESTINA)
q:add_kill_id(20794, 20795, 20796, 20797)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30926-02.htm" then
    st:start_quest()
  elseif event == "30926-05.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 36 and "30926-01a.htm" or "30926-01.htm"
  elseif sw == State.STARTED then
    local fangs = q:get_quest_items_count(player, FANG)
    if fangs == 0 then
      htmltext = "30926-03.htm"
    else
      local reward = 250 * fangs + (fangs > 10 and 5730 or 2000)
      htmltext = "30926-04.htm"
      q:take_items(player, 5881, -1)
      q:give_adena(player, reward, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    q:give_items(st:get_player(), FANG, 1)
    q:play_sound(st:get_player(), QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
