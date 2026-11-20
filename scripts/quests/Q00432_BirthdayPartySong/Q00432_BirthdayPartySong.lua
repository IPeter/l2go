-- Translated by questport from quests/Q00432_BirthdayPartySong (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(432, "Q00432_BirthdayPartySong", "Birthday Party Song")

-- NPC
local OCTAVIA = 31043
-- Item
local RED_CRYSTAL = 7541

q:register_quest_items(RED_CRYSTAL)
q:add_start_npc(OCTAVIA)
q:add_talk_id(OCTAVIA)
q:add_kill_id(21103)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31043-02.htm" then
    st:start_quest()
  elseif event == "31043-06.htm" and q:get_quest_items_count(player, RED_CRYSTAL) == 50 then
    htmltext = "31043-05.htm"
    q:take_items(player, RED_CRYSTAL, -1)
    q:reward_items(player, 7061, 25)
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 31 and "31043-00.htm" or "31043-01.htm"
  elseif sw == State.STARTED then
    htmltext = q:get_quest_items_count(player, RED_CRYSTAL) < 50 and "31043-03.htm" or "31043-04.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 1, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  if rnd.bool() then
    q:give_items(partyMember, RED_CRYSTAL, 1)
    if q:get_quest_items_count(partyMember, RED_CRYSTAL) < 50 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
