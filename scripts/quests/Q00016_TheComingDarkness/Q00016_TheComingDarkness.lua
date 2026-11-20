-- Translated by questport from quests/Q00016_TheComingDarkness (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(16, "Q00016_TheComingDarkness", "The Coming Darkness")

-- NPCs
local HIERARCH = 31517
local EVIL_ALTAR_1 = 31512
local EVIL_ALTAR_2 = 31513
local EVIL_ALTAR_3 = 31514
local EVIL_ALTAR_4 = 31515
local EVIL_ALTAR_5 = 31516
-- Item
local CRYSTAL_OF_SEAL = 7167

q:register_quest_items(CRYSTAL_OF_SEAL)
q:add_start_npc(HIERARCH)
q:add_talk_id(HIERARCH, EVIL_ALTAR_1, EVIL_ALTAR_2, EVIL_ALTAR_3, EVIL_ALTAR_4, EVIL_ALTAR_5)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31517-2.htm" then
    st:start_quest()
    q:give_items(player, CRYSTAL_OF_SEAL, 5)
  elseif event == "31512-1.htm" then
    st:set_cond(2, true)
    q:take_items(player, CRYSTAL_OF_SEAL, 1)
  elseif event == "31513-1.htm" then
    st:set_cond(3, true)
    q:take_items(player, CRYSTAL_OF_SEAL, 1)
  elseif event == "31514-1.htm" then
    st:set_cond(4, true)
    q:take_items(player, CRYSTAL_OF_SEAL, 1)
  elseif event == "31515-1.htm" then
    st:set_cond(5, true)
    q:take_items(player, CRYSTAL_OF_SEAL, 1)
  elseif event == "31516-1.htm" then
    st:set_cond(6, true)
    q:take_items(player, CRYSTAL_OF_SEAL, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 62 and "31517-0a.htm" or "31517-0.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local npcId = npc:id()
    if npcId == HIERARCH then
      if cond == 6 then
        htmltext = "31517-4.htm"
        q:add_exp_and_sp(player, 221958, 0)
        st:exit_quest(false, true)
      else
        if q:has_quest_items(player, CRYSTAL_OF_SEAL) then
          htmltext = "31517-3.htm"
        else
          htmltext = "31517-3a.htm"
          st:exit_quest(true)
        end
      end
    elseif npcId == EVIL_ALTAR_1 or npcId == EVIL_ALTAR_2 or npcId == EVIL_ALTAR_3 or npcId == EVIL_ALTAR_4 or npcId == EVIL_ALTAR_5 then
      local condAltar = npcId - 31511
      if cond == condAltar then
        if q:has_quest_items(player, CRYSTAL_OF_SEAL) then
          htmltext = npcId .. "-0.htm"
        else
          htmltext = "altar_nocrystal.htm"
        end
      elseif cond > condAltar then
        htmltext = npcId .. "-2.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
