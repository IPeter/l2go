-- Translated by questport from quests/Q00629_CleanUpTheSwampOfScreams (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(629, "Q00629_CleanUpTheSwampOfScreams", "Clean up the Swamp of Screams")

-- NPC
local PIERCE = 31553
-- Items
local TALON_OF_STAKATO = 7250
local GOLDEN_RAM_COIN = 7251
-- Drop chances
local CHANCES = {}

CHANCES[21508] = 500000
CHANCES[21509] = 431000
CHANCES[21510] = 521000
CHANCES[21511] = 576000
CHANCES[21512] = 746000
CHANCES[21513] = 530000
CHANCES[21514] = 538000
CHANCES[21515] = 545000
CHANCES[21516] = 553000
CHANCES[21517] = 560000

q:register_quest_items(TALON_OF_STAKATO, GOLDEN_RAM_COIN)
q:add_start_npc(PIERCE)
q:add_talk_id(PIERCE)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31553-1.htm" then
    if player:level() >= 66 then
      st:start_quest()
    else
      htmltext = "31553-0a.htm"
      st:exit_quest(true)
    end
  elseif event == "31553-3.htm" then
    if q:get_quest_items_count(player, TALON_OF_STAKATO) >= 100 then
      q:take_items(player, TALON_OF_STAKATO, 100)
      q:give_items(player, GOLDEN_RAM_COIN, 20)
    else
      htmltext = "31553-3a.htm"
    end
  elseif event == "31553-5.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  -- Only members of the Golden Ram mercenaries (Q628's badges).
  if not q:has_at_least_one_quest_item(player, 7246, 7247) then
    return "31553-6.htm"
  end
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 66 and "31553-0a.htm" or "31553-0.htm"
  elseif sw == State.STARTED then
    htmltext = q:get_quest_items_count(player, TALON_OF_STAKATO) >= 100 and "31553-2.htm" or "31553-1a.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil or not st:is_started() then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    local partyMember = st:get_player()
    q:give_items(partyMember, TALON_OF_STAKATO, 1)
    if q:get_quest_items_count(partyMember, TALON_OF_STAKATO) < 100 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
