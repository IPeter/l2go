-- The wyvern managers, ported from Mobius ai/others/WyvernManager.
--
-- The lord of the castle (the clan leader of the owner clan) riding a
-- strider of level 55 or more trades it and 25 B-grade crystals for a
-- wyvern (wyvernmanager-04.html); a rider who is not the lord or lacks the
-- crystals gets wyvernmanager-06.html, a player not on a strider the "ride
-- a strider" system message and wyvernmanager-05.html; everyone but the
-- lord is greeted with wyvernmanager-02.html. In the seal validation period
-- with Dusk owning the Seal of Strife the castle managers refuse
-- (wyvernmanager-dusk.html).
--
-- Mobius also lists the managers of two clan halls, whose owner is the
-- hall's; clan halls come with M11j, so their lords are nobody until then.
--
-- The pages are Mobius's, served from
-- data/html/scripts/ai/others/WyvernManager/.

local s = script.new("WyvernManager")

local CASTLE = "castle"
local CLAN_HALL = "clan_hall"

local MANAGERS = {
  [35101] = CASTLE,
  [35143] = CASTLE,
  [35185] = CASTLE,
  [35227] = CASTLE,
  [35275] = CASTLE,
  [35317] = CASTLE,
  [35364] = CASTLE,
  [35510] = CASTLE,
  [35536] = CASTLE,
  [35556] = CASTLE,
  [35419] = CLAN_HALL,
  [35638] = CLAN_HALL,
}

local DIR = "data/html/scripts/ai/others/WyvernManager/"

local CRYSTAL_B_GRADE = 1460
local WYVERN = 12621
local WYVERN_FEE = 25
local STRIDER_LEVEL = 55
local STRIDERS = {
  [12526] = true,
  [12527] = true,
  [12528] = true,
  [16038] = true,
  [16039] = true,
  [16040] = true,
  [16068] = true,
  [13197] = true,
}

-- Mobius MountType.STRIDER (player:mount_type()).
local MOUNT_STRIDER = 1

-- Mobius Feature.ini (AllowRideWyvernAlways, AllowRideWyvernDuringSiege).
local ALLOW_WYVERN_ALWAYS = false
local ALLOW_WYVERN_DURING_SIEGE = false

-- Seven Signs (Mobius SevenSigns).
local CABAL_DUSK = 1
local SEAL_STRIFE = 3
local PERIOD_SEAL_VALIDATION = 3

local function ids()
  local out = {}
  for id in pairs(MANAGERS) do
    out[#out + 1] = id
  end
  table.sort(out)
  return out
end

-- is_owner_clan is isOwnerClan: the player leads the residence's owner
-- clan. A clan hall has no owner until clan halls exist (M11j).
local function is_owner_clan(npc, player)
  if not player:is_clan_leader() then
    return false
  end
  if MANAGERS[npc:id()] == CASTLE then
    local castle = npc:castle()
    return castle ~= nil and player:clan_id() == castle:owner_id()
  end
  return false
end

-- is_in_siege is isInSiege: the castle's siege zone is active.
local function is_in_siege(npc)
  if MANAGERS[npc:id()] == CASTLE then
    local castle = npc:castle()
    return castle ~= nil and castle:is_zone_active()
  end
  local castle = npc:castle()
  return castle ~= nil and castle:is_siege_in_progress()
end

-- dusk_rules: a castle manager in the seal validation period with Dusk
-- owning the Seal of Strife.
local function dusk_rules(npc)
  if MANAGERS[npc:id()] ~= CASTLE then
    return false
  end
  local castle = npc:castle()
  return castle ~= nil and castle:seven_signs_period() == PERIOD_SEAL_VALIDATION
    and castle:seal_owner(SEAL_STRIFE) == CABAL_DUSK
end

local function part_vars()
  return { wyvern_fee = tostring(WYVERN_FEE), strider_level = tostring(STRIDER_LEVEL) }
end

-- replace_all is wyvernmanager-01.html with the residence's name.
local function replace_all(npc)
  local vars = part_vars()
  local castle = npc:castle()
  vars.residence_name = castle ~= nil and castle:name() or ""
  return DIR .. "wyvernmanager-01.html", vars
end

-- riding_strider is Mobius's mounted check: mounted on one of the striders,
-- of level 55 or more.
local function riding_strider(player)
  return player:is_mounted() and player:mount_level() >= STRIDER_LEVEL and STRIDERS[player:mount_npc_id()] == true
end

-- mount_wyvern is mountWyvern: the lord pays the fee, gets off the strider
-- and onto a wyvern that eats.
local function mount_wyvern(npc, player)
  if riding_strider(player) then
    if is_owner_clan(npc, player) and s:get_quest_items_count(player, CRYSTAL_B_GRADE) >= WYVERN_FEE then
      s:take_items(player, CRYSTAL_B_GRADE, WYVERN_FEE)
      player:dismount()
      player:mount(WYVERN, 0, true)
      return DIR .. "wyvernmanager-04.html"
    end
    return DIR .. "wyvernmanager-06.html", part_vars()
  end
  if player:mount_type() ~= MOUNT_STRIDER then
    player:send_system_message(SystemMessageId.YOU_MAY_ONLY_RIDE_A_WYVERN_WHILE_YOU_RE_RIDING_A_STRIDER)
  end
  return DIR .. "wyvernmanager-05.html", part_vars()
end

local function greeting(npc, player)
  if not is_owner_clan(npc, player) then
    return DIR .. "wyvernmanager-02.html"
  elseif not ALLOW_WYVERN_ALWAYS and dusk_rules(npc) then
    return DIR .. "wyvernmanager-dusk.html"
  end
  return replace_all(npc)
end

s:on_event(function(event, npc, player)
  if npc == nil or player == nil or MANAGERS[npc:id()] == nil then
    return nil
  end
  if event == "Return" then
    return greeting(npc, player)
  elseif event == "Help" then
    if MANAGERS[npc:id()] == CASTLE then
      return DIR .. "wyvernmanager-03.html", part_vars()
    end
    return DIR .. "wyvernmanager-03b.html", part_vars()
  elseif event == "RideWyvern" then
    if ALLOW_WYVERN_ALWAYS then
      return mount_wyvern(npc, player)
    end
    if not ALLOW_WYVERN_DURING_SIEGE and player:is_clan_leader() and is_in_siege(npc) then
      local castle = npc:castle()
      if castle == nil or player:clan_id() ~= castle:owner_id() then
        player:send_message("You cannot summon a wyvern during a siege.")
        return nil
      end
    end
    if dusk_rules(npc) then
      return DIR .. "wyvernmanager-dusk.html"
    end
    return mount_wyvern(npc, player)
  end
  return nil
end)

s:on_first_talk(ids(), greeting)
