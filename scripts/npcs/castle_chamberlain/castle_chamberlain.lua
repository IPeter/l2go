-- The castle chamberlains, ported from Mobius ai/others/CastleChamberlain.
--
-- The lord (and the members of the owner clan with the right privileges)
-- reads the report, sets the tax within the Seal of Strife's limit, moves
-- adena between the castle vault and the inventory, opens and closes the
-- castle gates, banishes foreigners, sees the siege window and buys the
-- castle's general items; everyone else gets chamberlain-04.html.
--
-- What the castle core does not have yet answers with Mobius's page that
-- fits or a short "not available" message (marked NOT BUILT below): the
-- castle functions (renting, teleports, buffs), trap upgrades and the
-- Seven Signs (M11l: no seal owner, the seal validation period, so the
-- dawn ticket sale never opens). The manor page's links open the game's
-- manor windows (M11k).
--
-- The pages are Mobius's, served from
-- data/html/scripts/ai/others/CastleChamberlain/.

local J = require("lib.java")

local s = script.new("CastleChamberlain")

local NPCS = {
  35100, -- Sayres
  35142, -- Crosby
  35184, -- Saul
  35226, -- Brasseur
  35274, -- Logan
  35316, -- Neurath
  35363, -- Alfred
  35509, -- Frederick
  35555, -- August
}

local DIR = "data/html/scripts/ai/others/CastleChamberlain/"

local CROWN = 6841 -- Lord's Crown
local MAX_ADENA = 2000000000 -- Inventory.MAX_ADENA (Mobius MaxAdena)

-- The castle functions (Mobius Castle.FUNC_*).
local FUNC_TELEPORT = 1
local FUNC_RESTORE_HP = 2
local FUNC_RESTORE_MP = 3
local FUNC_RESTORE_EXP = 4
local FUNC_SUPPORT = 5

-- Seven Signs (Mobius SevenSigns).
local CABAL_NULL = 0
local CABAL_DUSK = 1
local CABAL_DAWN = 2
local SEAL_AVARICE = 1
local SEAL_GNOSIS = 2
local SEAL_STRIFE = 3
local PERIOD_COMP_RECRUITING = 0
local PERIOD_COMPETITION = 1
local PERIOD_COMP_RESULTS = 2

-- The answer for what the castle core does not have yet.
local NOT_AVAILABLE = "This function is not available yet."

-- FeatureConfig's defaults (Feature.ini): the function fees by level, their
-- payment interval (7 days in ms), and the door and trap upgrade prices.
local DAY_MS = 86400000
local FEE_RATIO = 604800000
local FUNCTION_FEES = {
  [FUNC_RESTORE_EXP] = { level = 45, first = 63000, other = 70000 },
  [FUNC_RESTORE_HP] = { level = 300, first = 12000, other = 20000 },
  [FUNC_RESTORE_MP] = { level = 40, first = 45000, other = 65000 },
  [FUNC_SUPPORT] = { level = 5, first = 49000, other = 120000 },
  [FUNC_TELEPORT] = { level = 1, first = 1000, other = 10000 },
}
local DOOR_PRICES = {
  [1] = { [2] = 3000000, [3] = 4000000, [5] = 5000000 }, -- outer door
  [2] = { [2] = 750000, [3] = 900000, [5] = 1000000 }, -- inner door
  [3] = { [2] = 1600000, [3] = 1800000, [5] = 2000000 }, -- wall
}
local TRAP_PRICES = { [1] = 3000000, [2] = 4000000, [3] = 5000000, [4] = 6000000 }

local function page(name)
  return DIR .. name
end

