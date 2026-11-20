-- Translated by questport from quests/Q00618_IntoTheFlame (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(618, "Q00618_IntoTheFlame", "Into the Flame")

-- NPCs
local KLEIN = 31540
local HILDA = 31271
-- Items
local VACUALITE_ORE = 7265
local VACUALITE = 7266
-- Reward
local FLOATING_STONE = 7267

q:register_quest_items(VACUALITE_ORE, VACUALITE)
q:add_start_npc(KLEIN)
q:add_talk_id(KLEIN, HILDA)
-- Kookaburras, Bandersnatches, Grendels
q:add_kill_id(21274, 21275, 21276, 21277, 21282, 21283, 21284, 21285, 21290, 21291, 21292, 21293)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31540-03.htm" then
    st:start_quest()
  elseif event == "31540-05.htm" then
    q:take_items(player, VACUALITE, 1)
    q:give_items(player, FLOATING_STONE, 1)
    st:exit_quest(true, true)
  elseif event == "31271-02.htm" then
    st:set_cond(2, true)
  elseif event == "31271-05.htm" then
    st:set_cond(4, true)
    q:take_items(player, VACUALITE_ORE, -1)
    q:give_items(player, VACUALITE, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 60 and "31540-01.htm" or "31540-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == KLEIN then
      htmltext = cond == 4 and "31540-04.htm" or "31540-03.htm"
    elseif sw2 == HILDA then
      if cond == 1 then
        htmltext = "31271-01.htm"
      elseif cond == 2 then
        htmltext = "31271-03.htm"
      elseif cond == 3 then
        htmltext = "31271-04.htm"
      elseif cond == 4 then
        htmltext = "31271-06.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 2, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  if rnd.bool() then
    q:give_items(partyMember, VACUALITE_ORE, 1)
    if q:get_quest_items_count(partyMember, VACUALITE_ORE) < 50 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(3, true)
    end
  end
end)
