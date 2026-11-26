-- The Seven Signs priests: Mobius entity/actor/instance/SignsPriest,
-- DawnPriest and DuskPriest. The Dawn and Dusk priests' first page follows
-- the period and the player's cabal (showChatWindow); every SignsPriest
-- (the priests and the Mammon NPCs) runs the "SevenSigns <n> ..." links of
-- its pages, which the game sends here as events: buying the Record of the
-- Seven Signs, joining a cabal, contributing seal stones, exchanging stones
-- and ancient adena, the contribution reward, and the seal pages. The pages
-- are Mobius's, in data/html/seven_signs/.

local J = require("lib.java")

local s = script.new("SignsPriest")

local PATH = "data/html/seven_signs/"

local CABAL_NULL, CABAL_DUSK, CABAL_DAWN = 0, 1, 2
local SEAL_AVARICE, SEAL_GNOSIS, SEAL_STRIFE = 1, 2, 3
local PERIOD_RECRUITING = 0

local RECORD_ID, RECORD_COST = 5707, 500
local ANCIENT_ADENA = 5575
local ADENA = 57
local BLUE, GREEN, RED = 6360, 6361, 6362
local BLUE_VALUE, GREEN_VALUE, RED_VALUE = 3, 5, 10
local BLUE_POINTS, GREEN_POINTS, RED_POINTS = 3, 5, 10

local DAWN_PRIESTS = { 31078, 31079, 31080, 31081, 31082, 31083, 31084, 31168, 31692, 31694, 31997 }
local DUSK_PRIESTS = { 31085, 31086, 31087, 31088, 31089, 31090, 31091, 31169, 31693, 31695, 31998 }

local function short_name(cabal)
  if cabal == CABAL_DAWN then
    return "dawn"
  elseif cabal == CABAL_DUSK then
    return "dusk"
  end
  return "No Cabal"
end

local function cabal_name(cabal)
  if cabal == CABAL_DAWN then
    return "Lords of Dawn"
  elseif cabal == CABAL_DUSK then
    return "Revolutionaries of Dusk"
  end
  return "No Cabal"
end

local SEAL_SHORT = { [SEAL_AVARICE] = "Avarice", [SEAL_GNOSIS] = "Gnosis", [SEAL_STRIFE] = "Strife" }

local function seal_name(seal, short)
  local name = short and "" or "Seal of "
  return name .. (SEAL_SHORT[seal] or "")
end

-- page is SignsPriest.showChatWindow(player, value, suffix, isDescription).
local function page(value, suffix, desc)
  local name = PATH .. (desc and "desc_" or "signs_") .. value
  if suffix ~= nil then
    name = name .. "_" .. suffix
  end
  return name .. ".htm"
end

local function is_dawn(npc)
  return npc:is_type("DawnPriest")
end

-- side is "dawn" for a Dawn priest, "dusk" for any other SignsPriest (Mobius
-- tests instanceof DawnPriest).
local function side(npc)
  return is_dawn(npc) and "dawn" or "dusk"
end

local function class_level(player)
  return ClassId.level(player:class_id())
end

local function count(player, id)
  return player:item_count(id)
end

-- The Dawn and Dusk priests' first page (DawnPriest/DuskPriest.showChatWindow).
local function first_page(npc, player)
  local dawn = is_dawn(npc)
  local prefix = dawn and "dawn_priest_" or "dusk_priest_"
  local own = dawn and CABAL_DAWN or CABAL_DUSK
  local rival = dawn and CABAL_DUSK or CABAL_DAWN
  local gnosis = sevensigns.seal_owner(SEAL_GNOSIS)
  local cabal = sevensigns.cabal(player)
  local period = sevensigns.period()
  local validation = period == 3
  local results = period == 2
  local winner = sevensigns.highest_cabal()
  local file
  if cabal == own then
    if results then
      file = "5"
    elseif period == PERIOD_RECRUITING then
      file = "6"
    elseif validation then
      if winner == own then
        file = winner ~= gnosis and "2c" or "2a"
      elseif winner == CABAL_NULL then
        file = "2d"
      else
        file = "2b"
      end
    else
      file = "1b"
    end
  elseif cabal == rival then
    file = validation and "3a" or "3b"
  else
    if results then
      file = "5"
    elseif period == PERIOD_RECRUITING then
      file = "6"
    elseif validation then
      if winner == own then
        file = "4"
      elseif winner == CABAL_NULL then
        file = "2d"
      else
        file = "2b"
      end
    else
      file = "1a"
    end
  end
  return PATH .. prefix .. file .. ".htm"
