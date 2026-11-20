-- Translated by questport from quests/Q00274_SkirmishWithTheWerewolves (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(274, "Q00274_SkirmishWithTheWerewolves", "Skirmish with the Werewolves")

-- Needed items
local NECKLACE_OF_VALOR = 1507
local NECKLACE_OF_COURAGE = 1506
-- Items
local MARAKU_WEREWOLF_HEAD = 1477
local MARAKU_WOLFMEN_TOTEM = 1501

q:register_quest_items(MARAKU_WEREWOLF_HEAD, MARAKU_WOLFMEN_TOTEM)
q:add_start_npc(30569)
q:add_talk_id(30569)
q:add_kill_id(20363, 20364)

q:on_event(function(event, npc, player)
  local st = player:get_quest_state(q:name())
  local htmltext = event
  if st == nil then
    return htmltext
  end
  if event == "30569-03.htm" then
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
      htmltext = "30569-00.htm"
    elseif player:level() < 9 then
      htmltext = "30569-01.htm"
    elseif q:has_at_least_one_quest_item(player, NECKLACE_OF_COURAGE, NECKLACE_OF_VALOR) then
      htmltext = "30569-02.htm"
    else
      htmltext = "30569-07.htm"
    end
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30569-04.htm"
    else
      htmltext = "30569-05.htm"
      local amount = 3500 + q:get_quest_items_count(player, MARAKU_WOLFMEN_TOTEM) * 600
      q:take_items(player, MARAKU_WEREWOLF_HEAD, -1)
      q:take_items(player, MARAKU_WOLFMEN_TOTEM, -1)
      q:give_adena(player, amount, true)
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
  q:give_items(player, MARAKU_WEREWOLF_HEAD, 1)
  if q:get_quest_items_count(player, MARAKU_WEREWOLF_HEAD) < 40 then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  else
    st:set_cond(2, true)
  end
  if rnd(100) < 6 then
    q:give_items(player, MARAKU_WOLFMEN_TOTEM, 1)
  end
end)
