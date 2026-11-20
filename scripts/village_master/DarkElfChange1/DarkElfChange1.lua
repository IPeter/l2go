-- The first class change of Dark Elves, ported by hand from Mobius
-- village_master/DarkElfChange1. The masters (Xenos, Tobias, Tronix) turn a
-- level 20 Dark Fighter into a Palus Knight or Assassin, a Dark Mystic into
-- a Dark Wizard or Shillien Oracle, for the path quest's mark, with 15
-- Shadow Item Exchange Coupons (D grade) and the fanfare.
--
-- The class links send the row of CLASSES (0-based, as in Mobius); every
-- other event is the page it names. The pages are Mobius's, served from
-- data/html/scripts/village_master/DarkElfChange1/.

local J = require("lib.java")
local cm = require("lib.class_master")

local s = script.new("DarkElfChange1")

local NPCS = {
  30290, -- Xenos
  30297, -- Tobias
  30462, -- Tronix
}

-- Items
local GAZE_OF_ABYSS = 1244
local IRON_HEART = 1252
local JEWEL_OF_DARKNESS = 1261
local ORB_OF_ABYSS = 1270

-- The new class, the class it comes from, the page suffixes (below level
-- 20 without and with the proof, then at level 20 without and with it) and
-- the proof. Mobius's rows 0 to 3 are 1 to 4 here. Class IDs are numbers:
-- the ClassId table is filled only when the server runs the handlers.
local CLASSES = {
  { 32, 31, 15, 16, 17, 18, GAZE_OF_ABYSS }, -- Palus Knight from Dark Fighter
  { 35, 31, 19, 20, 21, 22, IRON_HEART }, -- Assassin from Dark Fighter
  { 39, 38, 23, 24, 25, 26, JEWEL_OF_DARKNESS }, -- Dark Wizard from Dark Mystic
  { 42, 38, 27, 28, 29, 30, ORB_OF_ABYSS }, -- Shillien Oracle from Dark Mystic
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

s:on_event(function(event, npc, player)
  if J.is_numeric(event) then
    local row = CLASSES[J.parse_int(event) + 1]
    if row == nil or npc == nil then
      return nil -- Mobius fails on a row it does not have
    end
    local cid = player:class_id()
    if ClassId.race(cid) == Race.DARK_ELF and cid == row[2] then
      local suffix
      local item = s:has_quest_items(player, row[7])
      if player:level() < 20 then
        suffix = item and row[4] or row[3]
      elseif not item then
        suffix = row[5]
      else
        suffix = row[6]
        s:give_items(player, cm.COUPON_D_GRADE, 15)
        s:take_items(player, row[7], -1)
        player:set_class_id(row[1])
        player:set_base_class(row[1])
        s:play_sound(player, QuestSound.ITEMSOUND_QUEST_FANFARE_2)
        player:broadcast_user_info()
      end
      return npc:id() .. "-" .. suffix .. ".html"
    end
  end
  return event
end)

s:on_talk(function(npc, player)
  if player:is_sub_class_active() then
    return s:get_no_quest_msg(player)
  end
  local cid = player:class_id()
  if ClassId.race(cid) == Race.DARK_ELF then
    if cid == ClassId.DARK_FIGHTER then
      return npc:id() .. "-01.html"
    elseif cid == ClassId.DARK_MAGE then
      return npc:id() .. "-08.html"
    elseif ClassId.level(cid) == 1 then
      return npc:id() .. "-32.html" -- first occupation change already made
    elseif ClassId.level(cid) >= 2 then
      return npc:id() .. "-31.html" -- second/third occupation change already made
    end
    return s:get_no_quest_msg(player)
  end
  return npc:id() .. "-33.html" -- other races
end)
