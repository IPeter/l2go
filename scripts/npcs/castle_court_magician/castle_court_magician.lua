-- The castle court magicians, ported from Mobius
-- ai/others/CastleCourtMagician.
--
-- Members of the owner clan move to their clan leader through the leader's
-- Clan Gate (skill 3632), and trade 10 Knight's Epaulettes for a random
-- talisman; everyone else gets courtmagician-01.html. The epaulettes and
-- talismans are items of later chronicles that the Interlude item data does
-- not have, so the trade always answers courtmagician-06.html (not enough
-- epaulettes). The low-ranked soldiers' skills answer courtmagician-05.html,
-- as in Mobius.
--
-- The pages are Mobius's, served from
-- data/html/scripts/ai/others/CastleCourtMagician/.

local s = script.new("CastleCourtMagician")

local COURT_MAGICIAN = {
  35648, -- Gludio
  35649, -- Dion
  35650, -- Giran
  35651, -- Oren
  35652, -- Aden
  35653, -- Innadril
  35654, -- Goddard
  35655, -- Rune
  35656, -- Schuttgart
}

local DIR = "data/html/scripts/ai/others/CastleCourtMagician/"

-- Skills
local CLAN_GATE = 3632 -- Clan Gate
local DISPLAY_CLAN_GATE = 5109 -- Production - Clan Gate, level 1
-- Items
local EPAULETTE = 9912 -- Knight's Epaulette
local RED_MEDITATION = 9931 -- Red Talisman of Meditation
local BLUE_DIV_PROTECTION = 9932 -- Blue Talisman - Divine Protection
local BLUE_EXPLOSION = 10416 -- Blue Talisman - Explosion
local BLUE_M_EXPLOSION = 10417 -- Blue Talisman - Magic Explosion
local RED_MIN_CLARITY = 9917 -- Red Talisman of Minimum Clarity
local RED_MAX_CLARITY = 9918 -- Red Talisman of Maximum Clarity
local RED_MENTAL_REG = 9928 -- Red Talisman of Mental Regeneration
local BLUE_PROTECTION = 9929 -- Blue Talisman of Protection
local BLUE_INVIS = 9920 -- Blue Talisman of Invisibility
local BLUE_DEFENSE = 9916 -- Blue Talisman of Defense
local BLACK_ESCAPE = 9923 -- Black Talisman - Escape
local BLUE_HEALING = 9924 -- Blue Talisman of Healing
local RED_RECOVERY = 9925 -- Red Talisman of Recovery
local BLUE_DEFENSE2 = 9926 -- Blue Talisman of Defense
local BLUE_M_DEFENSE = 9927 -- Blue Talisman of Magic Defense
local RED_LIFE_FORCE = 10518 -- Red Talisman - Life Force
local BLUE_GREAT_HEAL = 10424 -- Blue Talisman - Greater Healing
local WHITE_FIRE = 10421 -- White Talisman - Fire
-- Mobius COMMON_TALISMANS, in its order (0-based there).
local COMMON_TALISMANS = {
  9914, 9915, 9920, 9921, 9922, 9933, 9934, 9935, 9936, 9937, 9938, 9939, 9940, 9941, 9942, 9943,
  9944, 9945, 9946, 9947, 9948, 9949, 9950, 9952, 9953, 9954, 9955, 9956, 9957, 9958, 9959, 9960,
  9961, 9962, 9963, 9964, 9965, 10418, 10420, 10519, 10422, 10423, 10419,
}

-- talisman draws the talisman of giveTalisman with Mobius's chances.
local function talisman()
  local category = rnd(100)
  if category <= 5 then
    local chance = rnd(100)
    if chance <= 25 then
      return RED_MEDITATION
    elseif chance <= 50 then
      return BLUE_DIV_PROTECTION
    elseif chance <= 75 then
      return BLUE_EXPLOSION
    end
    return BLUE_M_EXPLOSION
  elseif category <= 15 then
    local chance = rnd(100)
    if chance <= 20 then
      return RED_MIN_CLARITY
    elseif chance <= 40 then
      return RED_MAX_CLARITY
    elseif chance <= 60 then
      return RED_MENTAL_REG
    elseif chance <= 80 then
      return BLUE_PROTECTION
    end
    return BLUE_INVIS
  elseif category <= 30 then
    local chance = rnd(100)
    if chance <= 12 then
      return BLUE_DEFENSE
    elseif chance <= 25 then
      return BLACK_ESCAPE
    elseif chance <= 37 then
      return BLUE_HEALING
    elseif chance <= 50 then
      return RED_RECOVERY
    elseif chance <= 62 then
      return BLUE_DEFENSE2
    elseif chance <= 75 then
      return BLUE_M_DEFENSE
    elseif chance <= 87 then
      return RED_LIFE_FORCE
    end
    return BLUE_GREAT_HEAL
  end
  local chance = rnd(46)
  if chance <= 42 then
    return COMMON_TALISMANS[chance + 1]
  end
  return WHITE_FIRE
end

local function is_owner_member(player, npc)
  local castle = npc:castle()
  return player:clan() ~= nil and castle ~= nil and player:clan_id() == castle:owner_id()
end

s:on_event(function(event, npc, player)
  if npc == nil or player == nil then
    return nil
  end
  if not is_owner_member(player, npc) then
    return DIR .. "courtmagician-01.html"
  end
  if event == "courtmagician.html" or event == "courtmagician-03.html" then
    return DIR .. event
  elseif event == "giveTalisman" then
    if s:get_quest_items_count(player, EPAULETTE) < 10 then
      return DIR .. "courtmagician-06.html"
    end
    -- Mobius refuses a full inventory here (YOUR_INVENTORY_IS_FULL); the
    -- slots are not in the Lua API, and give_items refuses a full
    -- inventory itself.
    local item = talisman()
    s:take_items(player, EPAULETTE, 10)
    s:give_items(player, item, 1)
    return DIR .. "courtmagician-04.html"
  elseif event == "squadSkill" then
    return DIR .. "courtmagician-05.html"
  elseif event == "clanTeleport" then
    local leader = player:clan():leader():player()
    if leader ~= nil and leader:is_affected_by_skill(CLAN_GATE) then
      -- Mobius also checks CallPc.checkSummonTargetStatus (its own custom
      -- check, not retail's).
      npc:set_target(player)
      npc:cast(skills.get(DISPLAY_CLAN_GATE, 1))
      player:teleport(leader:x(), leader:y(), leader:z())
      return nil
    end
    return DIR .. "courtmagician-02.html"
  end
  return nil
end)

s:on_first_talk(COURT_MAGICIAN, function(npc, player)
  if is_owner_member(player, npc) then
    return DIR .. "courtmagician.html"
  end
  return DIR .. "courtmagician-01.html"
end)
