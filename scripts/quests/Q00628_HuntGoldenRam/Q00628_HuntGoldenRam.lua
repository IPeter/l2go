-- Translated by questport from quests/Q00628_HuntGoldenRam (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(628, "Q00628_HuntGoldenRam", "Hunt of the Golden Ram Mercenary Force")

-- NPCs
local KAHMAN = 31554
-- Items
local SPLINTER_STAKATO_CHITIN = 7248
local NEEDLE_STAKATO_CHITIN = 7249
local GOLDEN_RAM_BADGE_RECRUIT = 7246
local GOLDEN_RAM_BADGE_SOLDIER = 7247
-- Drop chances
local CHANCES = {}

CHANCES[21508] = 500000
CHANCES[21509] = 430000
CHANCES[21510] = 521000
CHANCES[21511] = 575000
CHANCES[21512] = 746000
CHANCES[21513] = 500000
CHANCES[21514] = 430000
CHANCES[21515] = 520000
CHANCES[21516] = 531000
CHANCES[21517] = 744000

q:register_quest_items(SPLINTER_STAKATO_CHITIN, NEEDLE_STAKATO_CHITIN, GOLDEN_RAM_BADGE_RECRUIT, GOLDEN_RAM_BADGE_SOLDIER)
q:add_start_npc(KAHMAN)
q:add_talk_id(KAHMAN)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31554-02.htm" then
    st:start_quest()
  elseif event == "31554-03a.htm" then
    if q:get_quest_items_count(player, SPLINTER_STAKATO_CHITIN) >= 100 and st:is_cond(1) then
      htmltext = "31554-04.htm"
      st:set_cond(2, true)
      q:take_items(player, SPLINTER_STAKATO_CHITIN, -1)
      q:give_items(player, GOLDEN_RAM_BADGE_RECRUIT, 1)
    end
  elseif event == "31554-07.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 66 and "31554-01a.htm" or "31554-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      if q:get_quest_items_count(player, SPLINTER_STAKATO_CHITIN) >= 100 then
        htmltext = "31554-03.htm"
      else
        htmltext = "31554-03a.htm"
      end
    elseif cond == 2 then
      if q:get_quest_items_count(player, SPLINTER_STAKATO_CHITIN) >= 100 and q:get_quest_items_count(player, NEEDLE_STAKATO_CHITIN) >= 100 then
        htmltext = "31554-05.htm"
        st:set_cond(3)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_FINISH)
        q:take_items(player, SPLINTER_STAKATO_CHITIN, -1)
        q:take_items(player, NEEDLE_STAKATO_CHITIN, -1)
        q:take_items(player, GOLDEN_RAM_BADGE_RECRUIT, 1)
        q:give_items(player, GOLDEN_RAM_BADGE_SOLDIER, 1)
      elseif not q:has_quest_items(player, SPLINTER_STAKATO_CHITIN) and not q:has_quest_items(player, NEEDLE_STAKATO_CHITIN) then
        htmltext = "31554-04b.htm"
      else
        htmltext = "31554-04a.htm"
      end
    elseif cond == 3 then
      htmltext = "31554-05a.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil or not st:is_started() then
    return
  end
  local partyMember = st:get_player()
  local cond = st:get_cond()
  local npcId = npc:id()
  if npcId == 21508 or npcId == 21509 or npcId == 21510 or npcId == 21511 or npcId == 21512 then
    if (cond == 1 or cond == 2) and q:get_quest_items_count(partyMember, SPLINTER_STAKATO_CHITIN) < 100 and rnd(1000000) < CHANCES[npcId] then
      q:give_items(partyMember, SPLINTER_STAKATO_CHITIN, 1)
      if q:get_quest_items_count(partyMember, SPLINTER_STAKATO_CHITIN) < 100 then
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  elseif npcId == 21513 or npcId == 21514 or npcId == 21515 or npcId == 21516 or npcId == 21517 then
    if cond == 2 and q:get_quest_items_count(partyMember, NEEDLE_STAKATO_CHITIN) < 100 and rnd(1000000) < CHANCES[npcId] then
      q:give_items(partyMember, NEEDLE_STAKATO_CHITIN, 1)
      if q:get_quest_items_count(partyMember, NEEDLE_STAKATO_CHITIN) < 100 then
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      end
    end
  end
end)
