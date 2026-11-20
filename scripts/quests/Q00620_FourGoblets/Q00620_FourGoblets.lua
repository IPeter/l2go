-- Translated by questport from quests/Q00620_FourGoblets (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Needs the Four Sepulchers: the goblets drop from the sepulcher bosses,
-- whose halls do not exist before.
local J = require("lib.java")

local q = quest.new(620, "Q00620_FourGoblets", "Four Goblets")
q:requires("sepulchers")

-- NPCs
local GHOST_OF_WIGOTH_1 = 31452
local NAMELESS_SPIRIT = 31453
local GHOST_OF_WIGOTH_2 = 31454
local GHOST_CHAMBERLAIN_1 = 31919
local GHOST_CHAMBERLAIN_2 = 31920
local CONQUERORS_SEPULCHER_MANAGER = 31921
local EMPERORS_SEPULCHER_MANAGER = 31922
local GREAT_SAGES_SEPULCHER_MANAGER = 31923
local JUDGES_SEPULCHER_MANAGER = 31924
-- Items
local RELIC = 7254
local SEALED_BOX = 7255
local GOBLET_OF_ALECTIA = 7256
local GOBLET_OF_TISHAS = 7257
local GOBLET_OF_MEKARA = 7258
local GOBLET_OF_MORIGUL = 7259
local USED_GRAVE_PASS = 7261
-- Rewards
local ANTIQUE_BROOCH = 7262
local RCP_REWARDS = {6881, 6883, 6885, 6887, 6891, 6893, 6895, 6897, 6899, 7580}

-- Calculate boxes rewards, then return if there was a reward.
-- @param player the Player, used to reward him.
-- @return true if there was a reward, false if not (used to call a "no-reward" html)
local function calculateBoxReward(player)
  local reward = false
  local rnd_ = rnd(5)
  if rnd_ == 0 then
    q:give_adena(player, 10000, true)
    reward = true
  elseif rnd_ == 1 then
    if rnd(1000) < 848 then
      reward = true
      local i = rnd(1000)
      if i < 43 then
        q:give_items(player, 1884, 42)
      elseif i < 66 then
        q:give_items(player, 1895, 36)
      elseif i < 184 then
        q:give_items(player, 1876, 4)
      elseif i < 250 then
        q:give_items(player, 1881, 6)
      elseif i < 287 then
        q:give_items(player, 5549, 8)
      elseif i < 484 then
        q:give_items(player, 1874, 1)
      elseif i < 681 then
        q:give_items(player, 1889, 1)
      elseif i < 799 then
        q:give_items(player, 1877, 1)
      elseif i < 902 then
        q:give_items(player, 1894, 1)
      else
        q:give_items(player, 4043, 1)
      end
    end
    if rnd(1000) < 323 then
      reward = true
      local i = rnd(1000)
      if i < 335 then
        q:give_items(player, 1888, 1)
      elseif i < 556 then
        q:give_items(player, 4040, 1)
      elseif i < 725 then
        q:give_items(player, 1890, 1)
      elseif i < 872 then
        q:give_items(player, 5550, 1)
      elseif i < 962 then
        q:give_items(player, 1893, 1)
      elseif i < 986 then
        q:give_items(player, 4046, 1)
      else
        q:give_items(player, 4048, 1)
      end
    end
  elseif rnd_ == 2 then
    if rnd(1000) < 847 then
      reward = true
      local i = rnd(1000)
      if i < 148 then
        q:give_items(player, 1878, 8)
      elseif i < 175 then
        q:give_items(player, 1882, 24)
      elseif i < 273 then
        q:give_items(player, 1879, 4)
      elseif i < 322 then
        q:give_items(player, 1880, 6)
      elseif i < 357 then
        q:give_items(player, 1885, 6)
      elseif i < 554 then
        q:give_items(player, 1875, 1)
      elseif i < 685 then
        q:give_items(player, 1883, 1)
      elseif i < 803 then
        q:give_items(player, 5220, 1)
      elseif i < 901 then
        q:give_items(player, 4039, 1)
      else
        q:give_items(player, 4044, 1)
      end
    end
    if rnd(1000) < 251 then
      reward = true
      local i = rnd(1000)
      if i < 350 then
        q:give_items(player, 1887, 1)
      elseif i < 587 then
        q:give_items(player, 4042, 1)
      elseif i < 798 then
        q:give_items(player, 1886, 1)
      elseif i < 922 then
        q:give_items(player, 4041, 1)
      elseif i < 966 then
        q:give_items(player, 1892, 1)
      elseif i < 996 then
        q:give_items(player, 1891, 1)
      else
        q:give_items(player, 4047, 1)
      end
    end
  elseif rnd_ == 3 then
    if rnd(1000) < 31 then
      reward = true
      local i = rnd(1000)
      if i < 223 then
        q:give_items(player, 730, 1)
      elseif i < 893 then
        q:give_items(player, 948, 1)
      else
        q:give_items(player, 960, 1)
      end
    end
    if rnd(1000) < 5 then
      reward = true
      local i = rnd(1000)
      if i < 202 then
        q:give_items(player, 729, 1)
      elseif i < 928 then
        q:give_items(player, 947, 1)
      else
        q:give_items(player, 959, 1)
      end
    end
  elseif rnd_ == 4 then
    if rnd(1000) < 329 then
      reward = true
      local i = rnd(1000)
      if i < 88 then
        q:give_items(player, 6698, 1)
      elseif i < 185 then
        q:give_items(player, 6699, 1)
      elseif i < 238 then
        q:give_items(player, 6700, 1)
      elseif i < 262 then
        q:give_items(player, 6701, 1)
      elseif i < 292 then
        q:give_items(player, 6702, 1)
      elseif i < 356 then
        q:give_items(player, 6703, 1)
      elseif i < 420 then
        q:give_items(player, 6704, 1)
      elseif i < 482 then
        q:give_items(player, 6705, 1)
      elseif i < 554 then
        q:give_items(player, 6706, 1)
      elseif i < 576 then
        q:give_items(player, 6707, 1)
      elseif i < 640 then
        q:give_items(player, 6708, 1)
      elseif i < 704 then
        q:give_items(player, 6709, 1)
      elseif i < 777 then
        q:give_items(player, 6710, 1)
      elseif i < 799 then
        q:give_items(player, 6711, 1)
      elseif i < 863 then
        q:give_items(player, 6712, 1)
      elseif i < 927 then
        q:give_items(player, 6713, 1)
      else
        q:give_items(player, 6714, 1)
      end
    end
    if rnd(1000) < 54 then
      reward = true
      local i = rnd(1000)
      if i < 100 then
        q:give_items(player, 6688, 1)
      elseif i < 198 then
        q:give_items(player, 6689, 1)
      elseif i < 298 then
        q:give_items(player, 6690, 1)
      elseif i < 398 then
        q:give_items(player, 6691, 1)
      elseif i < 499 then
        q:give_items(player, 7579, 1)
      elseif i < 601 then
        q:give_items(player, 6693, 1)
      elseif i < 703 then
        q:give_items(player, 6694, 1)
      elseif i < 801 then
        q:give_items(player, 6695, 1)
      elseif i < 902 then
        q:give_items(player, 6696, 1)
      else
        q:give_items(player, 6697, 1)
      end
    end
  end
  return reward
end

q:register_quest_items(SEALED_BOX, USED_GRAVE_PASS, GOBLET_OF_ALECTIA, GOBLET_OF_TISHAS, GOBLET_OF_MEKARA, GOBLET_OF_MORIGUL)
q:add_start_npc(NAMELESS_SPIRIT, CONQUERORS_SEPULCHER_MANAGER, EMPERORS_SEPULCHER_MANAGER, GREAT_SAGES_SEPULCHER_MANAGER, JUDGES_SEPULCHER_MANAGER, GHOST_CHAMBERLAIN_1, GHOST_CHAMBERLAIN_2)
q:add_talk_id(NAMELESS_SPIRIT, CONQUERORS_SEPULCHER_MANAGER, EMPERORS_SEPULCHER_MANAGER, GREAT_SAGES_SEPULCHER_MANAGER, JUDGES_SEPULCHER_MANAGER, GHOST_CHAMBERLAIN_1, GHOST_CHAMBERLAIN_2, GHOST_OF_WIGOTH_1, GHOST_OF_WIGOTH_2)
for id = 18120, 18256 do
  q:add_kill_id(id)
end

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31452-05.htm" then
    if rnd.bool() then
      htmltext = rnd.bool() and "31452-03.htm" or "31452-04.htm"
    end
  elseif event == "EXIT" then
    player:teleport(169590, -90218, -2914) -- Wigoth : Teleport back to Pilgrim's Temple
  elseif event == "31453-13.htm" then
    st:start_quest()
  elseif event == "31453-16.htm" then
    if q:has_quest_items(player, GOBLET_OF_ALECTIA, GOBLET_OF_TISHAS, GOBLET_OF_MEKARA, GOBLET_OF_MORIGUL) then
      st:set_cond(2, true)
      q:take_items(player, GOBLET_OF_ALECTIA, -1)
      q:take_items(player, GOBLET_OF_TISHAS, -1)
      q:take_items(player, GOBLET_OF_MEKARA, -1)
      q:take_items(player, GOBLET_OF_MORIGUL, -1)
      q:give_items(player, ANTIQUE_BROOCH, 1)
    else
      htmltext = "31453-14.htm"
    end
  elseif event == "31453-18.htm" then
    st:exit_quest(true, true)
  elseif event == "boxes" then
    if q:has_quest_items(player, SEALED_BOX) then
      q:take_items(player, SEALED_BOX, 1)
      if not calculateBoxReward(player) then
        htmltext = rnd.bool() and "31454-09.htm" or "31454-10.htm"
      else
        htmltext = "31454-08.htm"
      end
    end
  elseif event == "TELEPORT_TO_4S" then
    if q:has_quest_items(player, ANTIQUE_BROOCH) then
      player:teleport(178298, -84574, -7216)
      return nil
    end
    if q:has_quest_items(player, USED_GRAVE_PASS) then
      q:take_items(player, USED_GRAVE_PASS, 1)
      player:teleport(178298, -84574, -7216)
      return nil
    end
    htmltext = npc:id() .. "-00.htm"
  elseif event == "TELEPORT_TO_IT" then
    if q:has_quest_items(player, ANTIQUE_BROOCH) then
      player:teleport(186942, -75602, -2834)
      return nil
    end
    if q:has_quest_items(player, USED_GRAVE_PASS) then
      q:take_items(player, USED_GRAVE_PASS, 1)
      player:teleport(186942, -75602, -2834)
      return nil
    end
    htmltext = npc:id() .. "-00.htm"
  elseif event == "31919-06.htm" then
    if q:has_quest_items(player, SEALED_BOX) then
      q:take_items(player, SEALED_BOX, 1)
      if not calculateBoxReward(player) then
        htmltext = rnd.bool() and "31919-04.htm" or "31919-05.htm"
      else
        htmltext = "31919-03.htm"
      end
    end
  elseif J.is_numeric(event) then
    local id = J.parse_int(event)
    if J.contains(RCP_REWARDS, id) and q:get_quest_items_count(player, RELIC) >= 1000 then
      q:take_items(player, RELIC, 1000)
      q:give_items(player, id, 1)
    end
    htmltext = "31454-12.htm"
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local npcId = npc:id()
  local cond = st:get_cond()
  if npcId == GHOST_OF_WIGOTH_1 then
    if cond == 1 then
      htmltext = "31452-01.htm"
    elseif cond == 2 then
      htmltext = "31452-02.htm"
    end
  elseif npcId == NAMELESS_SPIRIT then
    if cond == 0 then
      htmltext = player:level() >= 74 and "31453-01.htm" or "31453-12.htm"
    elseif cond == 1 then
      htmltext = q:has_quest_items(player, GOBLET_OF_ALECTIA, GOBLET_OF_TISHAS, GOBLET_OF_MEKARA, GOBLET_OF_MORIGUL) and "31453-15.htm" or "31453-14.htm"
    elseif cond == 2 then
      htmltext = "31453-17.htm"
    end
  elseif npcId == GHOST_OF_WIGOTH_2 then
    -- Possibilities : 0 = nothing, 1 = seal boxes only, 2 = relics only, 3 = both, 4/5/6/7 = "4 goblets" versions of 0/1/2/3.
    local index = 0
    if q:has_quest_items(player, GOBLET_OF_ALECTIA, GOBLET_OF_TISHAS, GOBLET_OF_MEKARA, GOBLET_OF_MORIGUL) then
      index = 4
    end
    local gotSealBoxes = q:has_quest_items(player, SEALED_BOX)
    local gotEnoughRelics = q:get_quest_items_count(player, RELIC) >= 1000
    if gotSealBoxes and gotEnoughRelics then
      index = index + 3
    elseif not gotSealBoxes and gotEnoughRelics then
      index = index + 2
    elseif gotSealBoxes then
      index = index + 1
    end
    htmltext = "31454-0" .. index .. ".htm"
  else
    htmltext = npcId .. "-01.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil or not st:is_started() then
    return
  end
  local partyMember = st:get_player()
  if rnd(10) < 3 then
    q:give_items(partyMember, SEALED_BOX, 1)
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
