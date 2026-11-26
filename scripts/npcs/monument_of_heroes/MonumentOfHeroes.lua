-- The Monuments of Heroes, ported from Mobius ai/others/MonumentOfHeroes.
--
-- A noble or a hero sees the game's olympiad/hero_main.htm at a monument;
-- its links come here: an unclaimed hero of the last Olympiad period
-- claims the hero status (HeroReceive), a hero takes one Infinity weapon
-- and one Wings of Destiny Circlet. The hero list ("Olympiad 4") is the
-- game's.

local s = script.new("MonumentOfHeroes")

local MONUMENTS = { 31690, 31769, 31770, 31771, 31772 }

local WINGS_OF_DESTINY_CIRCLET = 6842
local WEAPONS = {
  6611, -- Infinity Blade
  6612, -- Infinity Cleaver
  6613, -- Infinity Axe
  6614, -- Infinity Rod
  6615, -- Infinity Crusher
  6616, -- Infinity Scepter
  6617, -- Infinity Stinger
  6618, -- Infinity Fang
  6619, -- Infinity Bow
  6620, -- Infinity Wing
  6621, -- Infinity Spear
}

s:add_start_npc(MONUMENTS)
s:add_talk_id(MONUMENTS)

local function is_weapon(id)
  for _, w in ipairs(WEAPONS) do
    if w == id then
      return true
    end
  end
  return false
end

s:on_event(function(event, npc, player)
  if event == "HeroClaim" then
    if hero.is_hero(player) then
      return "already_hero_status.htm"
    elseif not hero.is_unclaimed(player) then
      return "no_hero_status.htm"
    end
    return "receive_hero.htm"
  elseif event == "HeroReceive" then
    hero.claim(player)
    return nil
  elseif event == "HeroWeapon" then
    if player:is_hero() then
      if s:has_at_least_one_quest_item(player, WEAPONS) then
        return "already_have_weapon.htm"
      end
      return "weapon_list.htm"
    end
    return "not_a_hero.htm"
  elseif event == "HeroCirclet" then
    if not player:is_hero() then
      return "not_a_hero.htm"
    end
    if s:has_quest_items(player, WINGS_OF_DESTINY_CIRCLET) then
      return "already_have_circlet.htm"
    end
    s:give_items(player, WINGS_OF_DESTINY_CIRCLET, 1)
    return nil
  end
  -- A weapon of the list (Mobius Integer.parseInt of the event).
  local id = tonumber(event)
  if id ~= nil and is_weapon(id) then
    s:give_items(player, id, 1)
  end
  return nil
end)
