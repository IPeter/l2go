-- Translated by questport from quests/Q00603_DaimonTheWhiteEyedPart1 (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(603, "Q00603_DaimonTheWhiteEyedPart1", "Daimon the White-Eyed - Part 1")

-- NPCs
local EYE_OF_ARGOS = 31683
local MYSTERIOUS_TABLET_1 = 31548
local MYSTERIOUS_TABLET_2 = 31549
local MYSTERIOUS_TABLET_3 = 31550
local MYSTERIOUS_TABLET_4 = 31551
local MYSTERIOUS_TABLET_5 = 31552
-- Monsters
local CANYON_BANDERSNATCH_SLAVE = 21297
local BUFFALO_SLAVE = 21299
local GRENDEL_SLAVE = 21304
-- Items
local EVIL_SPIRIT_BEADS = 7190
local BROKEN_CRYSTAL = 7191
local UNFINISHED_SUMMON_CRYSTAL = 7192
-- Drop chances
local CHANCES = {}

CHANCES[CANYON_BANDERSNATCH_SLAVE] = 500000
CHANCES[BUFFALO_SLAVE] = 519000
CHANCES[GRENDEL_SLAVE] = 673000

q:register_quest_items(EVIL_SPIRIT_BEADS, BROKEN_CRYSTAL)
q:add_start_npc(EYE_OF_ARGOS)
q:add_talk_id(EYE_OF_ARGOS, MYSTERIOUS_TABLET_1, MYSTERIOUS_TABLET_2, MYSTERIOUS_TABLET_3, MYSTERIOUS_TABLET_4, MYSTERIOUS_TABLET_5)
q:add_kill_id(BUFFALO_SLAVE, GRENDEL_SLAVE, CANYON_BANDERSNATCH_SLAVE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31683-03.htm" then
    st:start_quest()
  elseif event == "31683-06.htm" then
    if q:get_quest_items_count(player, BROKEN_CRYSTAL) > 4 then
      st:set_cond(7, true)
      q:take_items(player, BROKEN_CRYSTAL, -1)
    else
      htmltext = "31683-07.htm"
    end
  elseif event == "31683-10.htm" then
    if q:get_quest_items_count(player, EVIL_SPIRIT_BEADS) > 199 then
      q:take_items(player, EVIL_SPIRIT_BEADS, -1)
      q:give_items(player, UNFINISHED_SUMMON_CRYSTAL, 1)
      st:exit_quest(true, true)
    else
      st:set_cond(7)
      htmltext = "31683-11.htm"
    end
  elseif event == "31548-02.htm" then
    st:set_cond(2, true)
    q:give_items(player, BROKEN_CRYSTAL, 1)
  elseif event == "31549-02.htm" then
    st:set_cond(3, true)
    q:give_items(player, BROKEN_CRYSTAL, 1)
  elseif event == "31550-02.htm" then
    st:set_cond(4, true)
    q:give_items(player, BROKEN_CRYSTAL, 1)
  elseif event == "31551-02.htm" then
    st:set_cond(5, true)
    q:give_items(player, BROKEN_CRYSTAL, 1)
  elseif event == "31552-02.htm" then
    st:set_cond(6, true)
    q:give_items(player, BROKEN_CRYSTAL, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 73 and "31683-02.htm" or "31683-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == EYE_OF_ARGOS then
      if cond < 6 then
        htmltext = "31683-04.htm"
      elseif cond == 6 then
        htmltext = "31683-05.htm"
      elseif cond == 7 then
        htmltext = "31683-08.htm"
      elseif cond == 8 then
        htmltext = "31683-09.htm"
      end
    elseif sw2 == MYSTERIOUS_TABLET_1 then
      if cond == 1 then
        htmltext = "31548-01.htm"
      else
        htmltext = "31548-03.htm"
      end
    elseif sw2 == MYSTERIOUS_TABLET_2 then
      if cond == 2 then
        htmltext = "31549-01.htm"
      elseif cond > 2 then
        htmltext = "31549-03.htm"
      end
    elseif sw2 == MYSTERIOUS_TABLET_3 then
      if cond == 3 then
        htmltext = "31550-01.htm"
      elseif cond > 3 then
        htmltext = "31550-03.htm"
      end
    elseif sw2 == MYSTERIOUS_TABLET_4 then
      if cond == 4 then
        htmltext = "31551-01.htm"
      elseif cond > 4 then
        htmltext = "31551-03.htm"
      end
    elseif sw2 == MYSTERIOUS_TABLET_5 then
      if cond == 5 then
        htmltext = "31552-01.htm"
      elseif cond > 5 then
        htmltext = "31552-03.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local partyMember = q:get_random_party_member(player, 7)
  if partyMember == nil then
    return
  end
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    q:give_items(partyMember, EVIL_SPIRIT_BEADS, 1)
    if q:get_quest_items_count(partyMember, EVIL_SPIRIT_BEADS) < 200 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(8, true)
    end
  end
end)
