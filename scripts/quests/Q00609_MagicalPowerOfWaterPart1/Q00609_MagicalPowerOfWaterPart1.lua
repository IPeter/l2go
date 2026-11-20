-- Translated by questport from quests/Q00609_MagicalPowerOfWaterPart1 (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Magical Power of Water - Part 1 (609)
-- @author Joxit
local J = require("lib.java")

local q = quest.new(609, "Q00609_MagicalPowerOfWaterPart1", "Magical Power of Water - Part 1")

-- NPCs
local WAHKAN = 31371
local ASEFA = 31372
local UDANS_BOX = 31561
local UDANS_EYE = 31684
-- Monsters
local VARKA_MOBS = {
  21350,
  21351,
  21353,
  21354,
  21355,
  21357,
  21358,
  21360,
  21361,
  21362,
  21364,
  21365,
  21366,
  21368,
  21369,
  21370,
  21371,
  21372,
  21373,
  21374,
  21375,
}
-- Items
local KEY = 1661
local STOLEN_GREEN_TOTEM = 7237
local WISDOM_STONE = 7081
local GREEN_TOTEM = 7238
local KETRA_MARKS = {7211, 7212, 7213, 7214, 7215}
-- Skills
local GOW = {id = 4547, level = 1} -- Gaze of Watcher
local DISPEL_GOW = {id = 4548, level = 1} -- Quest - Dispel Watcher Gaze
-- Misc
local MIN_LEVEL = 74

q:add_start_npc(WAHKAN)
q:add_talk_id(ASEFA, WAHKAN, UDANS_BOX)
q:add_attack_id(VARKA_MOBS)
q:register_quest_items(STOLEN_GREEN_TOTEM)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = nil
  if event == "31371-02.html" then
    qs:start_quest()
    htmltext = event
  elseif event == "open_box" then
    if not q:has_quest_items(player, KEY) then
      htmltext = "31561-02.html"
    elseif qs:is_cond(2) then
      if qs:is_set("spawned") then
        q:take_items(player, KEY, 1)
        htmltext = "31561-04.html"
      else
        q:give_items(player, STOLEN_GREEN_TOTEM, 1)
        q:take_items(player, KEY, 1)
        qs:set_cond(3, true)
        htmltext = "31561-03.html"
      end
    end
  elseif event == "eye_despawn" then
    npc:broadcast_say(ChatType.NPC_GENERAL, "Udan has already seen your face!")
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
    local eye = q:add_spawn(UDANS_EYE, npc)
    eye:broadcast_say(ChatType.NPC_GENERAL, "You can't avoid the eyes of Udan!")
    q:start_quest_timer("eye_despawn", 10000, eye, attacker)
  end
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = npc:id()
  if sw == WAHKAN then
    local sw2 = qs:get_state()
    if sw2 == State.CREATED then
      htmltext = player:level() >= MIN_LEVEL and (q:has_at_least_one_quest_item(player, unpack(KETRA_MARKS)) and "31371-01.htm" or "31371-00a.html") or "31371-00b.html"
    elseif sw2 == State.STARTED then
      if qs:is_cond(1) then
        htmltext = "31371-03.html"
      end
    end
  elseif sw == ASEFA then
    if qs:is_started() then
      local sw3 = qs:get_cond()
      if sw3 == 1 then
        htmltext = "31372-01.html"
        qs:set_cond(2, true)
      elseif sw3 == 2 then
        if qs:is_set("spawned") then
          qs:unset("spawned")
          npc:set_target(player)
          npc:cast(skills.get(DISPEL_GOW.id, DISPEL_GOW.level))
          htmltext = "31372-03.html"
        else
          htmltext = "31372-02.html"
        end
      elseif sw3 == 3 then
        q:give_items(player, GREEN_TOTEM, 1)
        q:give_items(player, WISDOM_STONE, 1)
        qs:exit_quest(true, true)
        htmltext = "31372-04.html"
      end
    end
  elseif sw == UDANS_BOX then
    if qs:is_cond(2) then
      htmltext = "31561-01.html"
    end
  end
  return htmltext
end)
