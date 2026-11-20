-- The second class change of Dark Elves, ported by hand from Mobius
-- village_master/DarkElfChange2. The masters turn a level 40 first class
-- Dark Elf into its second class for three marks (always the Mark of Fate
-- among them), with 15 Shadow Item Exchange Coupons (C grade) and the
-- fanfare. Every master answers with the 30474-*.html pages.
--
-- The class links send the row of CLASSES (0-based, as in Mobius); every
-- other event is the page it names. The pages are Mobius's, served from
-- data/html/scripts/village_master/DarkElfChange2/.

local J = require("lib.java")
local cm = require("lib.class_master")

local s = script.new("DarkElfChange2")

local NPCS = {
  30195, -- Brecson
  30699, -- Medown
  30474, -- Angus
  31324, -- Andromeda
  30862, -- Oltran
  30910, -- Xairakin
  31285, -- Samael
  31334, -- Tifaren
  31974, -- Drizzit
  32096, -- Helminter
}

-- Marks
local MARK_OF_CHALLENGER = 2627
local MARK_OF_DUTY = 2633
local MARK_OF_SEEKER = 2673
local MARK_OF_SCHOLAR = 2674
local MARK_OF_PILGRIM = 2721
local MARK_OF_DUELIST = 2762
local MARK_OF_SEARCHER = 2809
local MARK_OF_REFORMER = 2821
local MARK_OF_MAGUS = 2840
local MARK_OF_FATE = 3172
local MARK_OF_SAGITTARIUS = 3293
local MARK_OF_WITCHCRAFT = 3307
local MARK_OF_SUMMONER = 3336

-- The new class, the class it comes from, the page suffixes (below level
-- 40 without and with the marks, then at level 40 without and with them)
-- and the three marks. Mobius's rows 0 to 6 are 1 to 7 here. Class IDs are
-- numbers: the ClassId table is filled only when the server runs the
-- handlers.
local CLASSES = {
  { 33, 32, 26, 27, 28, 29, MARK_OF_DUTY, MARK_OF_FATE, MARK_OF_WITCHCRAFT }, -- Shillien Knight from Palus Knight
  { 34, 32, 30, 31, 32, 33, MARK_OF_CHALLENGER, MARK_OF_FATE, MARK_OF_DUELIST }, -- Bladedancer from Palus Knight
  { 43, 42, 34, 35, 36, 37, MARK_OF_PILGRIM, MARK_OF_FATE, MARK_OF_REFORMER }, -- Shillien Elder from Shillien Oracle
  { 36, 35, 38, 39, 40, 41, MARK_OF_SEEKER, MARK_OF_FATE, MARK_OF_SEARCHER }, -- Abyss Walker from Assassin
  { 37, 35, 42, 43, 44, 45, MARK_OF_SEEKER, MARK_OF_FATE, MARK_OF_SAGITTARIUS }, -- Phantom Ranger from Assassin
  { 40, 39, 46, 47, 48, 49, MARK_OF_SCHOLAR, MARK_OF_FATE, MARK_OF_MAGUS }, -- Spellhowler from Dark Wizard
  { 41, 39, 50, 51, 52, 53, MARK_OF_SCHOLAR, MARK_OF_FATE, MARK_OF_SUMMONER }, -- Phantom Summoner from Dark Wizard
}

-- The class lists of the first classes: Palus Knight, Shillien Oracle,
-- Assassin, Dark Wizard.
local CLASS_LISTS = {
  [32] = "30474-01.html",
  [42] = "30474-08.html",
  [35] = "30474-12.html",
  [39] = "30474-19.html",
}

s:add_start_npc(NPCS)
s:add_talk_id(NPCS)

s:on_event(function(event, npc, player)
  if J.is_numeric(event) then
    local row = CLASSES[J.parse_int(event) + 1]
    if row == nil then
      return nil -- Mobius fails on a row it does not have
    end
    local cid = player:class_id()
    if ClassId.race(cid) == Race.DARK_ELF and cid == row[2] then
      local suffix
      local marks = s:has_quest_items(player, row[7]) and s:has_quest_items(player, row[8]) and s:has_quest_items(player, row[9])
      if player:level() < 40 then
        suffix = marks and row[4] or row[3]
      elseif not marks then
        suffix = row[5]
      else
        suffix = row[6]
        s:take_items(player, row[7], -1)
        s:take_items(player, row[8], -1)
        s:take_items(player, row[9], -1)
        s:play_sound(player, QuestSound.ITEMSOUND_QUEST_FANFARE_2)
        player:set_class_id(row[1])
        player:set_base_class(row[1])
        s:give_items(player, cm.COUPON_C_GRADE, 15)
        player:broadcast_user_info()
      end
      return "30474-" .. suffix .. ".html"
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
    local list = CLASS_LISTS[cid]
    if list ~= nil then
      return list
    elseif ClassId.level(cid) == 0 then
      return "30474-55.html" -- first occupation not made yet
    elseif ClassId.level(cid) >= 2 then
      return "30474-54.html" -- second/third occupation change already made
    end
    return "30474-56.html"
  end
  return "30474-56.html" -- other races
end)
