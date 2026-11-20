-- Translated by questport from quests/Q00294_CovertBusiness (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(294, "Q00294_CovertBusiness", "Covert Business")

-- Item
local BAT_FANG = 1491
-- Reward
local RING_OF_RACCOON = 1508

q:register_quest_items(BAT_FANG)
q:add_start_npc(30534) -- Keef
q:add_talk_id(30534)
q:add_kill_id(20370, 20480) -- Barded Bat, Blade Bat

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30534-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DWARF then
      htmltext = "30534-00.htm"
    elseif player:level() < 10 then
      htmltext = "30534-01.htm"
    else
      htmltext = "30534-02.htm"
    end
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30534-04.htm"
    else
      q:take_items(player, BAT_FANG, -1)
      if not q:has_quest_items(player, RING_OF_RACCOON) then
        htmltext = "30534-05.htm"
        q:give_items(player, RING_OF_RACCOON, 1)
      else
        htmltext = "30534-06.htm"
        q:give_adena(player, 2400, true)
      end
      q:add_exp_and_sp(player, 0, 600)
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
  local count = 1
  local chance = rnd(10)
  local isBarded = npc:id() == 20370
  if chance < 3 then
    count = count + 1
  elseif chance < (isBarded and 5 or 6) then
    count = count + 2
  elseif isBarded and chance < 7 then
    count = count + 3
  end
  q:give_items(player, BAT_FANG, count)
  if q:get_quest_items_count(player, BAT_FANG) < 100 then
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  else
    st:set_cond(2, true)
  end
end)
