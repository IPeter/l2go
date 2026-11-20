-- Turek orcs flee and come back with help, ported from Mobius
-- ai/others/TurekOrcs (GKR).
--
-- A hit orc below half HP may shout and run to the flee point of its
-- spawn (AI data fleeX/fleeY/fleeZ). There it warns the orcs within 400,
-- which attack the player who hit it, and after 15 s it goes home if it
-- has healed above 70 %. The state variable: 1 fleeing, 2 at the flee
-- point, 3 going home.

local s = script.new("TurekOrcs")

local MOBS = {
  20494, -- Turek War Hound
  20495, -- Turek Orc Warlord
  20497, -- Turek Orc Skirmisher
  20498, -- Turek Orc Supplier
  20499, -- Turek Orc Footman
  20500, -- Turek Orc Sentinel
}

local MESSAGES = {
  "We shall see about that!",
  "I will definitely repay this humiliation!",
  "Retreat!",
  "Tactical retreat!",
  "Mass fleeing!",
  "It's stronger than expected!",
  "I'll kill you next time!",
  "I'll definitely kill you next time!",
  "Oh! How strong!",
  "Invader!",
  "There is no reason for you to kill me! I have nothing you need!",
  "Someday you will pay!",
  "I won't just stand still while you hit me.",
  "Stop hitting!",
  "It hurts to the bone!",
  "Am I the neighborhood drum for beating!",
  "Follow me if you want!",
  "Surrender!",
  "Oh, I'm dead!",
  "I'll be back!",
  "I'll give you ten million arena if you let me live!",
}

s:add_attack_id(MOBS)
s:add_event_received_id(MOBS)
s:add_move_finished_id(MOBS)

-- flee_point is the spawn's flee point.
local function flee_point(npc)
  return { x = npc:ai_value("fleeX"), y = npc:ai_value("fleeY"), z = npc:ai_value("fleeZ") }
end

s:on_event(function(event, npc, player)
  if event == "checkState" and npc ~= nil and not npc:is_dead() and npc:intention() ~= Intention.ATTACK then
    if npc:hp() > npc:max_hp() * 0.7 and npc:var_int("state") == 2 then
      npc:set_var("state", 3)
      npc:return_home()
    else
      npc:unset_var("state")
    end
  end
end)

s:on_attack(function(npc, attacker, damage, is_summon)
  if not npc:has_var("isHit") then
    npc:set_var("isHit", 1)
  elseif npc:hp() < npc:max_hp() * 0.5 and npc:hp() > npc:max_hp() * 0.3
    and attacker:hp() > attacker:max_hp() * 0.25
    and npc:has_ai_value("fleeX") and npc:has_ai_value("fleeY") and npc:has_ai_value("fleeZ")
    and npc:var_int("state") == 0 and rnd(100) < 10 then
    -- Say and flee.
    npc:broadcast_say(ChatType.GENERAL, rnd.entry(MESSAGES))
    npc:disable_core_ai(true) -- no attacking while it flees
    npc:set_running()
    npc:move_to(flee_point(npc))
    npc:set_var("state", 1)
    npc:set_var("attacker", attacker:object_id())
  end
end)

s:on_event_received(function(event, sender, receiver, reference)
  if event == "WARNING" and not receiver:is_dead() and receiver:intention() ~= Intention.ATTACK and reference ~= nil then
    local player = world.get_player(reference:object_id())
    if player ~= nil and not player:is_dead() then
      receiver:set_var("state", 3)
      receiver:set_running()
      receiver:add_damage_hate(player, 0, 99999)
      receiver:attack(player)
    end
  end
end)

s:on_move_finished(function(npc)
  local state = npc:var_int("state")
  if state == 1 then
    -- At the flee point: call for help.
    if npc:x() == npc:ai_value("fleeX") and npc:y() == npc:ai_value("fleeY") then
      npc:disable_core_ai(false)
      s:start_quest_timer("checkState", 15000, npc, nil)
      npc:set_var("state", 2)
      npc:broadcast_event("WARNING", 400, world.get_player(npc:var_int("attacker")))
    else
      npc:move_to(flee_point(npc))
    end
  elseif state == 3 and npc:stays_in_spawn_loc() then
    npc:disable_core_ai(false)
    npc:unset_var("state")
  end
end)
