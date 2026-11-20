-- Translated by questport from quests/Q00165_ShilensHunt (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(165, "Q00165_ShilensHunt", "Shilen's Hunt")

-- NPCs
local MONSTERS = {}
-- Items
local DARK_BEZOAR = 1160
local LESSER_HEALING_POTION = 1060

MONSTERS[20456] = 3 -- Ashen Wolf
MONSTERS[20529] = 1 -- Young Brown Keltir
MONSTERS[20532] = 1 -- Brown Keltir
MONSTERS[20536] = 2 -- Elder Brown Keltir

q:register_quest_items(DARK_BEZOAR)
q:add_start_npc(30348) -- Nelsya
q:add_talk_id(30348)
q:add_kill_id(J.keys(MONSTERS))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30348-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30348-00.htm"
    elseif player:level() < 3 then
      htmltext = "30348-01.htm"
    else
      htmltext = "30348-02.htm"
    end
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, DARK_BEZOAR) >= 13 then
      htmltext = "30348-05.htm"
      q:take_items(player, DARK_BEZOAR, -1)
      q:reward_items(player, LESSER_HEALING_POTION, 5)
      q:add_exp_and_sp(player, 1000, 0)
      st:exit_quest(false, true)
    else
      htmltext = "30348-04.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil and qs:is_cond(1) and rnd(3) < MONSTERS[npc:id()] then
    q:give_items(killer, DARK_BEZOAR, 1)
    if q:get_quest_items_count(killer, DARK_BEZOAR) < 13 then
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      qs:set_cond(2, true)
    end
  end
end)
