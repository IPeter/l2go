-- Translated by questport from quests/Q00320_BonesTellTheFuture (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(320, "Q00320_BonesTellTheFuture", "Bones Tell the Future")

-- Item
local BONE_FRAGMENT = 809

q:register_quest_items(BONE_FRAGMENT)
q:add_start_npc(30359) -- Kaitar
q:add_talk_id(30359)
q:add_kill_id(20517, 20518)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30359-04.htm" then
    st:start_quest()
  end
  return event
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30359-00.htm"
    elseif player:level() < 10 then
      htmltext = "30359-02.htm"
    else
      htmltext = "30359-03.htm"
    end
  elseif sw == State.STARTED then
    if st:is_cond(1) then
      htmltext = "30359-05.htm"
    else
      htmltext = "30359-06.htm"
      q:take_items(player, BONE_FRAGMENT, -1)
      q:give_adena(player, 8470, true)
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
  if rnd(100) < (npc:id() == 20517 and 18 or 20) then
    q:give_items(player, BONE_FRAGMENT, 1)
    if q:get_quest_items_count(player, BONE_FRAGMENT) < 10 then
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
