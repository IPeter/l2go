-- Translated by questport from quests/Q00384_WarehouseKeepersPastime (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Warehouse Keeper's Pastime (384)
-- @author Zealar
-- @since 2.6.0.0
--
-- Checked by hand: the bingo board lives in the quest variables "numbers"
-- and "selected" (space separated, as Mobius stores them); the pages with
-- the board are read with q:get_htm and filled here. Collections.shuffle
-- draws from the quest random source (J.shuffle).
local J = require("lib.java")

local q = quest.new(384, "Q00384_WarehouseKeepersPastime", "Warehouse Keeper's Pastime")

-- NPCs
local CLIFF = 30182
local WAREHOUSE_CHIEF_BAXT = 30685
-- Monsters
local DUST_WIND = 20242
local INNERSEN = 20950
local CONGERER = 20774
local CARINKAIN = 20635
local CONNABI = 20947
local HUNTER_GARGOYLE = 20241
local NIGHTMARE_GUIDE = 20942
local DRAGON_BEARER_WARRIOR = 20759
local DRAGON_BEARER_CHIEF = 20758
local DUST_WIND_HOLD = 20281
local WEIRD_DRAKE = 20605
local THUNDER_WYRM_HOLD = 20282
local CADEINE = 20945
local CONGERER_LORD = 20773
local DRAGON_BEARER_ARCHER = 20760
local NIGHTMARE_LORD = 20944
local SANHIDRO = 20946
local GIANT_MONSTEREYE = 20556
local BARTAL = 20948
local HUNTER_GARGOYLE_HOLD = 20286
local ROT_GOLEM = 20559
local GRAVE_GUARD = 20668
local TULBEN = 20677
local NIGHTMARE_KEEPER = 20943
local LUMINUN = 20949
local THUNDER_WYRM = 20243
-- Items
local Q_IRONGATE_MEDAL = 5964
-- Reward
local MOONSTONE_EARING = 852
local DRAKE_LEATHER_BOOTS = 2437
local DRAKE_LEATHER_MAIL = 401
local MOLD_HARDENER = 4041
local NECKLACE_OF_MERMAID = 917
local SCRL_OF_ENCH_AM_C = 952
local BLACKSMITH_S_FRAME = 1892
local SCRL_OF_ENCH_WP_C = 951
local ORIHARUKON = 1893
local SAMURAI_LONGSWORD = 135
local AQUASTONE_RING = 883
local SYNTHESIS_COKES = 1888
local MITHIRL_ALLOY = 1890
local GREAT_HELMET = 500
local VARNISH_OF_PURITY = 1887
local BLESSED_GLOVES = 2463
local CRAFTED_LEATHER = 1894

local takeHtml, fillBoard, colorBoard, beforeReward, reward, createBingoBoard, getMatchedBingoLineCount, selectBingoNumber
local isSelectedBingoNumber, getNumberFromBingoBoard, getBingoSelectCount, getRandomPlayerFromParty

function takeHtml(player, qs, num, npcId)
  local html_ = nil
  local i3
  if not isSelectedBingoNumber(qs, num) then
    selectBingoNumber(qs, num)
    i3 = getBingoSelectCount(qs)
    if i3 == 2 then
      html_ = q:get_htm(player, npcId .. "-14.html")
    elseif i3 == 3 then
      html_ = q:get_htm(player, npcId .. "-16.html")
    elseif i3 == 4 then
      html_ = q:get_htm(player, npcId .. "-18.html")
    elseif i3 == 5 then
      html_ = q:get_htm(player, npcId .. "-20.html")
    end
    return fillBoard(qs, html_)
  end
  i3 = getBingoSelectCount(qs)
  if i3 == 1 then
    html_ = q:get_htm(player, npcId .. "-15.html")
  elseif i3 == 2 then
    html_ = q:get_htm(player, npcId .. "-17.html")
  elseif i3 == 3 then
    html_ = q:get_htm(player, npcId .. "-19.html")
  elseif i3 == 4 then
    html_ = q:get_htm(player, npcId .. "-21.html")
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

function beforeReward(player, qs, num, npcId)
  if not isSelectedBingoNumber(qs, num) then
    selectBingoNumber(qs, num)
    local i3 = getMatchedBingoLineCount(qs)
    local html_
    if i3 == 3 and getBingoSelectCount(qs) == 6 then
      reward(player, qs, i3)
      html_ = q:get_htm(player, npcId .. "-22.html")
    elseif i3 == 0 and getBingoSelectCount(qs) == 6 then
      reward(player, qs, i3)
      html_ = q:get_htm(player, npcId .. "-24.html")
    else
      html_ = q:get_htm(player, npcId .. "-23.html")
    end
    return colorBoard(qs, html_)
  end
  return fillBoard(qs, q:get_htm(player, npcId .. "-25.html"))
end

function reward(player, qs, i3)
  if i3 == 3 then
    if qs:get_memo_state() == 10 then
      local random = rnd(100)
      if random < 16 then
        q:give_items(player, SYNTHESIS_COKES, 1)
      elseif random < 32 then
        q:give_items(player, VARNISH_OF_PURITY, 1)
      elseif random < 50 then
        q:give_items(player, CRAFTED_LEATHER, 1)
      elseif random < 80 then
        q:give_items(player, SCRL_OF_ENCH_AM_C, 1)
      elseif random < 89 then
        q:give_items(player, MITHIRL_ALLOY, 1)
      elseif random < 98 then
        q:give_items(player, ORIHARUKON, 1)
      else
        q:give_items(player, SCRL_OF_ENCH_WP_C, 1)
      end
    elseif qs:get_memo_state() == 20 then
      local random = rnd(100)
      if random < 50 then
        q:give_items(player, AQUASTONE_RING, 1)
      elseif random < 80 then
        q:give_items(player, SCRL_OF_ENCH_WP_C, 1)
      elseif random < 98 then
        q:give_items(player, MOONSTONE_EARING, 1)
      else
        q:give_items(player, DRAKE_LEATHER_MAIL, 1)
      end
    end
  elseif i3 == 0 then
    if qs:get_memo_state() == 10 then
      local random = rnd(100)
      if random < 50 then
        q:give_items(player, MOLD_HARDENER, 1)
      elseif random < 80 then
        q:give_items(player, SCRL_OF_ENCH_AM_C, 1)
      elseif random < 98 then
        q:give_items(player, BLACKSMITH_S_FRAME, 1)
      else
        q:give_items(player, NECKLACE_OF_MERMAID, 1)
      end
    elseif qs:get_memo_state() == 20 then
      local random = rnd(100)
      if random < 50 then
        q:give_items(player, SCRL_OF_ENCH_WP_C, 1)
      elseif random < 80 then
        q:give_items(player, GREAT_HELMET, 1)
      elseif random < 98 then
        q:give_items(player, DRAKE_LEATHER_BOOTS, 1)
        q:give_items(player, BLESSED_GLOVES, 1)
      else
        q:give_items(player, SAMURAI_LONGSWORD, 1)
      end
    end
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
  local current = qs:get("selected")
  return #(string.gsub(current, "%D", ""))
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

q:add_start_npc(CLIFF)
q:add_talk_id(CLIFF, WAREHOUSE_CHIEF_BAXT)
q:add_kill_id(WAREHOUSE_CHIEF_BAXT, DUST_WIND, INNERSEN, CLIFF, CONGERER, CARINKAIN, CONNABI, HUNTER_GARGOYLE, NIGHTMARE_GUIDE, DRAGON_BEARER_WARRIOR, DRAGON_BEARER_CHIEF, DUST_WIND_HOLD, WEIRD_DRAKE, THUNDER_WYRM_HOLD, CADEINE, CONGERER_LORD, DRAGON_BEARER_ARCHER, NIGHTMARE_LORD, SANHIDRO, GIANT_MONSTEREYE, BARTAL, HUNTER_GARGOYLE_HOLD, ROT_GOLEM, GRAVE_GUARD, TULBEN, NIGHTMARE_KEEPER, LUMINUN, THUNDER_WYRM)
q:register_quest_items(Q_IRONGATE_MEDAL)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = npc:id()
  if sw == CLIFF then
    if qs:is_created() then
      if player:level() >= 40 then
        return "30182-01.htm"
      end
      return "30182-04.html"
    end
    if q:get_quest_items_count(player, Q_IRONGATE_MEDAL) < 10 then
      return "30182-06.html"
    end
    return "30182-07.html"
  elseif sw == WAREHOUSE_CHIEF_BAXT then
    if qs:get_memo_state() > 0 then
      if q:get_quest_items_count(player, Q_IRONGATE_MEDAL) < 10 then
        return "30685-06.html"
      end
      return "30685-07.html"
    end
  end
  return htmltext
end)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs ~= nil then
    if J.str_contains(event, ".htm") then
      return event
    end
    local sw = npc:id()
    if sw == CLIFF then
      if event == "QUEST_ACCEPTED" then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ACCEPT)
        qs:set_memo_state(384)
        qs:start_quest()
        qs:show_question_mark(384)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        return "30182-05.htm"
      end
      local ask = J.parse_int(event)
      if ask == 3 then
        return "30182-09a.htm"
      elseif ask == 4 then
        return "30182-02.htm"
      elseif ask == 5 then
        if npc:id() ~= CLIFF then
          return q:get_no_quest_msg(player)
        end
        return "30182-03.htm"
      elseif ask == 6 then
        qs:exit_quest(true)
        return "30182-08.html"
      elseif ask == 9 then
        return "30182-09.html"
      elseif ask == 7 then
        if q:get_quest_items_count(player, Q_IRONGATE_MEDAL) >= 10 then
          q:take_items(player, Q_IRONGATE_MEDAL, 10)
          qs:set_memo_state(10)
          createBingoBoard(qs)
          return "30182-10.html"
        end
        return "30182-11.html"
      elseif ask == 8 then
        if q:get_quest_items_count(player, Q_IRONGATE_MEDAL) >= 100 then
          q:take_items(player, Q_IRONGATE_MEDAL, 100)
          qs:set_memo_state(20)
          createBingoBoard(qs)
          return "30182-12.html"
        end
        return "30182-11.html"
      elseif ask == 10 or ask == 11 or ask == 12 or ask == 13 or ask == 14 or ask == 15 or ask == 16 or ask == 17 or ask == 18 then
        selectBingoNumber(qs, ask - 10 + 1)
        return fillBoard(qs, q:get_htm(player, "30182-13.html"))
      elseif ask == 19 or ask == 20 or ask == 21 or ask == 22 or ask == 23 or ask == 24 or ask == 25 or ask == 26 or ask == 27 then
        return takeHtml(player, qs, ask - 18, CLIFF)
      elseif ask == 55 or ask == 56 or ask == 57 or ask == 58 or ask == 59 or ask == 60 or ask == 61 or ask == 62 or ask == 63 then
        return beforeReward(player, qs, ask - 54, CLIFF)
      end
    elseif sw == WAREHOUSE_CHIEF_BAXT then
      local ask = J.parse_int(event)
      if ask == 3 then
        return "30685-09a.html"
      elseif ask == 6 then
        qs:exit_quest(true)
        return "30685-08.html"
      elseif ask == 9 then
        return "30685-09.html"
      elseif ask == 7 then
        if q:get_quest_items_count(player, Q_IRONGATE_MEDAL) >= 10 then
          q:take_items(player, Q_IRONGATE_MEDAL, 10)
          qs:set_memo_state(10)
          createBingoBoard(qs)
          return "30685-10.html"
        end
        return "30685-11.html"
      elseif ask == 8 then
        if q:get_quest_items_count(player, Q_IRONGATE_MEDAL) >= 100 then
          q:take_items(player, Q_IRONGATE_MEDAL, 100)
          qs:set_memo_state(20)
          createBingoBoard(qs)
          return "30685-12.html"
        end
        return "30685-11.html"
      elseif ask == 10 or ask == 11 or ask == 12 or ask == 13 or ask == 14 or ask == 15 or ask == 16 or ask == 17 or ask == 18 then
        selectBingoNumber(qs, ask - 9)
        return fillBoard(qs, q:get_htm(player, "30685-13.html"))
      elseif ask == 19 or ask == 20 or ask == 21 or ask == 22 or ask == 23 or ask == 24 or ask == 25 or ask == 26 or ask == 27 then
        return takeHtml(player, qs, ask - 18, WAREHOUSE_CHIEF_BAXT)
      elseif ask == 55 or ask == 56 or ask == 57 or ask == 58 or ask == 59 or ask == 60 or ask == 61 or ask == 62 or ask == 63 then
        return beforeReward(player, qs, ask - 54, WAREHOUSE_CHIEF_BAXT)
      end
    end
  end
  return nil
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = getRandomPlayerFromParty(killer, npc)
  if qs ~= nil then
    local sw = npc:id()
    if sw == HUNTER_GARGOYLE then
      if rnd(1000) < 328 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == DUST_WIND then
      if rnd(100) < 35 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == THUNDER_WYRM then
      if rnd(1000) < 312 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == DUST_WIND_HOLD then
      if rnd(100) < 35 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == THUNDER_WYRM_HOLD then
      if rnd(1000) < 312 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == HUNTER_GARGOYLE_HOLD then
      if rnd(1000) < 328 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == GIANT_MONSTEREYE then
      if rnd(1000) < 176 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == ROT_GOLEM then
      if rnd(1000) < 226 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == WEIRD_DRAKE then
      if rnd(1000) < 218 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == CARINKAIN then
      if rnd(1000) < 216 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == GRAVE_GUARD then
      if rnd(1000) < 312 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == TULBEN then
      if rnd(1000) < 522 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == DRAGON_BEARER_CHIEF then
      if rnd(100) < 38 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == DRAGON_BEARER_WARRIOR then
      if rnd(100) < 39 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == DRAGON_BEARER_ARCHER then
      if rnd(1000) < 372 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == CONGERER_LORD then
      if rnd(1000) < 802 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == CONGERER then
      if rnd(1000) < 844 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == NIGHTMARE_GUIDE then
      if rnd(1000) < 118 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == NIGHTMARE_KEEPER then
      if rnd(100) < 17 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == NIGHTMARE_LORD then
      if rnd(1000) < 144 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == CADEINE then
      if rnd(1000) < 162 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == SANHIDRO then
      if rnd(100) < 25 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == CONNABI then
      if rnd(1000) < 272 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == BARTAL then
      if rnd(100) < 27 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == LUMINUN then
      if rnd(100) < 32 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    elseif sw == INNERSEN then
      if rnd(1000) < 346 then
        q:give_item_randomly(qs:get_player(), npc, Q_IRONGATE_MEDAL, 1, 1, 0, 1, true)
      end
    end
  end
end)
