-- The Ketra Orc outpost NPCs, ported from Mobius
-- ai/areas/KetraOrcOutpust/KetraOrcSupport (Nyaran, from Emperorc's and
-- Kerberos's Jython script).
--
-- They talk to Ketra's allies only, by the level of their Mark of Ketra's
-- Alliance: the soul guide Asefa buffs for Buffalo Horns, the warehouse
-- keeper, the trader and the gate keeper open up level by level.

local s = script.new("KetraOrcSupport")

-- NPCs
local KADUN = 31370 -- Hierarch
local WAHKAN = 31371 -- Messenger
local ASEFA = 31372 -- Soul Guide
local ATAN = 31373 -- Grocer
local JAFF = 31374 -- Warehouse Keeper
local JUMARA = 31375 -- Trader
local KURFA = 31376 -- Gate Keeper

-- Items
local HORN = 7186
local KETRA_MARKS = {
  7211, -- Mark of Ketra's Alliance - Level 1
  7212, -- Mark of Ketra's Alliance - Level 2
  7213, -- Mark of Ketra's Alliance - Level 3
  7214, -- Mark of Ketra's Alliance - Level 4
  7215, -- Mark of Ketra's Alliance - Level 5
}

-- Asefa's buffs by link: the skill (level 1) and its price in horns.
local BUFF = {
  [1] = { skill = 4359, cost = 2 }, -- Focus
  [2] = { skill = 4360, cost = 2 }, -- Death Whisper
  [3] = { skill = 4345, cost = 3 }, -- Might
  [4] = { skill = 4355, cost = 3 }, -- Acumen
  [5] = { skill = 4352, cost = 3 }, -- Berserker
  [6] = { skill = 4354, cost = 3 }, -- Vampiric Rage
  [7] = { skill = 4356, cost = 6 }, -- Empower
  [8] = { skill = 4357, cost = 6 }, -- Haste
}

s:add_first_talk_id(KADUN, WAHKAN, ASEFA, ATAN, JAFF, JUMARA, KURFA)
s:add_talk_id(ASEFA, KURFA, JAFF)
s:add_start_npc(KURFA, JAFF)

-- alliance_level is the level of the player's first mark, 0 without one.
local function alliance_level(player)
  for i, mark in ipairs(KETRA_MARKS) do
    if s:has_quest_items(player, mark) then
      return i
    end
  end
  return 0
end

s:on_event(function(event, npc, player)
  if npc == nil or player == nil then
    return nil
  end
  local buff = event:match("^%d+$") and BUFF[tonumber(event)]
  if buff then
    if s:get_quest_items_count(player, HORN) < buff.cost then
      return "31372-02.html"
    end
    s:take_items(player, HORN, buff.cost)
    npc:set_target(player)
    npc:cast(skills.get(buff.skill, 1))
    npc:set_hp_mp(npc:max_hp(), npc:max_mp())
  elseif event == "TELEPORT" then
    npc:show_teleport_list(player, "list" .. alliance_level(player))
  end
  return nil
end)

-- friend_page picks the page by whether the player is an ally.
local function friend_page(level, id)
  return level > 0 and id .. "-friend.html" or id .. "-no.html"
end

s:on_first_talk(function(npc, player)
  local level = alliance_level(player)
  local id = npc:id()
  if id == KADUN or id == WAHKAN or id == ATAN then
    return friend_page(level, id)
  elseif id == ASEFA then
    if level == 0 then
      return "31372-03.html"
    end
    return level < 3 and "31372-01.html" or "31372-04.html"
  elseif id == JAFF then
    if level == 0 then
      return "31374-no.html"
    end
    return level == 1 and "31374-01.html" or "31374-02.html"
  elseif id == JUMARA then
    if level == 1 or level == 2 then
      return "31375-01.html"
    elseif level == 3 or level == 4 then
      return "31375-02.html"
    elseif level == 5 then
      return "31375-03.html"
    end
    return "31375-no.html"
  elseif id == KURFA then
    if level >= 1 and level <= 3 then
      return "31376-01.html"
    elseif level == 4 then
      return "31376-02.html"
    elseif level == 5 then
      return "31376-03.html"
    end
    return "31376-no.html"
  end
  return s:get_no_quest_msg(player)
end)
