-- Translated by questport from quests/Q00615_MagicalPowerOfFirePart1 (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Magical Power of Fire - Part 1 (615)
-- @author Joxit
local J = require("lib.java")

local q = quest.new(615, "Q00615_MagicalPowerOfFirePart1", "Magical Power of Fire - Part 1")

-- NPCs
local NARAN = 31378
local UDAN = 31379
local ASEFA_BOX = 31559
local ASEFA_EYE = 31684
-- Monsters
local KETRA_MOBS = {
  21324,
  21325,
  21327,
  21328,
  21329,
  21331,
  21332,
  21334,
  21335,
  21336,
  21338,
  21339,
  21340,
  21342,
  21343,
  21344,
  21345,
  21346,
  21347,
  21348,
  21349,
}
-- Items
local KEY = 1661
local STOLEN_RED_TOTEM = 7242
local WISDOM_STONE = 7081
local RED_TOTEM = 7243
local VARKA_MARKS = {7221, 7222, 7223, 7224, 7225}
-- Skills
local GOW = {id = 4547, level = 1} -- Gaze of Watcher
local DISPEL_GOW = {id = 4548, level = 1} -- Quest - Dispel Watcher Gaze
-- Misc
local MIN_LEVEL = 74

q:add_start_npc(NARAN)
q:add_talk_id(UDAN, NARAN, ASEFA_BOX)
q:add_attack_id(KETRA_MOBS)
q:register_quest_items(STOLEN_RED_TOTEM)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = nil
  if event == "31378-02.html" then
    qs:start_quest()
    htmltext = event
  elseif event == "open_box" then
    if not q:has_quest_items(player, KEY) then
      htmltext = "31559-02.html"
    elseif qs:is_cond(2) then
      if qs:is_set("spawned") then
        q:take_items(player, KEY, 1)
        htmltext = "31559-04.html"
      else
        q:give_items(player, STOLEN_RED_TOTEM, 1)
        q:take_items(player, KEY, 1)
        qs:set_cond(3, true)
        htmltext = "31559-03.html"
      end
    end
  elseif event == "eye_despawn" then
    npc:broadcast_say(ChatType.NPC_GENERAL, "Asefa has already seen your face!")
    npc:delete()
  end
  return htmltext
end)

q:on_attack(function(npc, attacker, damage, isSummon)
  local qs = q:get_quest_state(attacker, false)
  if qs ~= nil and qs:is_cond(2) and not qs:is_set("spawned") then
    qs:set("spawned", "1")
    npc:set_target(attacker)
    npc:cast(skills.get(GOW.id, GOW.level))
    local eye = q:add_spawn(ASEFA_EYE, npc)
    eye:broadcast_say(ChatType.NPC_GENERAL, "You can't avoid the eyes of Asefa!")
    q:start_quest_timer("eye_despawn", 10000, eye, attacker)
  end
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = npc:id()
  if sw == NARAN then
    local sw2 = qs:get_state()
    if sw2 == State.CREATED then
      htmltext = player:level() >= MIN_LEVEL and (q:has_at_least_one_quest_item(player, unpack(VARKA_MARKS)) and "31378-01.htm" or "31378-00a.html") or "31378-00b.html"
    elseif sw2 == State.STARTED then
      if qs:is_cond(1) then
        htmltext = "31378-03.html"
      end
    end
  elseif sw == UDAN then
    if qs:is_started() then
      local sw3 = qs:get_cond()
      if sw3 == 1 then
        htmltext = "31379-01.html"
        qs:set_cond(2, true)
      elseif sw3 == 2 then
        if qs:is_set("spawned") then
          qs:unset("spawned")
          npc:set_target(player)
          npc:cast(skills.get(DISPEL_GOW.id, DISPEL_GOW.level))
          htmltext = "31379-03.html"
        else
          htmltext = "31379-02.html"
        end
      elseif sw3 == 3 then
        q:give_items(player, RED_TOTEM, 1)
        q:give_items(player, WISDOM_STONE, 1)
        qs:exit_quest(true, true)
        htmltext = "31379-04.html"
      end
    end
  elseif sw == ASEFA_BOX then
    if qs:is_cond(2) then
      htmltext = "31559-01.html"
    end
  end
  return htmltext
end)
