-- Translated by questport from quests/Q00276_TotemOfTheHestui (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(276, "Q00276_TotemOfTheHestui", "Totem of the Hestui")

-- Items
local KASHA_PARASITE = 1480
local KASHA_CRYSTAL = 1481
-- Rewards
local HESTUI_TOTEM = 1500
local LEATHER_PANTS = 29

q:register_quest_items(KASHA_PARASITE, KASHA_CRYSTAL)
q:add_start_npc(30571) -- Tanapi
q:add_talk_id(30571)
q:add_kill_id(20479, 27044)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30571-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ORC then
      htmltext = "30571-00.htm"
    elseif player:level() < 15 then
      htmltext = "30571-01.htm"
    else
      htmltext = "30571-02.htm"
    end
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30571-04.htm"
    else
      htmltext = "30571-05.htm"
      q:take_items(player, KASHA_CRYSTAL, -1)
      q:take_items(player, KASHA_PARASITE, -1)
      q:give_items(player, HESTUI_TOTEM, 1)
      q:give_items(player, LEATHER_PANTS, 1)
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if not q:has_quest_items(player, KASHA_CRYSTAL) then
    local sw = npc:id()
    if sw == 20479 then
      local count = q:get_quest_items_count(player, KASHA_PARASITE)
      local random = rnd(100)
      if count >= 79 or count >= 69 and random <= 20 or count >= 59 and random <= 15 or count >= 49 and random <= 10 or count >= 39 and random < 2 then
        q:add_spawn(27044, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 0)
        q:take_items(player, KASHA_PARASITE, count)
      else
        q:give_items(player, KASHA_PARASITE, 1)
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      end
    elseif sw == 27044 then
      st:set_cond(2, true)
      q:give_items(player, KASHA_CRYSTAL, 1)
    end
  end
end)
