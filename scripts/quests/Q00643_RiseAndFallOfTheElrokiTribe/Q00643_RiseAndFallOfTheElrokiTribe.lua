-- Translated by questport from quests/Q00643_RiseAndFallOfTheElrokiTribe (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(643, "Q00643_RiseAndFallOfTheElrokiTribe", "Rise and Fall of the Elroki Tribe")

-- NPCs
local SINGSING = 32106
local KARAKAWEI = 32117
-- Items
local BONES = 8776

q:register_quest_items(BONES)
q:add_start_npc(SINGSING)
q:add_talk_id(SINGSING, KARAKAWEI)
q:add_kill_id(22208, 22209, 22210, 22211, 22212, 22213, 22221, 22222, 22226, 22227)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32106-03.htm" then
    st:start_quest()
  elseif event == "32106-07.htm" then
    local count = q:get_quest_items_count(player, BONES)
    q:take_items(player, BONES, count)
    q:give_adena(player, count * 1374, true)
  elseif event == "32106-09.htm" then
    st:exit_quest(true, true)
  elseif event == "32117-03.htm" then
    local count = q:get_quest_items_count(player, BONES)
    if count >= 300 then
      q:take_items(player, BONES, 300)
      q:reward_items(player, rnd(8712, 8722), 5)
    else
      htmltext = "32117-04.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 75 and "32106-00.htm" or "32106-01.htm"
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == SINGSING then
      htmltext = q:has_quest_items(player, BONES) and "32106-06.htm" or "32106-05.htm"
    elseif sw2 == KARAKAWEI then
      htmltext = "32117-01.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil or not st:is_started() then
    return
  end
  if rnd(100) < 75 then
    local partyMember = st:get_player()
    q:give_items(partyMember, BONES, 1)
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
