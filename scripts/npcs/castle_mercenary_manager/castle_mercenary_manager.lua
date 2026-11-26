-- The castle mercenary managers, ported from Mobius
-- ai/others/CastleMercenaryManager.
--
-- The lord (or a GM, or a member of the owner clan with the mercenary
-- privilege) hires mercenaries: the three buylists of posting tickets
-- (<npc ID><1-3>, without the castle tax shown, as Mobius), in the seal
-- validation period only; the ticket limits per castle; everyone else gets
-- mercmanager-no.html. During a siege the manager refuses
-- (mercmanager-siege.html). The page depends on the Seal of Strife's owner,
-- which is nobody until the Seven Signs exist (M11l).
--
-- The pages are Mobius's, served from
-- data/html/scripts/ai/others/CastleMercenaryManager/.

local J = require("lib.java")

local s = script.new("CastleMercenaryManager")

local NPCS = {
  35102, -- Greenspan
  35144, -- Sanford
  35186, -- Arvid
  35228, -- Morrison
  35276, -- Eldon
  35318, -- Solinus
  35365, -- Rowell
  35511, -- Gompus
  35557, -- Kendrew
}

local DIR = "data/html/scripts/ai/others/CastleMercenaryManager/"

-- Seven Signs (Mobius SevenSigns).
local CABAL_DUSK = 1
local CABAL_DAWN = 2
local SEAL_STRIFE = 3
local PERIOD_SEAL_VALIDATION = 3

local function first_talk(npc, player)
  local castle = npc:castle()
  if castle == nil then
    return nil
  end
  if player:is_gm() or (player:clan_id() == castle:owner_id() and player:has_clan_privilege("CASTLE_MERCENARIES")) then
    if castle:is_siege_in_progress() then
      return DIR .. "mercmanager-siege.html"
    end
    local owner = castle:seal_owner(SEAL_STRIFE)
    if owner == CABAL_DUSK then
      return DIR .. "mercmanager-dusk.html"
    elseif owner == CABAL_DAWN then
      return DIR .. "mercmanager-dawn.html"
    end
    return DIR .. "mercmanager.html"
  end
  return DIR .. "mercmanager-no.html"
end

s:on_first_talk(NPCS, first_talk)

s:on_event(function(event, npc, player)
  if npc == nil or player == nil then
    return nil
  end
  local cmd, arg = event:match("^(%S+)%s*(%S*)")
  if cmd == "limit" then
    local castle = npc:castle()
    if castle == nil then
      return nil
    end
    -- Mobius compares the name with "aden" and "rune" case-sensitively,
    -- which never matches its capitalised castle names; the comparison
    -- here ignores case, so that Aden and Rune show their own limits.
    local name = castle:name():lower()
    local page = "mercmanager-limit.html"
    if name == "aden" then
      page = "mercmanager-aden-limit.html"
    elseif name == "rune" then
      page = "mercmanager-rune-limit.html"
    end
    return DIR .. page, { feud_name = castle:name() }
  elseif cmd == "buy" then
    local castle = npc:castle()
    if castle == nil then
      return nil
    end
    if castle:seven_signs_period() == PERIOD_SEAL_VALIDATION then
      -- Not affected by the castle taxes: the buylists carry their 20 %.
      npc:show_buy_window(player, J.parse_int(npc:id() .. arg), false)
      return nil
    end
    return DIR .. "mercmanager-ssq.html"
  elseif cmd == "main" then
    return first_talk(npc, player)
  elseif cmd == "mercmanager-01.html" then
    return DIR .. cmd
  end
  return nil
end)
