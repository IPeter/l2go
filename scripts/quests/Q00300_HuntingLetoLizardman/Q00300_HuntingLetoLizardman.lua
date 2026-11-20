-- Translated by questport from quests/Q00300_HuntingLetoLizardman (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(300, "Q00300_HuntingLetoLizardman", "Hunting Leto Lizardman")

-- Monsters
local LETO_LIZARDMAN = 20577
local LETO_LIZARDMAN_ARCHER = 20578
local LETO_LIZARDMAN_SOLDIER = 20579
local LETO_LIZARDMAN_WARRIOR = 20580
local LETO_LIZARDMAN_OVERLORD = 20582
-- Item
local BRACELET = 7139
-- Drop chances
local CHANCES = {}

CHANCES[LETO_LIZARDMAN] = 300000
CHANCES[LETO_LIZARDMAN_ARCHER] = 320000
CHANCES[LETO_LIZARDMAN_SOLDIER] = 350000
CHANCES[LETO_LIZARDMAN_WARRIOR] = 650000
CHANCES[LETO_LIZARDMAN_OVERLORD] = 700000

q:register_quest_items(BRACELET)
q:add_start_npc(30126) -- Rath
q:add_talk_id(30126)
q:add_kill_id(LETO_LIZARDMAN, LETO_LIZARDMAN_ARCHER, LETO_LIZARDMAN_SOLDIER, LETO_LIZARDMAN_WARRIOR, LETO_LIZARDMAN_OVERLORD)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30126-03.htm" then
    st:start_quest()
  elseif event == "30126-05.htm" and q:get_quest_items_count(player, BRACELET) >= 60 then
    htmltext = "30126-06.htm"
    q:take_items(player, BRACELET, -1)
    local luck = rnd(3)
    if luck == 0 then
      q:give_adena(player, 30000, true)
    elseif luck == 1 then
      q:reward_items(player, 1867, 50)
    elseif luck == 2 then
      q:reward_items(player, 1872, 50)
    end
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 34 and "30126-01.htm" or "30126-02.htm"
  elseif sw == State.STARTED then
    htmltext = st:is_cond(1) and "30126-04a.htm" or "30126-04.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, 1, 3, npc)
  if st == nil then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    local partyMember = st:get_player()
    q:give_items(partyMember, BRACELET, 1)
    if q:get_quest_items_count(partyMember, BRACELET) < 60 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
