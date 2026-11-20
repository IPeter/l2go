-- Translated by questport from quests/Q00688_DefeatTheElrokianRaiders (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(688, "Q00688_DefeatTheElrokianRaiders", "Defeat the Elrokian Raiders!")

-- NPC
local DINN = 32105
-- Monster
local ELROKI = 22214
-- Item
local DINOSAUR_FANG_NECKLACE = 8785

q:register_quest_items(DINOSAUR_FANG_NECKLACE)
q:add_start_npc(DINN)
q:add_talk_id(DINN)
q:add_kill_id(ELROKI)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "32105-03.htm" then
    st:start_quest()
  elseif event == "32105-08.htm" then
    local count = q:get_quest_items_count(player, DINOSAUR_FANG_NECKLACE)
    if count > 0 then
      q:take_items(player, DINOSAUR_FANG_NECKLACE, -1)
      q:give_adena(player, count * 3000, true)
    end
    st:exit_quest(true, true)
  elseif event == "32105-06.htm" then
    local count = q:get_quest_items_count(player, DINOSAUR_FANG_NECKLACE)
    q:take_items(player, DINOSAUR_FANG_NECKLACE, -1)
    q:give_adena(player, count * 3000, true)
  elseif event == "32105-07.htm" then
    local count = q:get_quest_items_count(player, DINOSAUR_FANG_NECKLACE)
    if count >= 100 then
      q:take_items(player, DINOSAUR_FANG_NECKLACE, 100)
      q:give_adena(player, 450000, true)
    else
      htmltext = "32105-04.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 75 and "32105-00.htm" or "32105-01.htm"
  elseif sw == State.STARTED then
    htmltext = not q:has_quest_items(player, DINOSAUR_FANG_NECKLACE) and "32105-04.htm" or "32105-05.htm"
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil or not st:is_started() then
    return
  end
  if rnd.bool() then
    local partyMember = st:get_player()
    q:give_items(partyMember, DINOSAUR_FANG_NECKLACE, 1)
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
