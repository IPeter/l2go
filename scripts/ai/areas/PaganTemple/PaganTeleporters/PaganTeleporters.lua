-- The Pagan Temple gates and Triol's mirrors, ported from Mobius
-- ai/areas/PaganTemple/PaganTeleporters (Plim, from BiTi's Jython script).
--
-- The gate keepers open the temple doors for 10 s: the entrance door for
-- a visitor's or a pagan's mark (32034; 32035 opens it from the inside),
-- the chapel doors for a pagan's mark (32036; 32037 from the inside).
-- Triol's mirrors take whoever talks to them through the mirror.

local s = script.new("PaganTeleporters")

local TRIOLS_MIRROR_1, TRIOLS_MIRROR_2 = 32039, 32040
local NPCS = { 32034, 32035, 32036, 32037, TRIOLS_MIRROR_1, TRIOLS_MIRROR_2 }

-- Where each mirror leads.
local TRIOLS_LOCS = {
  [TRIOLS_MIRROR_1] = { x = -12766, y = -35840, z = -10856 },
  [TRIOLS_MIRROR_2] = { x = 36640, y = -51218, z = 718 },
}

-- Items
local VISITORS_MARK = 8064
local FADED_VISITORS_MARK = 8065
local PAGANS_MARK = 8067

-- Doors
local ENTRANCE_DOOR = 19160001
local CHAPEL_DOORS = { 19160010, 19160011 }
local DOOR_OPEN_MS = 10000

s:add_start_npc(NPCS)

local function open_entrance()
  doors.open(ENTRANCE_DOOR)
  s:start_quest_timer("CLOSE_DOOR_1", DOOR_OPEN_MS, nil, nil)
end

local function open_chapel()
  for _, id in ipairs(CHAPEL_DOORS) do
    doors.open(id)
  end
  s:start_quest_timer("CLOSE_DOOR_2", DOOR_OPEN_MS, nil, nil)
end

s:on_event(function(event, npc, player)
  if event == "CLOSE_DOOR_1" then
    doors.close(ENTRANCE_DOOR)
  elseif event == "CLOSE_DOOR_2" then
    for _, id in ipairs(CHAPEL_DOORS) do
      doors.close(id)
    end
  end
  return ""
end)

s:on_first_talk({ TRIOLS_MIRROR_1, TRIOLS_MIRROR_2 }, function(npc, player)
  local loc = TRIOLS_LOCS[npc:id()]
  if loc ~= nil then
    player:teleport(loc)
  end
  return ""
end)

s:on_talk(NPCS, function(npc, player)
  local id = npc:id()
  if id == 32034 then
    if not s:has_at_least_one_quest_item(player, VISITORS_MARK, FADED_VISITORS_MARK, PAGANS_MARK) then
      return "noItem.htm"
    end
    open_entrance()
    return "FadedMark.htm"
  elseif id == 32035 then
    open_entrance()
    return "FadedMark.htm"
  elseif id == 32036 then
    if not s:has_quest_items(player, PAGANS_MARK) then
      return "noMark.htm"
    end
    open_chapel()
    return "openDoor.htm"
  elseif id == 32037 then
    open_chapel()
    return "FadedMark.htm"
  end
  return nil
end)