-- tokens splits a link's event on spaces (Java StringTokenizer).
local function tokens(event)
  local out = {}
  for t in event:gmatch("[^ ]+") do
    out[#out + 1] = t
  end
  return out
end

-- format_adena is FormatUtil.formatAdena: 1,234,567.
local function format_adena(n)
  local str = string.format("%d", n)
  local sign = ""
  if str:sub(1, 1) == "-" then
    sign, str = "-", str:sub(2)
  end
  local out = str:reverse():gsub("(%d%d%d)", "%1,"):reverse()
  if out:sub(1, 1) == "," then
    out = out:sub(2)
  end
  return sign .. out
end

-- strife_price is the Seal of Strife's change of an upgrade price: ×3 for
-- Dusk, ×0.8 for Dawn (Java int *= double).
local function strife_price(castle, price)
  local owner = castle:seal_owner(SEAL_STRIFE)
  if owner == CABAL_DUSK then
    return price * 3
  elseif owner == CABAL_DAWN then
    return math.floor(price * 0.8)
  end
  return price
end

local function door_upgrade_price(castle, typ, level)
  return strife_price(castle, (DOOR_PRICES[typ] or {})[level] or 0)
end

local function trap_upgrade_price(castle, level)
  return strife_price(castle, TRAP_PRICES[level] or 0)
end

local function function_fee(func, level)
  local f = FUNCTION_FEES[func]
  if f == nil then
    return 0
  end
  return level == f.level and f.first or f.other
end

local function seal_owner_name(castle, seal)
  local owner = castle:seal_owner(seal)
  if owner == CABAL_DAWN then
    return "Dawn"
  elseif owner == CABAL_DUSK then
    return "Dusk"
  end
  return "No Owner"
end

-- player_cabal is SevenSigns.getPlayerCabal.
local function player_cabal(player)
  return sevensigns.cabal(player)
end

local function is_owner(player, castle)
  return player:is_gm() or (player:clan() ~= nil and castle ~= nil and player:clan_id() == castle:owner_id())
end

local function can(player, castle, privilege)
  return is_owner(player, castle) and player:has_clan_privilege(privilege)
end

-- func_confirm is funcConfirmHtml: the confirmation page of setting a
-- castle function to a level.
local function func_confirm(player, npc, castle, func, level)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  local fstring = func == FUNC_TELEPORT and "9" or "10"
  local current = castle:get_function(func)
  if level == 0 then
    return page("castleresetdeco.html"), { AgitDecoSubmit = tostring(func) }
  elseif current ~= nil and current.level == level then
    return page("castledecoalreadyset.html"),
      { AgitDecoEffect = '<fstring p1="' .. level .. '">' .. fstring .. "</fstring>" }
  end
  return page("castledeco-0" .. func .. ".html"), {
    AgitDecoCost = '<fstring p1="' .. function_fee(func, level) .. '" p2="' .. J.idiv(FEE_RATIO, DAY_MS) .. '">6</fstring>',
    AgitDecoEffect = '<fstring p1="' .. level .. '">' .. fstring .. "</fstring>",
    AgitDecoSubmit = func .. " " .. level,
  }
end

-- func_replace is funcReplace: a function's lines of the recovery and other
-- pages.
local function func_replace(castle, vars, func, str)
  local f = castle:get_function(func)
  if f == nil then
    vars[str .. "Depth"] = "<fstring>4</fstring>"
    vars[str .. "Cost"] = ""
    vars[str .. "Expire"] = "<fstring>4</fstring>"
    vars[str .. "Reset"] = ""
    return
  end
  local fstring = (func == FUNC_SUPPORT or func == FUNC_TELEPORT) and "9" or "10"
  vars[str .. "Depth"] = '<fstring p1="' .. f.level .. '">' .. fstring .. "</fstring>"
  vars[str .. "Cost"] = '<fstring p1="' .. f.lease .. '" p2="' .. J.idiv(f.rate, DAY_MS) .. '">6</fstring>'
  vars[str .. "Expire"] = '<fstring p1="' .. f.end_day .. '" p2="' .. f.end_month .. '" p3="' .. f.end_year .. '">5</fstring>'
  vars[str .. "Reset"] = '[<a action="bypass -h Script CastleChamberlain ' .. str .. ' 0">Deactivate</a>]'
end

-- The pages a link may name directly.
local PAGES = {
  ["chamberlain-01.html"] = true,
  ["manor-help-01.html"] = true,
  ["manor-help-02.html"] = true,
  ["manor-help-03.html"] = true,
  ["manor-help-04.html"] = true,
  ["manor-help-05.html"] = true,
}

-- The function links: HP, MP, XP, TP and BF with a level.
local FUNCTION_LINKS = {
  HP = FUNC_RESTORE_HP,
  MP = FUNC_RESTORE_MP,
  XP = FUNC_RESTORE_EXP,
  TP = FUNC_TELEPORT,
  BF = FUNC_SUPPORT,
}

local handlers = {}

handlers.siege_functions = function(npc, player, castle)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  elseif castle:seven_signs_period() ~= PERIOD_COMP_RESULTS then
    return page("chamberlain-26.html")
  end
  return page("chamberlain-12.html")
end

handlers.manage_doors = function(npc, player, castle, args)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  elseif #args == 0 then
    return page(npc:id() .. "-du.html")
  end
  local doors = ""
  for i = 2, #args do
    doors = doors .. " " .. args[i]
  end
  return page("chamberlain-13.html"), { type = args[1], doors = doors }
end

handlers.upgrade_doors = function(npc, player, castle, args, event)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  local typ, level = J.parse_int(args[1]), J.parse_int(args[2])
  return page("chamberlain-14.html"), {
    gate_price = tostring(door_upgrade_price(castle, typ, level)),
    event = event:sub(#"upgrade_doors" + 2),
  }
end

-- "upgrade_doors_confirm <type> <level> <door> [<door>]": the doors (two at
-- most, as Mobius reads them) get the HP ratio when the first is below it
-- and the player pays (Castle.setDoorUpgrade); a door that is not the
-- castle's answers nothing.
handlers.upgrade_doors_confirm = function(npc, player, castle, args)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  end
  local typ, level = J.parse_int(args[1]), J.parse_int(args[2])
  local price = door_upgrade_price(castle, typ, level)
  local doors = {}
  for i = 3, math.min(#args, 4) do
    doors[#doors + 1] = J.parse_int(args[i])
  end
  if #doors == 0 then
    return nil
  end
  local current = castle:door_upgrade_level(doors[1])
  if current == nil then
    return nil
  elseif current >= level then
    return page("chamberlain-15.html"), { doorlevel = tostring(current) }
  elseif player:adena() < price then
    return page("chamberlain-09.html")
  end
  if price > 0 then
    s:take_items(player, 57, price)
  end
  for _, id in ipairs(doors) do
    castle:set_door_upgrade(id, level)
  end
  return page("chamberlain-16.html")
end

handlers.manage_trap = function(npc, player, castle, args)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  elseif #args == 0 then
    return page(npc:id() .. "-tu.html")
  end
  local name = castle:name():lower() == "aden" and "chamberlain-17a.html" or "chamberlain-17.html"
  return page(name), { trapIndex = args[1] }
end

handlers.upgrade_trap = function(npc, player, castle, args)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  local level = J.parse_int(args[2])
  return page("chamberlain-18.html"), {
    trapIndex = args[1],
    level = tostring(level),
    dmgzone_price = tostring(trap_upgrade_price(castle, level)),
  }
end

-- NOT BUILT: trap upgrades (Castle.setTrapUpgrade).
handlers.upgrade_trap_confirm = function(npc, player, castle)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  end
  return NOT_AVAILABLE
end

handlers.receive_report = function(npc, player, castle)
  if not npc:is_my_lord(player) then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-07.html")
  end
  local clan = castle:owner()
  local period = castle:seven_signs_period()
  local event
  if period == PERIOD_COMP_RECRUITING then
    event = "Preparation"
  elseif period == PERIOD_COMPETITION then
    event = "Competition"
  else
    event = "Seal Validation"
  end
  return page("chamberlain-02.html"), {
    clanleadername = clan:leader():name(),
    clanname = clan:name(),
    castlename = castle:name(),
    ss_event = event,
    ss_avarice = seal_owner_name(castle, SEAL_AVARICE),
    ss_gnosis = seal_owner_name(castle, SEAL_GNOSIS),
    ss_strife = seal_owner_name(castle, SEAL_STRIFE),
  }
end

handlers.manage_tax = function(npc, player, castle)
  if can(player, castle, "CASTLE_VAULT") then
    if castle:is_siege_in_progress() then
      return page("chamberlain-08.html")
    end
    return page("castlesettaxrate.html"), {
      tax_rate = tostring(castle:tax_percent()),
      next_tax_rate = "0", -- Mobius: TODO
      tax_limit = tostring(castle:max_tax()),
    }
  elseif is_owner(player, castle) then
    return page("chamberlain-03.html"), {
      tax_rate = tostring(castle:tax_percent()),
      next_tax_rate = "0", -- Mobius: TODO
    }
  end
  return page("chamberlain-21.html")
end

handlers.set_tax = function(npc, player, castle, args)
  if not can(player, castle, "CASTLE_VAULT") then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  end
  local tax = args[1] ~= nil and J.parse_int(args[1]) or 0
  local limit = castle:max_tax()
  if tax > limit then
    return page("castletoohightaxrate.html"), { tax_limit = tostring(limit) }
  end
  castle:set_tax_percent(tax)
  return page("castleaftersettaxrate.html"), { next_tax_rate = tostring(tax) }
end

handlers.manage_vault = function(npc, player, castle)
  if not can(player, castle, "CASTLE_VAULT") then
    return page("chamberlain-21.html")
  end
  local seed_income = 0
  if manor.is_enabled() then
    seed_income = manor.seed_income(castle:id())
  end
  return page("castlemanagevault.html"), {
    tax_income = format_adena(castle:treasury()),
    tax_income_reserved = "0", -- Mobius: TODO
    seed_income = format_adena(seed_income),
  }
end

handlers.deposit = function(npc, player, castle, args)
  if not can(player, castle, "CASTLE_VAULT") then
    return page("chamberlain-21.html")
  end
  local amount = args[1] ~= nil and J.parse_int(args[1]) or 0
  if amount > 0 and amount < MAX_ADENA then
    if player:adena() >= amount then
      castle:deposit(player, amount)
    else
      player:send_system_message(SystemMessageId.YOU_DO_NOT_HAVE_ENOUGH_ADENA)
    end
  end
  return page("chamberlain-01.html")
end

handlers.withdraw = function(npc, player, castle, args)
  if not can(player, castle, "CASTLE_VAULT") then
    return page("chamberlain-21.html")
  end
  local amount = args[1] ~= nil and J.parse_int(args[1]) or 0
  if amount <= 0 then
    return page("chamberlain-01.html")
  elseif amount <= castle:treasury() then
    castle:withdraw(player, amount)
    return page("chamberlain-01.html")
  end
  return page("castlenotenoughbalance.html"), {
    tax_income = format_adena(castle:treasury()),
    withdraw_amount = format_adena(amount),
  }
end

handlers.manage_functions = function(npc, player, castle)
  if not is_owner(player, castle) then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  end
  return page("chamberlain-23.html")
end

handlers.banish_foreigner_show = function(npc, player, castle)
  if not can(player, castle, "CASTLE_BANISH") then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  end
  return page("chamberlain-10.html")
end

handlers.banish_foreigner = function(npc, player, castle)
  if not can(player, castle, "CASTLE_BANISH") then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  end
  castle:banish_foreigners()
  return page("chamberlain-11.html")
end

handlers.doors = function(npc, player, castle)
  if not can(player, castle, "CASTLE_OPEN_DOOR") then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  end
  return page(npc:id() .. "-d.html")
end

handlers.operate_door = function(npc, player, castle, args)
  if not can(player, castle, "CASTLE_OPEN_DOOR") then
    return page("chamberlain-21.html")
  elseif castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  end
  local open = J.parse_int(args[1]) == 1
  for i = 2, #args do
    castle:open_close_door(player, J.parse_int(args[i]), open)
  end
  return page(open and "chamberlain-05.html" or "chamberlain-06.html")
end

handlers.additional_functions = function(npc, player, castle)
  if can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("castletdecomanage.html")
  end
  return page("chamberlain-21.html")
end

handlers.recovery = function(npc, player, castle)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  local vars = {}
  func_replace(castle, vars, FUNC_RESTORE_HP, "HP")
  func_replace(castle, vars, FUNC_RESTORE_MP, "MP")
  func_replace(castle, vars, FUNC_RESTORE_EXP, "XP")
  return page("castledeco-AR01.html"), vars
end

handlers.other = function(npc, player, castle)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  local vars = {}
  func_replace(castle, vars, FUNC_TELEPORT, "TP")
  func_replace(castle, vars, FUNC_SUPPORT, "BF")
  return page("castledeco-AE01.html"), vars
end

-- NOT BUILT: renting castle functions (Castle.updateFunctions with the fee
-- from the clan warehouse).
handlers.set_func = function(npc, player, castle)
  if not can(player, castle, "CASTLE_MANAGE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  return NOT_AVAILABLE
end

handlers.functions = function(npc, player, castle)
  if not can(player, castle, "CASTLE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  local function depth(func)
    local f = castle:get_function(func)
    return f == nil and "0" or tostring(f.level)
  end
  return page("castledecofunction.html"), {
    HPDepth = depth(FUNC_RESTORE_HP),
    MPDepth = depth(FUNC_RESTORE_MP),
    XPDepth = depth(FUNC_RESTORE_EXP),
  }
end

handlers.teleport = function(npc, player, castle)
  if not can(player, castle, "CASTLE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  local f = castle:get_function(FUNC_TELEPORT)
  if f == nil then
    return page("castlefuncdisabled.html")
  end
  return page(npc:id() .. "-t" .. f.level .. ".html")
end

-- NOT BUILT: the teleport function's destinations (TeleporterData OTHER
-- lists of the chamberlains).
handlers["goto"] = function(npc, player, castle)
  if not can(player, castle, "CASTLE_FUNCTIONS") then
    return page("chamberlain-21.html")
  elseif castle:get_function(FUNC_TELEPORT) == nil then
    return page("castlefuncdisabled.html")
  end
  return NOT_AVAILABLE
end

handlers.buffer = function(npc, player, castle)
  if not can(player, castle, "CASTLE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  local f = castle:get_function(FUNC_SUPPORT)
  if f == nil then
    return page("castlefuncdisabled.html")
  end
  return page("castlebuff-0" .. f.level .. ".html"), { MPLeft = tostring(math.floor(npc:mp())) }
end

-- NOT BUILT: the support function's buffs (the skill's MP cost is not in
-- the Lua API).
handlers.cast_buff = function(npc, player, castle)
  if not can(player, castle, "CASTLE_FUNCTIONS") then
    return page("chamberlain-21.html")
  elseif castle:get_function(FUNC_SUPPORT) == nil then
    return page("castlefuncdisabled.html")
  end
  return NOT_AVAILABLE
end

handlers.list_siege_clans = function(npc, player, castle)
  if not can(player, castle, "CASTLE_SIEGE") then
    return page("chamberlain-21.html")
  end
  castle:show_siege_info(player)
  return nil
end

-- The manor page; its manor_menu_select links open the game's manor
-- windows (Mobius CastleChamberlain.onNpcManorBypass).
handlers.manor = function(npc, player, castle)
  if not manor.is_enabled() then
    player:send_message("Manor system is deactivated.")
    return nil
  end
  if can(player, castle, "CASTLE_MANOR") then
    return page("manor.html")
  end
  return page("chamberlain-21.html")
end

handlers.products = function(npc, player, castle)
  if not can(player, castle, "CASTLE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  return page("chamberlain-22.html"), { npcId = tostring(npc:id()) }
end

handlers.buy = function(npc, player, castle, args)
  if not can(player, castle, "CASTLE_FUNCTIONS") then
    return page("chamberlain-21.html")
  end
  npc:show_buy_window(player, J.parse_int(args[1]))
  return nil
end

handlers.give_crown = function(npc, player, castle)
  if castle:is_siege_in_progress() then
    return page("chamberlain-08.html")
  elseif not npc:is_my_lord(player) then
    return page("chamberlain-21.html")
  elseif s:has_quest_items(player, CROWN) then
    return page("chamberlain-24.html")
  end
  s:give_items(player, CROWN, 1)
  return page("chamberlain-25.html"), { owner_name = player:name(), feud_name = castle:name() }
end

-- The Lord of the Manor's Certificate of Approval, sold to a lord of Dawn
-- in the competition period in bundles, up to the castle's quantity per
-- competition (manors_cert shows the sale, manors_cert_confirm buys).
local function manors_cert(confirm)
  return function(npc, player, castle)
    if not npc:is_my_lord(player) then
      return page("chamberlain-21.html")
    elseif castle:is_siege_in_progress() then
      return page("chamberlain-08.html")
    elseif player_cabal(player) ~= CABAL_DAWN or castle:seven_signs_period() ~= PERIOD_COMPETITION then
      return page("ssq_notdawnorevent.html")
    end
    local cfg = sevensigns.config()
    local sold = sevensigns.ticket_count(castle:id())
    if sold >= J.idiv(cfg.dawn_ticket_quantity, cfg.dawn_ticket_bundle) then
      return page("ssq_notenoughticket.html")
    end
    if not confirm then
      return page("ssq_selldawnticket.html"), {
        DawnTicketLeft = tostring(cfg.dawn_ticket_quantity - sold * cfg.dawn_ticket_bundle),
        DawnTicketBundle = tostring(cfg.dawn_ticket_bundle),
        DawnTicketPrice = tostring(cfg.dawn_ticket_price * cfg.dawn_ticket_bundle),
      }
    end
    local cost = cfg.dawn_ticket_price * cfg.dawn_ticket_bundle
    if player:adena() < cost then
      return page("chamberlain-09.html")
    end
    s:take_items(player, 57, cost) -- adena
    s:give_items(player, cfg.manors_agreement_id, cfg.dawn_ticket_bundle)
    sevensigns.set_ticket_count(castle:id(), sold + 1)
    return nil
  end
end
handlers.manors_cert = manors_cert(false)
handlers.manors_cert_confirm = manors_cert(true)

s:on_first_talk(NPCS, function(npc, player)
  if is_owner(player, npc:castle()) then
    return page("chamberlain-01.html")
  end
  return page("chamberlain-04.html")
end)

s:on_event(function(event, npc, player)
  if npc == nil or player == nil then
    return nil
  end
  local args = tokens(event)
  local cmd = table.remove(args, 1)
  if cmd == nil then
    return nil
  end
  if PAGES[cmd] then
    return page(cmd)
  end
  local castle = npc:castle()
  if castle == nil then
    return nil
  end
  local func = FUNCTION_LINKS[cmd]
  if func ~= nil then
    return func_confirm(player, npc, castle, func, J.parse_int(args[1]))
  end
  local handler = handlers[cmd]
  if handler == nil then
    return nil
  end
  return handler(npc, player, castle, args, event)
end)
