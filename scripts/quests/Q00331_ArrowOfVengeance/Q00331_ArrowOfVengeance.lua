-- Translated by questport from quests/Q00331_ArrowOfVengeance (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(331, "Q00331_ArrowOfVengeance", "Arrow of Vengeance")

-- Items
local HARPY_FEATHER = 1452
local MEDUSA_VENOM = 1453
local WYRM_TOOTH = 1454

q:register_quest_items(HARPY_FEATHER, MEDUSA_VENOM, WYRM_TOOTH)
q:add_start_npc(30125) -- Belton
q:add_talk_id(30125)
q:add_kill_id(20145, 20158, 20176)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30125-03.htm" then
    st:start_quest()
  elseif event == "30125-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 32 and "30125-01.htm" or "30125-02.htm"
  elseif sw == State.STARTED then
    local harpyFeather = q:get_quest_items_count(player, HARPY_FEATHER)
    local medusaVenom = q:get_quest_items_count(player, MEDUSA_VENOM)
    local wyrmTooth = q:get_quest_items_count(player, WYRM_TOOTH)
    if harpyFeather + medusaVenom + wyrmTooth > 0 then
      htmltext = "30125-05.htm"
      q:take_items(player, HARPY_FEATHER, -1)
      q:take_items(player, MEDUSA_VENOM, -1)
      q:take_items(player, WYRM_TOOTH, -1)
      local reward = harpyFeather * 80 + medusaVenom * 90 + wyrmTooth * 100
      if harpyFeather + medusaVenom + wyrmTooth > 10 then
        reward = reward + 3100
      end
      q:give_adena(player, reward, true)
    else
      htmltext = "30125-04.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if rnd.bool() then
    local sw = npc:id()
    if sw == 20145 then
      q:give_items(player, HARPY_FEATHER, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    elseif sw == 20158 then
      q:give_items(player, MEDUSA_VENOM, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    elseif sw == 20176 then
      q:give_items(player, WYRM_TOOTH, 1)
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
end)
