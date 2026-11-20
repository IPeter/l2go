-- Translated by questport from quests/Q00606_BattleAgainstVarkaSilenos (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Battle against Varka Silenos (606)
-- @author malyelfik
local J = require("lib.java")

local q = quest.new(606, "Q00606_BattleAgainstVarkaSilenos", "War with Varka Silenos")

-- NPC
local KADUN = 31370
-- Monsters
local MOBS = {}
-- Items
local HORN = 7186
local MANE = 7233
-- Misc
local MIN_LEVEL = 74
local MANE_COUNT = 100

MOBS[21350] = 500 -- Varka Silenos Recruit
MOBS[21353] = 510 -- Varka Silenos Scout
MOBS[21354] = 522 -- Varka Silenos Hunter
MOBS[21355] = 519 -- Varka Silenos Shaman
MOBS[21357] = 529 -- Varka Silenos Priest
MOBS[21358] = 529 -- Varka Silenos Warrior
MOBS[21360] = 539 -- Varka Silenos Medium
MOBS[21362] = 539 -- Varka Silenos Officer
MOBS[21364] = 558 -- Varka Silenos Seer
MOBS[21365] = 568 -- Varka Silenos Great Magus
MOBS[21366] = 568 -- Varka Silenos General
MOBS[21368] = 568 -- Varka Silenos Great Seer
MOBS[21369] = 664 -- Varka's Commander
MOBS[21371] = 713 -- Varka's Head Magus
MOBS[21373] = 738 -- Varka's Prophet

q:add_start_npc(KADUN)
q:add_talk_id(KADUN)
q:add_kill_id(J.keys(MOBS))
q:register_quest_items(MANE)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = event
  if event == "31370-03.htm" then
    qs:start_quest()
  elseif event == "31370-06.html" then
  elseif event == "31370-07.html" then
    if q:get_quest_items_count(player, MANE) < MANE_COUNT then
      return "31370-08.html"
    end
    q:take_items(player, MANE, MANE_COUNT)
    q:give_items(player, HORN, 20)
  elseif event == "31370-09.html" then
    qs:exit_quest(true, true)
  else
    htmltext = nil
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local member = q:get_random_party_member(killer, 1)
  if member ~= nil and rnd(1000) < MOBS[npc:id()] then
    q:give_items(member, MANE, 1)
    q:play_sound(member, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = qs:get_state()
  if sw == State.CREATED then
    htmltext = player:level() >= MIN_LEVEL and "31370-01.htm" or "31370-02.htm"
  elseif sw == State.STARTED then
    htmltext = q:has_quest_items(player, MANE) and "31370-04.html" or "31370-05.html"
  end
  return htmltext
end)
