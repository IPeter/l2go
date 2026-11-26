-- The Varka Silenos barracks NPCs, ported from Mobius
-- ai/areas/VarkaSilenosBarracks/VarkaSilenosSupport (Nyaran, from
-- Emperorc's and Kerberos's Jython script).
--
-- They talk to Varka's allies only, by the level of their Mark of Varka's
-- Alliance: the buffer Udan buffs for Nepenthes Seeds, the warehouse
-- keeper, the trader and the teleporter open up level by level. As in
-- Mobius, Varka levels are negative (-1 to -5), and so are the teleport
-- list names ("list-1").

local s = script.new("VarkaSilenosSupport")

-- NPCs
local ASHAS = 31377 -- Hierarch
local NARAN = 31378 -- Messenger
local UDAN = 31379 -- Buffer
local DIYABU = 31380 -- Grocer
local HAGOS = 31381 -- Warehouse Keeper
local SHIKON = 31382 -- Trader
local TERANU = 31383 -- Teleporter

-- Items
local SEED = 7187
local VARKA_MARKS = {
  7221, -- Mark of Varka's Alliance - Level 1
  7222, -- Mark of Varka's Alliance - Level 2
  7223, -- Mark of Varka's Alliance - Level 3
  7224, -- Mark of Varka's Alliance - Level 4
  7225, -- Mark of Varka's Alliance - Level 5
}

-- Udan's buffs by link: the skill (level 1) and its price in seeds.
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

s:add_first_talk_id(ASHAS, NARAN, UDAN, DIYABU, HAGOS, SHIKON, TERANU)
s:add_talk_id(UDAN, HAGOS, TERANU)
s:add_start_npc(HAGOS, TERANU)

-- alliance_level is minus the level of the player's first mark, 0 without
-- one.
local function alliance_level(player)
  for i, mark in ipairs(VARKA_MARKS) do
    if s:has_quest_items(player, mark) then
      return -i
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
    if s:get_quest_items_count(player, SEED) < buff.cost then
      return "31379-02.html"
    end
    s:take_items(player, SEED, buff.cost)
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
  return level < 0 and id .. "-friend.html" or id .. "-no.html"
end

s:on_first_talk(function(npc, player)
  local level = alliance_level(player)
  local id = npc:id()
  if id == ASHAS or id == NARAN or id == DIYABU then
    return friend_page(level, id)
  elseif id == UDAN then
    if level == 0 then
      return "31379-03.html"
    end
    return level > -3 and "31379-01.html" or "31379-04.html"
  elseif id == HAGOS then
    if level == 0 then
      return "31381-no.html"
    end
    return level == -1 and "31381-01.html" or "31381-02.html"
  elseif id == SHIKON then
    if level == -1 or level == -2 then
      return "31382-01.html"
    elseif level == -3 or level == -4 then
      return "31382-02.html"
    elseif level == -5 then
      return "31382-03.html"
    end
    return "31382-no.html"
  elseif id == TERANU then
    if level <= -1 and level >= -3 then
      return "31383-01.html"
    elseif level == -4 then
      return "31383-02.html"
    elseif level == -5 then
      return "31383-03.html"
    end
    return "31383-no.html"
  end
  return s:get_no_quest_msg(player)
end)
