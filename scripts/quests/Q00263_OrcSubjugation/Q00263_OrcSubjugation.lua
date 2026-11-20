-- Translated by questport from quests/Q00263_OrcSubjugation (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(263, "Q00263_OrcSubjugation", "Orc Subjugation")

-- Items
local ORC_AMULET = 1116
local ORC_NECKLACE = 1117

q:register_quest_items(ORC_AMULET, ORC_NECKLACE)
q:add_start_npc(30346) -- Kayleen
q:add_talk_id(30346)
q:add_kill_id(20385, 20386, 20387, 20388)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30346-03.htm" then
    st:start_quest()
  elseif event == "30346-06.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30346-00.htm"
    elseif player:level() < 8 then
      htmltext = "30346-01.htm"
    else
      htmltext = "30346-02.htm"
    end
  elseif sw == State.STARTED then
    local amulet = q:get_quest_items_count(player, ORC_AMULET)
    local necklace = q:get_quest_items_count(player, ORC_NECKLACE)
    if amulet == 0 and necklace == 0 then
      htmltext = "30346-04.htm"
    else
      htmltext = "30346-05.htm"
      q:take_items(player, ORC_AMULET, -1)
      q:take_items(player, ORC_NECKLACE, -1)
      local reward = amulet * 20 + necklace * 30
      if not Config.ALT_VILLAGES_REPEATABLE_QUEST_REWARD and amulet + necklace >= 10 then
        reward = reward + 1000
      end
      q:give_adena(player, reward, true)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_cond(1) then
    return
  end
  if rnd.bool() then
    q:give_items(player, npc:id() == 20385 and ORC_AMULET or ORC_NECKLACE, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
