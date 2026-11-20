-- Translated by questport from quests/Q00612_BattleAgainstKetraOrcs (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Battle against Ketra Orcs (612)
-- @author malyelfik
local J = require("lib.java")

local q = quest.new(612, "Q00612_BattleAgainstKetraOrcs", "War with Ketra Orcs")

-- NPC
local ASHAS = 31377
-- Monsters
local MOBS = {}
-- Items
local SEED = 7187
local MOLAR = 7234
-- Misc
local MIN_LEVEL = 74
local MOLAR_COUNT = 100

MOBS[21324] = 500 -- Ketra Orc Footman
MOBS[21327] = 510 -- Ketra Orc Raider
MOBS[21328] = 522 -- Ketra Orc Scout
MOBS[21329] = 519 -- Ketra Orc Shaman
MOBS[21331] = 529 -- Ketra Orc Warrior
MOBS[21332] = 529 -- Ketra Orc Lieutenant
MOBS[21334] = 539 -- Ketra Orc Medium
MOBS[21336] = 548 -- Ketra Orc White Captain
MOBS[21338] = 558 -- Ketra Orc Seer
MOBS[21339] = 568 -- Ketra Orc General
MOBS[21340] = 568 -- Ketra Orc Battalion Commander
MOBS[21342] = 578 -- Ketra Orc Grand Seer
MOBS[21343] = 664 -- Ketra Commander
MOBS[21345] = 713 -- Ketra's Head Shaman
MOBS[21347] = 738 -- Ketra Prophet

q:add_start_npc(ASHAS)
q:add_talk_id(ASHAS)
q:add_kill_id(J.keys(MOBS))
q:register_quest_items(MOLAR)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = event
  if event == "31377-03.htm" then
    qs:start_quest()
  elseif event == "31377-06.html" then
  elseif event == "31377-07.html" then
    if q:get_quest_items_count(player, MOLAR) < MOLAR_COUNT then
      return "31377-08.html"
    end
    q:take_items(player, MOLAR, MOLAR_COUNT)
    q:give_items(player, SEED, 20)
  elseif event == "31377-09.html" then
    qs:exit_quest(true, true)
  else
    htmltext = nil
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local member = q:get_random_party_member(killer, 1)
  if member ~= nil and rnd(1000) < MOBS[npc:id()] then
    q:give_items(member, MOLAR, 1)
    q:play_sound(member, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = qs:get_state()
  if sw == State.CREATED then
    htmltext = player:level() >= MIN_LEVEL and "31377-01.htm" or "31377-02.htm"
  elseif sw == State.STARTED then
    htmltext = q:has_quest_items(player, MOLAR) and "31377-04.html" or "31377-05.html"
  end
  return htmltext
end)
