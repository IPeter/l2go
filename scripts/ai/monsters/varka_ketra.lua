-- Varka Silenos Barracks and Ketra Orc Outpost, ported from Mobius
-- ai/others/VarkaKetra (malyelfik).
--
-- A side's monsters leave its allies (players with one of its alliance
-- marks) alone. An ally who kills one of them, and each party member
-- within 1,500, loses one level of the alliance (the mark becomes the one
-- a level below; a level 1 mark is taken) and the side's started quests.

local s = script.new("VarkaKetra")

local KETRA = {
  21324, -- Ketra Orc Footman
  21325, -- Ketra's War Hound
  21327, -- Ketra Orc Raider
  21328, -- Ketra Orc Scout
  21329, -- Ketra Orc Shaman
  21331, -- Ketra Orc Warrior
  21332, -- Ketra Orc Lieutenant
  21334, -- Ketra Orc Medium
  21335, -- Ketra Orc Elite Soldier
  21336, -- Ketra Orc White Captain
  21338, -- Ketra Orc Seer
  21339, -- Ketra Orc General
  21340, -- Ketra Orc Battalion Commander
  21342, -- Ketra Orc Grand Seer
  21343, -- Ketra Commander
  21344, -- Ketra Elite Guard
  21345, -- Ketra's Head Shaman
  21346, -- Ketra's Head Guard
  21347, -- Ketra Prophet
  21348, -- Prophet's Guard
  21349, -- Prophet's Aide
  25299, -- Ketra's Hero Hekaton (Raid Boss)
  25302, -- Ketra's Commander Tayr (Raid Boss)
  25305, -- Ketra's Chief Brakki (Raid Boss)
  25306, -- Soul of Fire Nastron (Raid Boss)
}

local VARKA = {
  21350, -- Varka Silenos Recruit
  21351, -- Varka Silenos Footman
  21353, -- Varka Silenos Scout
  21354, -- Varka Silenos Hunter
  21355, -- Varka Silenos Shaman
  21357, -- Varka Silenos Priest
  21358, -- Varka Silenos Warrior
  21360, -- Varka Silenos Medium
  21361, -- Varka Silenos Magus
  21362, -- Varka Silenos Officer
  21364, -- Varka Silenos Seer
  21365, -- Varka Silenos Great Magus
  21366, -- Varka Silenos General
  21368, -- Varka Silenos Great Seer
  21369, -- Varka's Commander
  21370, -- Varka's Elite Guard
  21371, -- Varka's Head Magus
  21372, -- Varka's Head Guard
  21373, -- Varka's Prophet
  21374, -- Prophet's Guard
  21375, -- Disciple of Prophet
  25309, -- Varka's Hero Shadith (Raid Boss)
  25312, -- Varka's Commander Mos (Raid Boss)
  25315, -- Varka's Chief Horus (Raid Boss)
  25316, -- Soul of Water Ashutar (Raid Boss)
}

-- The alliance marks, level 1 to 5.
local KETRA_MARKS = { 7211, 7212, 7213, 7214, 7215 }
local VARKA_MARKS = { 7221, 7222, 7223, 7224, 7225 }

local KETRA_QUESTS = {
  "Q00605_AllianceWithKetraOrcs",
  "Q00606_BattleAgainstVarkaSilenos",
  "Q00607_ProveYourCourageKetra",
  "Q00608_SlayTheEnemyCommanderKetra",
  "Q00609_MagicalPowerOfWaterPart1",
  "Q00610_MagicalPowerOfWaterPart2",
}

local VARKA_QUESTS = {
  "Q00611_AllianceWithVarkaSilenos",
  "Q00612_BattleAgainstKetraOrcs",
  "Q00613_ProveYourCourageVarka",
  "Q00614_SlayTheEnemyCommanderVarka",
  "Q00615_MagicalPowerOfFirePart1",
  "Q00616_MagicalPowerOfFirePart2",
}

local PARTY_RANGE = 1500

-- is_ketra[id] is true for Ketra's monsters.
local is_ketra = {}
for _, id in ipairs(KETRA) do
  is_ketra[id] = true
end

local is_varka = {}
for _, id in ipairs(VARKA) do
  is_varka[id] = true
end

s:add_kill_id(KETRA, VARKA)
s:add_npc_hate_id(KETRA, VARKA)

-- decrease_alliance swaps the lowest mark the player has for the one below
-- it (the first is only taken).
local function decrease_alliance(player, marks)
  for i, mark in ipairs(marks) do
    if s:has_quest_items(player, mark) then
      s:take_items(player, mark, -1)
      if i > 1 then
        s:give_items(player, marks[i - 1], 1)
      end
      return
    end
  end
end

local function exit_quests(player, quests)
  for _, name in ipairs(quests) do
    local st = player:get_quest_state(name)
    if st ~= nil and st:is_started() then
      st:exit_quest(true)
    end
  end
end

s:on_action_for_each_player(function(player, npc, is_summon)
  if not util.in_range(PARTY_RANGE, player, npc, false) then
    return
  end
  if is_ketra[npc:id()] and s:has_at_least_one_quest_item(player, KETRA_MARKS) then
    decrease_alliance(player, KETRA_MARKS)
    exit_quests(player, KETRA_QUESTS)
  elseif is_varka[npc:id()] and s:has_at_least_one_quest_item(player, VARKA_MARKS) then
    decrease_alliance(player, VARKA_MARKS)
    exit_quests(player, VARKA_QUESTS)
  end
end)

s:on_kill(function(npc, killer, is_summon)
  s:execute_for_each_player(killer, npc, is_summon, true, false)
end)

s:on_npc_hate(function(npc, player, is_summon)
  if is_ketra[npc:id()] then
    return not s:has_at_least_one_quest_item(player, KETRA_MARKS)
  end
  return not s:has_at_least_one_quest_item(player, VARKA_MARKS)
end)
