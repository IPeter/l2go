-- Translated by questport from quests/Q00003_WillTheSealBeBroken (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Will the Seal be Broken? (3)
-- @author malyelfik
local J = require("lib.java")

local q = quest.new(3, "Q00003_WillTheSealBeBroken", "Will the Seal be Broken?")

-- NPC
local TALLOTH = 30141
-- Monsters
local OMEN_BEAST = 20031
local TAINTED_ZOMBIE = 20041
local STINK_ZOMBIE = 20046
local LESSER_SUCCUBUS = 20048
local LESSER_SUCCUBUS_TUREN = 20052
local LESSER_SUCCUBUS_TILFO = 20057
-- Items
local OMEN_BEAST_EYE = 1081
local TAINT_STONE = 1082
local SUCCUBUS_BLOOD = 1083
local ENCHANT = 956
-- Misc
local MIN_LEVEL = 16

local function giveItem(player, qs, item, items_)
  if not q:has_quest_items(player, item) then
    q:give_items(player, item, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    if q:has_quest_items(player, unpack(items_)) then
      qs:set_cond(2, true)
    end
  end
end

q:add_start_npc(TALLOTH)
q:add_talk_id(TALLOTH)
q:add_kill_id(OMEN_BEAST, TAINTED_ZOMBIE, STINK_ZOMBIE, LESSER_SUCCUBUS, LESSER_SUCCUBUS_TILFO, LESSER_SUCCUBUS_TUREN)
q:register_quest_items(OMEN_BEAST_EYE, TAINT_STONE, SUCCUBUS_BLOOD)

q:on_event(function(event, npc, player)
  local qs = q:get_quest_state(player, false)
  if qs == nil then
    return nil
  end
  local htmltext = event
  if event == "30141-03.htm" then
    qs:start_quest()
  elseif event == "30141-05.html" then
  else
    htmltext = nil
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local qs = q:get_quest_state(player, true)
  local htmltext = q:get_no_quest_msg(player)
  local sw = qs:get_state()
  if sw == State.CREATED then
    htmltext = player:race() ~= Race.DARK_ELF and "30141-00.htm" or player:level() >= MIN_LEVEL and "30141-02.htm" or "30141-01.html"
  elseif sw == State.STARTED then
    if qs:is_cond(1) then
      htmltext = "30141-04.html"
    else
      q:give_items(player, ENCHANT, 1)
      qs:exit_quest(false, true)
      htmltext = "30141-06.html"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isSummon)
  local member = q:get_random_party_member(player, 1)
  if member == nil then
    return
  end
  local qs = q:get_quest_state(member, false)
  local sw = npc:id()
  if sw == OMEN_BEAST then
    giveItem(member, qs, OMEN_BEAST_EYE, q:get_registered_item_ids())
  elseif sw == STINK_ZOMBIE or sw == TAINTED_ZOMBIE then
    giveItem(member, qs, TAINT_STONE, q:get_registered_item_ids())
  elseif sw == LESSER_SUCCUBUS or sw == LESSER_SUCCUBUS_TILFO or sw == LESSER_SUCCUBUS_TUREN then
    giveItem(member, qs, SUCCUBUS_BLOOD, q:get_registered_item_ids())
  end
end)
