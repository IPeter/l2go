-- Translated by questport from quests/Q00627_HeartInSearchOfPower (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(627, "Q00627_HeartInSearchOfPower", "Heart in Search of Power")

-- NPCs
local NECROMANCER = 31518
local ENFEUX = 31519
-- Items
local SEAL_OF_LIGHT = 7170
local BEAD_OF_OBEDIENCE = 7171
local GEM_OF_SAINTS = 7172
-- Drop chances
local CHANCES = {}
-- Rewards
local REWARDS = {}

CHANCES[21520] = 550000
CHANCES[21523] = 584000
CHANCES[21524] = 621000
CHANCES[21525] = 621000
CHANCES[21526] = 606000
CHANCES[21529] = 625000
CHANCES[21530] = 578000
CHANCES[21531] = 690000
CHANCES[21532] = 671000
CHANCES[21535] = 693000
CHANCES[21536] = 615000
CHANCES[21539] = 762000
CHANCES[21540] = 762000
CHANCES[21658] = 690000
REWARDS["adena"] = {0, 0, 100000}
REWARDS["asofe"] = {4043, 13, 6400}
REWARDS["thon"] = {4044, 13, 6400}
REWARDS["enria"] = {4042, 6, 13600}
REWARDS["mold"] = {4041, 6, 17200}

q:register_quest_items(BEAD_OF_OBEDIENCE)
q:add_start_npc(NECROMANCER)
q:add_talk_id(NECROMANCER, ENFEUX)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31518-01.htm" then
    st:start_quest()
  elseif event == "31518-03.htm" then
    if q:get_quest_items_count(player, BEAD_OF_OBEDIENCE) == 300 then
      st:set_cond(3, true)
      q:take_items(player, BEAD_OF_OBEDIENCE, -1)
      q:give_items(player, SEAL_OF_LIGHT, 1)
    else
      st:set_cond(1)
      q:take_items(player, BEAD_OF_OBEDIENCE, -1)
      htmltext = "31518-03a.htm"
    end
  elseif event == "31519-01.htm" then
    if q:get_quest_items_count(player, SEAL_OF_LIGHT) == 1 then
      st:set_cond(4, true)
      q:take_items(player, SEAL_OF_LIGHT, 1)
      q:give_items(player, GEM_OF_SAINTS, 1)
    end
  elseif REWARDS[event] ~= nil then
    if q:get_quest_items_count(player, GEM_OF_SAINTS) == 1 then
      htmltext = "31518-07.htm"
      q:take_items(player, GEM_OF_SAINTS, 1)
      if REWARDS[event][1] > 0 then
        q:give_items(player, REWARDS[event][1], REWARDS[event][2])
      end
      q:give_adena(player, REWARDS[event][3], true)
      st:exit_quest(true, true)
    else
      htmltext = "31518-7.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 60 and "31518-00a.htm" or "31518-00.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == NECROMANCER then
      if cond == 1 then
        htmltext = "31518-01a.htm"
      elseif cond == 2 then
        htmltext = "31518-02.htm"
      elseif cond == 3 then
        htmltext = "31518-04.htm"
      elseif cond == 4 then
        htmltext = "31518-05.htm"
      end
    elseif sw2 == ENFEUX then
      if cond == 3 then
        htmltext = "31519-00.htm"
      elseif cond == 4 then
        htmltext = "31519-02.htm"
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    q:give_items(player, BEAD_OF_OBEDIENCE, 1)
    if q:get_quest_items_count(player, BEAD_OF_OBEDIENCE) < 300 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