end

s:on_first_talk(DAWN_PRIESTS, first_page)
s:on_first_talk(DUSK_PRIESTS, first_page)

local function parse(text)
  if text == nil then
    return nil
  end
  local ok, v = pcall(J.parse_int, J.trim(text))
  if ok then
    return v
  end
  return nil
end

local function tokens(event)
  local out = {}
  for word in event:gmatch("%S+") do
    out[#out + 1] = word
  end
  return out
end

-- The stone colours of the contribution and exchange pages.
local STONES = {
  [1] = { color = "Blue", lower = "blue", id = BLUE, value = BLUE_VALUE },
  [2] = { color = "Green", lower = "green", id = GREEN, value = GREEN_VALUE },
  [3] = { color = "Red", lower = "red", id = RED, value = RED_VALUE },
}

-- take_stones destroys stones and tells the player, as SignsPriest does
-- (destroyItemByItemId without its message, then "$s2 $s1 has disappeared.").
local function take_stones(player, id, n)
  if n > 0 and sevensigns.destroy_item(player, id, n, false) then
    sevensigns.send_disappeared(player, id, n)
    return true
  end
  return false
end

local handlers = {}

-- 2: buy the Record of the Seven Signs.
handlers[2] = function(npc, player, val)
  if not sevensigns.has_room(player, 1) then
    player:send_system_message(SystemMessageId.YOUR_INVENTORY_IS_FULL)
    return ""
  end
  if not sevensigns.reduce_adena(player, RECORD_COST) then
    player:send_system_message(SystemMessageId.YOU_DO_NOT_HAVE_ENOUGH_ADENA)
    return ""
  end
  sevensigns.add_item(player, RECORD_ID, 1, true)
  return page(val, side(npc))
end

-- 33: "I want to participate".
handlers[33] = function(npc, player, val, cabal)
  local cfg = sevensigns.config()
  if sevensigns.cabal(player) ~= CABAL_NULL then
    return page(val, side(npc) .. "_member")
  elseif class_level(player) == 0 then
    return page(val, side(npc) .. "_firstclass")
  elseif cabal == CABAL_DUSK and cfg.castle_for_dusk then
    if sevensigns.clan_castle(player) > 0 then
      return PATH .. "signs_33_dusk_no.htm"
    end
  elseif cabal == CABAL_DAWN and cfg.castle_for_dawn and class_level(player) >= 2 then
    if sevensigns.clan_castle(player) == 0 then
      return PATH .. "signs_33_dawn_fee.htm"
    end
  end
  return page(val, side(npc))
end

-- 34: pay the participation fee.
handlers[34] = function(npc, player)
  local cfg = sevensigns.config()
  if class_level(player) > 0 and (player:adena() >= cfg.join_dawn_fee or count(player, cfg.manors_agreement_id) > 0) then
    return PATH .. "signs_33_dawn.htm"
  end
  return PATH .. "signs_33_dawn_no.htm"
end

-- 3 and 8: the cabal's introduction and the festival's.
local function intro(npc, player, val, cabal)
  return page(val, short_name(cabal))
end
handlers[3] = intro
handlers[8] = intro

-- 4: join a cabal for a seal.
handlers[4] = function(npc, player, val, cabal, args)
  local seal = parse(args[4])
  if seal == nil then
    return ""
  end
  local mine = is_dawn(npc) and CABAL_DAWN or CABAL_DUSK
  if class_level(player) == 0 then
    return page(33, side(npc) .. "_firstclass")
  end
  if seal < SEAL_AVARICE or seal > SEAL_STRIFE or cabal ~= mine then
    return ""
  end
  if sevensigns.period() ~= 1 then
    -- Mobius showChatWindow(player): the priests' first page.
    if npc:is_type("DawnPriest") or npc:is_type("DuskPriest") then
      return first_page(npc, player)
    end
    return ""
  end
  if sevensigns.cabal(player) ~= CABAL_NULL then
    return page(33, side(npc) .. "_member")
  end
  local cfg = sevensigns.config()
  if class_level(player) >= 1 then
    local castle = sevensigns.clan_castle(player)
    if cabal == CABAL_DUSK and cfg.castle_for_dusk and castle > 0 then
      return PATH .. "signs_33_dusk_no.htm"
    end
    if cfg.castle_for_dawn and cabal == CABAL_DAWN and class_level(player) >= 2 then
      local allowed = castle > 0
        or sevensigns.destroy_item(player, cfg.manors_agreement_id, 1, true)
        or sevensigns.reduce_adena(player, cfg.join_dawn_fee)
      if not allowed then
        return PATH .. "signs_33_dawn_fee.htm"
      end
    end
  end
  sevensigns.join(player, cabal, seal)
  if cabal == CABAL_DAWN then
    player:send_system_message(SystemMessageId.YOU_WILL_PARTICIPATE_IN_THE_SEVEN_SIGNS_AS_A_MEMBER_OF_THE_LORDS_OF_DAWN)
  else
    player:send_system_message(SystemMessageId.YOU_WILL_PARTICIPATE_IN_THE_SEVEN_SIGNS_AS_A_MEMBER_OF_THE_REVOLUTIONARIES_OF_DUSK)
  end
  if seal == SEAL_AVARICE then
    player:send_system_message(SystemMessageId.YOU_VE_CHOSEN_TO_FIGHT_FOR_THE_SEAL_OF_AVARICE_DURING_THIS_QUEST_EVENT_PERIOD)
  elseif seal == SEAL_GNOSIS then
    player:send_system_message(SystemMessageId.YOU_VE_CHOSEN_TO_FIGHT_FOR_THE_SEAL_OF_GNOSIS_DURING_THIS_QUEST_EVENT_PERIOD)
  else
    player:send_system_message(SystemMessageId.YOU_VE_CHOSEN_TO_FIGHT_FOR_THE_SEAL_OF_STRIFE_DURING_THIS_QUEST_EVENT_PERIOD)
  end
  return page(4, short_name(cabal))
end

-- 5: the contribution page.
handlers[5] = function(npc, player, val)
  if sevensigns.cabal(player) == CABAL_NULL then
    return page(val, side(npc) .. "_no")
  end
  return page(val, side(npc))
end

local function contributed(npc, player, blue, green, red)
  local score = sevensigns.contribute(player, blue, green, red)
  player:send_system_message(SystemMessageId.YOUR_CONTRIBUTION_SCORE_IS_INCREASED_BY_S1, score)
  return page(6, side(npc))
end

-- 21: contribute a number of stones of one colour.
handlers[21] = function(npc, player, val, cabal, args)
  local stone = parse(args[3])
  local n = parse(args[4])
  local score = sevensigns.contribution(player)
  if n == nil or stone == nil then
    return page(6, side(npc) .. "_failure")
  end
  local max = sevensigns.config().max_player_contrib
  local blue, green, red = 0, 0, 0
  if stone == BLUE then
    blue = math.min(n, J.idiv(max - score, BLUE_POINTS))
  elseif stone == GREEN then
    green = math.min(n, J.idiv(max - score, GREEN_POINTS))
  elseif stone == RED then
    red = math.min(n, J.idiv(max - score, RED_POINTS))
  end
  local found = false
  if red > 0 and take_stones(player, RED, red) then
    found = true
  end
  if green > 0 and take_stones(player, GREEN, green) then
    found = true
  end
  if blue > 0 and take_stones(player, BLUE, blue) then
    found = true
  end
  if n > 0 and red + green + blue <= 0 then
    player:send_system_message(SystemMessageId.CONTRIBUTION_LEVEL_HAS_EXCEEDED_THE_LIMIT_YOU_MAY_NOT_CONTINUE)
    return ""
  elseif not found then
    return page(6, side(npc) .. "_low_stones")
  end
  return contributed(npc, player, blue, green, red)
end

-- 6: the contribution of one colour (the amount page), or of all stones.
handlers[6] = function(npc, player, val, cabal, args)
  local kind = parse(args[3])
  local score = sevensigns.contribution(player)
  local max = sevensigns.config().max_player_contrib
  if score >= max then
    player:send_system_message(SystemMessageId.CONTRIBUTION_LEVEL_HAS_EXCEEDED_THE_LIMIT_YOU_MAY_NOT_CONTINUE)
    return ""
  end
  local blues, greens, reds = count(player, BLUE), count(player, GREEN), count(player, RED)
  if kind == 4 then
    local tmp = score
    local red = math.min(J.idiv(max - tmp, RED_POINTS), reds)
    tmp = tmp + red * RED_POINTS
    local green = math.min(J.idiv(max - tmp, GREEN_POINTS), greens)
    tmp = tmp + green * GREEN_POINTS
    local blue = math.min(J.idiv(max - tmp, BLUE_POINTS), blues)
    local found = false
    if red > 0 and take_stones(player, RED, red) then
      found = true
    end
    if green > 0 and take_stones(player, GREEN, green) then
      found = true
    end
    if blue > 0 and take_stones(player, BLUE, blue) then
      found = true
    end
    if not found and reds + greens + blues > 0 then
      player:send_system_message(SystemMessageId.CONTRIBUTION_LEVEL_HAS_EXCEEDED_THE_LIMIT_YOU_MAY_NOT_CONTINUE)
      return ""
    elseif not found then
      return page(val, side(npc) .. "_no_stones")
    end
    return contributed(npc, player, blue, green, red)
  end
  local st = STONES[kind]
  if st == nil then
    return ""
  end
  local n = ({ blues, greens, reds })[kind]
  return PATH .. "signs_6_" .. side(npc) .. "_contribute.htm",
    { contribStoneColor = st.color, stoneColor = st.lower, stoneCount = tostring(n), stoneItemId = tostring(st.id) }
end

-- 7: ancient adena for adena.
handlers[7] = function(npc, player, val, cabal, args)
  local n = parse(args[3])
  if n == nil or n < 1 then
    return PATH .. "blkmrkt_3.htm"
  end
  if count(player, ANCIENT_ADENA) < n then
    return PATH .. "blkmrkt_4.htm"
  end
  sevensigns.destroy_item(player, ANCIENT_ADENA, n, true)
  sevensigns.add_item(player, ADENA, n, true)
  return PATH .. "blkmrkt_5.htm"
end

-- 9: the contribution reward of the winners.
handlers[9] = function(npc, player)
  if sevensigns.period() == 3 and sevensigns.cabal(player) == sevensigns.highest_cabal() then
    local reward = sevensigns.take_reward(player)
    if reward < 3 then
      return page(9, side(npc) .. "_b")
    end
    sevensigns.add_item(player, ANCIENT_ADENA, reward, true)
    return page(9, side(npc) .. "_a")
  end
  return ""
end

-- 11: a teleport to the hunting grounds for ancient adena.
handlers[11] = function(npc, player, val, cabal, args)
  local x, y, z, cost = parse(args[3]), parse(args[4]), parse(args[5]), parse(args[6])
  if x == nil or y == nil or z == nil or cost == nil then
    log.warn("SevenSigns: bad teleport link", "event", table.concat(args, " "))
    return ""
  end
  if cost > 0 and not sevensigns.destroy_item(player, ANCIENT_ADENA, cost, true) then
    return ""
  end
  player:teleport(x, y, z)
  return ""
end

handlers[16] = function(npc, player, val)
  return page(val, side(npc))
end

-- 17: the exchange page of one colour, or all stones at once.
handlers[17] = function(npc, player, val, cabal, args)
  local kind = parse(args[3])
  if kind == 4 then
    local blues, greens, reds = count(player, BLUE), count(player, GREEN), count(player, RED)
    local reward = blues * BLUE_VALUE + greens * GREEN_VALUE + reds * RED_VALUE
    if reward == 0 then
      return page(18, side(npc) .. "_no_stones")
    end
    if blues > 0 then
      sevensigns.destroy_item(player, BLUE, blues, true)
    end
    if greens > 0 then
      sevensigns.destroy_item(player, GREEN, greens, true)
    end
    if reds > 0 then
      sevensigns.destroy_item(player, RED, reds, true)
    end
    sevensigns.add_item(player, ANCIENT_ADENA, reward, true)
    return page(18, side(npc))
  end
  local st = STONES[kind]
  if st == nil then
    return ""
  end
  return PATH .. "signs_17_" .. side(npc) .. ".htm",
    { stoneColor = st.lower, stoneValue = tostring(st.value), stoneCount = tostring(count(player, st.id)), stoneItemId = tostring(st.id) }
end

-- 18: exchange a number of stones of one colour.
handlers[18] = function(npc, player, val, cabal, args)
  local stone = parse(args[3])
  local n = parse(args[4])
  if n == nil or stone == nil then
    return page(18, side(npc) .. "_failed")
  end
  local have = count(player, stone)
  if have == 0 then
    return page(18, side(npc) .. "_no_stones")
  end
  if n > have or n <= 0 then
    return page(18, side(npc) .. "_low_stones")
  end
  local value = ({ [BLUE] = BLUE_VALUE, [GREEN] = GREEN_VALUE, [RED] = RED_VALUE })[stone] or 0
  if sevensigns.destroy_item(player, stone, n, true) then
    sevensigns.add_item(player, ANCIENT_ADENA, n * value, true)
    return page(18, side(npc))
  end
  return ""
end

-- 19: a seal's description when joining.
handlers[19] = function(npc, player, val, cabal, args)
  local seal = parse(args[4])
  if seal == nil then
    return ""
  end
  return page(val, seal_name(seal, true) .. "_" .. short_name(cabal))
end

-- 20: the seals' owners.
handlers[20] = function(npc, player)
  local b = {}
  if is_dawn(npc) then
    b[#b + 1] = '<html><body>Priest of Dawn:<br><font color="LEVEL">[ Seal Status ]</font><br>'
  else
    b[#b + 1] = '<html><body>Dusk Priestess:<br><font color="LEVEL">[ Status of the Seals ]</font><br>'
  end
  for seal = 1, 3 do
    local owner = sevensigns.seal_owner(seal)
    if owner ~= CABAL_NULL then
      b[#b + 1] = "[" .. seal_name(seal, false) .. ": " .. cabal_name(owner) .. "]<br>"
    else
      b[#b + 1] = "[" .. seal_name(seal, false) .. ": Nothingness]<br>"
    end
  end
  b[#b + 1] = '<a action="bypass -h npc_' .. npc:object_id() .. '_Chat 0">Go back.</a></body></html>'
  return table.concat(b)
end

s:on_event(function(event, npc, player)
  if npc == nil or player == nil then
    return nil
  end
  local args = tokens(event)
  if args[1] == "SevenSignsDesc" then
    local val = parse(args[2])
    if val == nil then
      return ""
    end
    return page(val, nil, true)
  end
  if args[1] ~= "SevenSigns" then
    return nil
  end
  local val = parse(args[2])
  if val == nil then
    return ""
  end
  local cabal = parse(args[3]) or CABAL_NULL
  local handler = handlers[val]
  if handler == nil then
    return page(val, nil)
  end
  return handler(npc, player, val, cabal, args)
end)
