-- Translated by questport from quests/Q00386_StolenDignity (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Stolen Dignity (386)
-- @author Zealar
local J = require("lib.java")

local q = quest.new(386, "Q00386_StolenDignity", "Stolen Dignity")

-- NPCs
local WAREHOUSE_KEEPER_ROMP = 30843
-- Monsters
local CRIMSON_DRAKE = 20670
local KADIOS = 20671
local HUNGRY_CORPSE = 20954
local PAST_KNIGHT = 20956
local BLADE_DEATH = 20958
local DARK_GUARD = 20959
local BLOODY_GHOST = 20960
local BLOODY_LORD = 20963
local PAST_CREATURE = 20967
local GIANT_SHADOW = 20969
local ANCIENTS_SOLDIER = 20970
local ANCIENTS_WARRIOR = 20971
local SPITE_SOUL_LEADER = 20974
local SPITE_SOUL_WIZARD = 20975
local WRECKED_ARCHER = 21001
local FLOAT_OF_GRAVE = 21003
local GRAVE_PREDATOR = 21005
local FALLEN_ORC_SHAMAN = 21020
local SHARP_TALON_TIGER = 21021
local GLOW_WISP = 21108
local MARSH_PREDATOR = 21110
local HAMES_ORC_SNIPER = 21113
local CURSED_GUARDIAN = 21114
local HAMES_ORC_CHIEFTAIN = 21116
local FALLEN_ORC_SHAMAN_TRANS = 21258
local SHARP_TALON_TIGER_TRANS = 21259
-- Items
local Q_STOLEN_INF_ORE = 6363
-- Reward
local DRAGON_SLAYER_EDGE = 5529
local METEOR_SHOWER_HEAD = 5532
local ELYSIAN_HEAD = 5533
local SOUL_BOW_SHAFT = 5534
local CARNIUM_BOW_SHAFT = 5535
local BLOODY_ORCHID_HEAD = 5536
local SOUL_SEPARATOR_HEAD = 5537
local DRAGON_GRINDER_EDGE = 5538
local BLOOD_TORNADO_EDGE = 5539
local TALLUM_GLAIVE_EDGE = 5541
local HALBARD_EDGE = 5542
local DASPARIONS_STAFF_HEAD = 5543
local WORLDTREES_BRANCH_HEAD = 5544
local DARK_LEGIONS_EDGE_EDGE = 5545
local SWORD_OF_MIRACLE_EDGE = 5546
local ELEMENTAL_SWORD_EDGE = 5547
local TALLUM_BLADE_EDGE = 5548
local INFERNO_MASTER_BLADE = 8331
local EYE_OF_SOUL_PIECE = 8341
local DRAGON_FLAME_HEAD_PIECE = 8342
local DOOM_CRUSHER_HEAD = 8349
local HAMMER_OF_DESTROYER_PIECE = 8346
local SIRR_BLADE_BLADE = 8712
local SWORD_OF_IPOS_BLADE = 8713
local BARAKIEL_AXE_PIECE = 8714
local TUNING_FORK_OF_BEHEMOTH_PIECE = 8715
local NAGA_STORM_PIECE = 8716
local TIPHON_SPEAR_EDGE = 8717
local SHYID_BOW_SHAFT = 8718
local SOBEKK_HURRICANE_EDGE = 8719
local TONGUE_OF_THEMIS_PIECE = 8720
local HAND_OF_CABRIO_HEAD = 8721
local CRYSTAL_OF_DEAMON_PIECE = 8722

local takeHtml, fillBoard, colorBoard, beforeReward, reward, createBingoBoard, getMatchedBingoLineCount, selectBingoNumber
local isSelectedBingoNumber, getNumberFromBingoBoard, getBingoSelectCount, getRandomPlayerFromParty

function takeHtml(player, qs, num)
  local html_ = nil
  local i3
  if not isSelectedBingoNumber(qs, num) then
    selectBingoNumber(qs, num)
    i3 = getBingoSelectCount(qs)
    if i3 == 2 then
      html_ = q:get_htm(player, "30843-14.html")
    elseif i3 == 3 then
      html_ = q:get_htm(player, "30843-16.html")
    elseif i3 == 4 then
      html_ = q:get_htm(player, "30843-18.html")
    elseif i3 == 5 then
      html_ = q:get_htm(player, "30843-20.html")
    end
    return fillBoard(qs, html_)
  end
  i3 = getBingoSelectCount(qs)
  if i3 == 1 then
    html_ = q:get_htm(player, "30843-15.html")
  elseif i3 == 2 then
    html_ = q:get_htm(player, "30843-17.html")
  elseif i3 == 3 then
    html_ = q:get_htm(player, "30843-19.html")
  elseif i3 == 4 then
    html_ = q:get_htm(player, "30843-21.html")
  end
  return fillBoard(qs, html_)
end

function fillBoard(qs, html_)
  local result = html_
  for i0 = 0, 8 do
    local i1 = getNumberFromBingoBoard(qs, i0)
    if isSelectedBingoNumber(qs, i1) then
      result = J.replace(result, "<?Cell" .. (i0 + 1) .. "?>", tostring(i1))
    else
      result = J.replace(result, "<?Cell" .. (i0 + 1) .. "?>", "?")
    end
  end
  return result
end

function colorBoard(qs, html_)
  local result = html_
  for i0 = 0, 8 do
    local i1 = getNumberFromBingoBoard(qs, i0)
    result = J.replace(result, "<?FontColor" .. (i0 + 1) .. "?>", isSelectedBingoNumber(qs, i1) and "ff0000" or "ffffff")
    result = J.replace(result, "<?Cell" .. (i0 + 1) .. "?>", tostring(i1))
  end
  return result
end

function beforeReward(player, qs, num)
  if not isSelectedBingoNumber(qs, num) then
    selectBingoNumber(qs, num)
    local i3 = getMatchedBingoLineCount(qs)
    local html_
    if i3 == 3 and getBingoSelectCount(qs) == 6 then
      reward(player, qs, 4)
      html_ = q:get_htm(player, "30843-22.html")
    elseif i3 == 0 and getBingoSelectCount(qs) == 6 then
      reward(player, qs, 10)
      html_ = q:get_htm(player, "30843-24.html")
    else
      html_ = q:get_htm(player, "30843-23.html")
    end
    return colorBoard(qs, html_)
  end
  return fillBoard(qs, q:get_htm(player, "30843-25.html"))
end

function reward(player, qs, count)
  local sw = rnd(33)
  if sw == 0 then
    q:give_items(player, DRAGON_SLAYER_EDGE, count)
  elseif sw == 1 then
    q:give_items(player, METEOR_SHOWER_HEAD, count)
  elseif sw == 2 then
    q:give_items(player, ELYSIAN_HEAD, count)
  elseif sw == 3 then
    q:give_items(player, SOUL_BOW_SHAFT, count)
  elseif sw == 4 then
    q:give_items(player, CARNIUM_BOW_SHAFT, count)
  elseif sw == 5 then
    q:give_items(player, BLOODY_ORCHID_HEAD, count)
  elseif sw == 6 then
    q:give_items(player, SOUL_SEPARATOR_HEAD, count)
  elseif sw == 7 then
    q:give_items(player, DRAGON_GRINDER_EDGE, count)
  elseif sw == 8 then
    q:give_items(player, BLOOD_TORNADO_EDGE, count)
  elseif sw == 9 then
    q:give_items(player, TALLUM_GLAIVE_EDGE, count)
  elseif sw == 10 then
    q:give_items(player, HALBARD_EDGE, count)
  elseif sw == 11 then
    q:give_items(player, DASPARIONS_STAFF_HEAD, count)
  elseif sw == 12 then
    q:give_items(player, WORLDTREES_BRANCH_HEAD, count)
  elseif sw == 13 then
    q:give_items(player, DARK_LEGIONS_EDGE_EDGE, count)
  elseif sw == 14 then
    q:give_items(player, SWORD_OF_MIRACLE_EDGE, count)
  elseif sw == 15 then
    q:give_items(player, ELEMENTAL_SWORD_EDGE, count)
  elseif sw == 16 then
    q:give_items(player, TALLUM_BLADE_EDGE, count)
  elseif sw == 17 then
    q:give_items(player, INFERNO_MASTER_BLADE, count)
  elseif sw == 18 then
    q:give_items(player, EYE_OF_SOUL_PIECE, count)
  elseif sw == 19 then
    q:give_items(player, DRAGON_FLAME_HEAD_PIECE, count)
  elseif sw == 20 then
    q:give_items(player, DOOM_CRUSHER_HEAD, count)
  elseif sw == 21 then
    q:give_items(player, HAMMER_OF_DESTROYER_PIECE, count)
  elseif sw == 22 then
    q:give_items(player, SIRR_BLADE_BLADE, count)
  elseif sw == 23 then
    q:give_items(player, SWORD_OF_IPOS_BLADE, count)
  elseif sw == 24 then
    q:give_items(player, BARAKIEL_AXE_PIECE, count)
  elseif sw == 25 then
    q:give_items(player, TUNING_FORK_OF_BEHEMOTH_PIECE, count)
  elseif sw == 26 then
    q:give_items(player, NAGA_STORM_PIECE, count)
  elseif sw == 27 then
    q:give_items(player, TIPHON_SPEAR_EDGE, count)
  elseif sw == 28 then
    q:give_items(player, SHYID_BOW_SHAFT, count)
  elseif sw == 29 then
    q:give_items(player, SOBEKK_HURRICANE_EDGE, count)
  elseif sw == 30 then
    q:give_items(player, TONGUE_OF_THEMIS_PIECE, count)
  elseif sw == 31 then
    q:give_items(player, HAND_OF_CABRIO_HEAD, count)
  elseif sw == 32 then
    q:give_items(player, CRYSTAL_OF_DEAMON_PIECE, count)
  end
end

-- @param qs
function createBingoBoard(qs)
  local arr = {1, 2, 3, 4, 5, 6, 7, 8, 9}
  J.shuffle(arr)
  qs:set("numbers", (string.gsub(J.list_to_string(arr), "[^%d ]", "")))
  qs:set("selected", "? ? ? ? ? ? ? ? ?")
end

-- @param qs
-- @return
function getMatchedBingoLineCount(qs)
  local q_ = J.split(qs:get("selected"), " ")
  local found = 0
  -- Horizontal
  if J.is_numeric(q_[1] .. q_[2] .. q_[3]) then
    found = found + 1
  end
  if J.is_numeric(q_[4] .. q_[5] .. q_[6]) then
    found = found + 1
  end
  if J.is_numeric(q_[7] .. q_[8] .. q_[9]) then
    found = found + 1
  end
  -- Vertical
  if J.is_numeric(q_[1] .. q_[4] .. q_[7]) then
    found = found + 1
  end
  if J.is_numeric(q_[2] .. q_[5] .. q_[8]) then
    found = found + 1
  end
  if J.is_numeric(q_[3] .. q_[6] .. q_[9]) then
    found = found + 1
  end
  -- Diagonal
  if J.is_numeric(q_[1] .. q_[5] .. q_[9]) then
    found = found + 1
  end
  if J.is_numeric(q_[3] .. q_[5] .. q_[7]) then
    found = found + 1
  end
  return found
end

-- @param qs
-- @param num
function selectBingoNumber(qs, num)
  local numbers = J.split(qs:get("numbers"), " ")
  local pos = 0
  for i = 0, #numbers - 1 do
    if J.parse_int(numbers[i + 1]) == num then
      pos = i
      break
    end
  end
  local selected = J.split(qs:get("selected"), " ")
  for i = 0, #selected - 1 do
    if i == pos then
      selected[i + 1] = tostring(num)
    end
  end
  local result = selected[1]
  for i = 1, #selected - 1 do
    result = result .. " " .. selected[i + 1]
  end
  qs:set("selected", result)
end

-- @param qs
-- @param num
-- @return
function isSelectedBingoNumber(qs, num)
  return J.str_contains(qs:get("selected"), tostring(num))
end

-- @param qs
-- @param num
-- @return
function getNumberFromBingoBoard(qs, num)
  return J.parse_int(J.split(qs:get("numbers"), " ")[num + 1])
end

-- @param qs
-- @return
function getBingoSelectCount(qs)
  return #(string.gsub(qs:get("selected"), "%D", ""))
end

function getRandomPlayerFromParty(player, npc)
  local qs = q:get_quest_state(player, false)
  local candidates = {}
  if qs ~= nil and qs:is_started() then
    table.insert(candidates, qs)
    table.insert(candidates, qs)
  end
  if player:is_in_party() then
    for _, pm in ipairs(player:party_members()) do
      local qss = q:get_quest_state(pm, false)
      if qss ~= nil and qss:is_started() and util.in_range(Config.ALT_PARTY_RANGE, npc, pm, true) then
        table.insert(candidates, qss)
      end
    end
  end
  if #candidates == 0 then
    return nil
  else
    return candidates[rnd(#candidates) + 1]
  end
end

q:add_start_npc(WAREHOUSE_KEEPER_ROMP)
q:add_talk_id(WAREHOUSE_KEEPER_ROMP)
q:add_kill_id(CRIMSON_DRAKE, KADIOS, HUNGRY_CORPSE, PAST_KNIGHT, BLADE_DEATH, DARK_GUARD, BLOODY_GHOST, BLOODY_LORD, PAST_CREATURE, GIANT_SHADOW, ANCIENTS_SOLDIER, ANCIENTS_WARRIOR, SPITE_SOUL_LEADER, SPITE_SOUL_WIZARD, WRECKED_ARCHER, FLOAT_OF_GRAVE, GRAVE_PREDATOR, FALLEN_ORC_SHAMAN, SHARP_TALON_TIGER, GLOW_WISP, MARSH_PREDATOR, HAMES_ORC_SNIPER, CURSED_GUARDIAN, HAMES_ORC_CHIEFTAIN, FALLEN_ORC_SHAMAN_TRANS, SHARP_TALON_TIGER_TRANS)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  if npc:id() == WAREHOUSE_KEEPER_ROMP then
    if qs:is_created() then
      if player:level() >= 58 then
        return "30843-01.htm"
      end
      return "30843-04.html"
    end
    if q:get_quest_items_count(player, Q_STOLEN_INF_ORE) < 100 then
      return "30843-06.html"
    end
    return "30843-07.html"
  end
  return htmltext
end)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil and npc:id() == WAREHOUSE_KEEPER_ROMP then
    if event == "QUEST_ACCEPTED" then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
      qs:set_memo_state(336)
      qs:start_quest()
      qs:show_question_mark(336)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      return "30843-05.htm"
    end
    if J.str_contains(event, ".html") then
      return event
    end
    local ask = J.parse_int(event)
    if ask == 3 then
      return "30843-09a.html"
    elseif ask == 5 then
      return "30843-03.html"
    elseif ask == 6 then
      qs:exit_quest(true)
      return "30843-08.html"
    elseif ask == 9 then
      return "30843-09.htm"
    elseif ask == 8 then
      if q:get_quest_items_count(player, Q_STOLEN_INF_ORE) >= 100 then
        q:take_items(player, Q_STOLEN_INF_ORE, 100)
        createBingoBoard(qs)
        return "30843-12.html"
      end
      return "30843-11.html"
    elseif ask == 10 or ask == 11 or ask == 12 or ask == 13 or ask == 14 or ask == 15 or ask == 16 or ask == 17 or ask == 18 then
      selectBingoNumber(qs, ask - 10 + 1)
      return fillBoard(qs, q:get_htm(player, "30843-13.html"))
    elseif ask == 19 then
      return takeHtml(player, qs, 1)
    elseif ask == 20 then
      return takeHtml(player, qs, 2)
    elseif ask == 21 then
      return takeHtml(player, qs, 3)
    elseif ask == 22 then
      return takeHtml(player, qs, 4)
    elseif ask == 23 then
      return takeHtml(player, qs, 5)
    elseif ask == 24 then
      return takeHtml(player, qs, 6)
    elseif ask == 25 then
      return takeHtml(player, qs, 7)
    elseif ask == 26 then
      return takeHtml(player, qs, 8)
    elseif ask == 27 then
      return takeHtml(player, qs, 9)
    elseif ask == 55 then
      return beforeReward(player, qs, 1)
    elseif ask == 56 then
      return beforeReward(player, qs, 2)
    elseif ask == 57 then
      return beforeReward(player, qs, 3)
    elseif ask == 58 then
      return beforeReward(player, qs, 4)
    elseif ask == 59 then
      return beforeReward(player, qs, 5)
    elseif ask == 60 then
      return beforeReward(player, qs, 6)
    elseif ask == 61 then
      return beforeReward(player, qs, 7)
    elseif ask == 62 then
      return beforeReward(player, qs, 8)
    elseif ask == 63 then
      return beforeReward(player, qs, 9)
    end
  end
  return nil
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = getRandomPlayerFromParty(killer, npc)
  if qs ~= nil then
    local sw = npc:id()
    if sw == CRIMSON_DRAKE then
      if rnd(1000) < 20.200001 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == KADIOS then
      if rnd(1000) < 211 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == HUNGRY_CORPSE then
      if rnd(1000) < 184 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == PAST_KNIGHT then
      if rnd(1000) < 216 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == BLADE_DEATH then
      if rnd(100) < 17 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == DARK_GUARD then
      if rnd(1000) < 273 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == BLOODY_GHOST then
      if rnd(1000) < 149 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == BLOODY_LORD then
      if rnd(1000) < 199 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == PAST_CREATURE then
      if rnd(1000) < 257 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == GIANT_SHADOW then
      if rnd(1000) < 205 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == ANCIENTS_SOLDIER then
      if rnd(1000) < 208 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == ANCIENTS_WARRIOR then
      if rnd(1000) < 299 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == SPITE_SOUL_LEADER then
      if rnd(100) < 44 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == SPITE_SOUL_WIZARD then
      if rnd(100) < 39 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == WRECKED_ARCHER then
      if rnd(1000) < 214 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == FLOAT_OF_GRAVE then
      if rnd(1000) < 173 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == GRAVE_PREDATOR then
      if rnd(1000) < 211 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == FALLEN_ORC_SHAMAN then
      if rnd(1000) < 478 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == SHARP_TALON_TIGER then
      if rnd(1000) < 234 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == GLOW_WISP then
      if rnd(1000) < 245 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == MARSH_PREDATOR then
      if rnd(100) < 26 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == HAMES_ORC_SNIPER then
      if rnd(100) < 37 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == CURSED_GUARDIAN then
      if rnd(1000) < 352 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    elseif sw == HAMES_ORC_CHIEFTAIN or sw == FALLEN_ORC_SHAMAN_TRANS or sw == SHARP_TALON_TIGER_TRANS then
      if rnd(1000) < 487 then
        q:give_item_randomly(qs:get_player(), npc, Q_STOLEN_INF_ORE, 1, 1, 0, 1, true)
      end
    end
  end
end)
