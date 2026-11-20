-- Translated by questport from quests/Q00352_HelpRoodRaiseANewPet (Mobius CT_0 Interlude); owned by L2Go from here on.
-- The quest only collects eggs for adena; it needs no pet system.
local J = require("lib.java")

local q = quest.new(352, "Q00352_HelpRoodRaiseANewPet", "Help Rood Raise A New Pet!")

-- Items
local LIENRIK_EGG_1 = 5860
local LIENRIK_EGG_2 = 5861

q:register_quest_items(LIENRIK_EGG_1, LIENRIK_EGG_2)
q:add_start_npc(31067) -- Rood
q:add_talk_id(31067)
q:add_kill_id(20786, 20787, 21644, 21645)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31067-04.htm" then
    st:start_quest()
  elseif event == "31067-09.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 39 and "31067-00.htm" or "31067-01.htm"
  elseif sw == State.STARTED then
    local eggs1 = q:get_quest_items_count(player, LIENRIK_EGG_1)
    local eggs2 = q:get_quest_items_count(player, LIENRIK_EGG_2)
    if eggs1 + eggs2 == 0 then
      htmltext = "31067-05.htm"
    else
      local reward = 2000
      if eggs1 > 0 and eggs2 == 0 then
        htmltext = "31067-06.htm"
        reward = reward + eggs1 * 34
        q:take_items(player, LIENRIK_EGG_1, -1)
        q:give_adena(player, reward, true)
      elseif eggs1 == 0 and eggs2 > 0 then
        htmltext = "31067-08.htm"
        reward = reward + eggs2 * 1025
        q:take_items(player, LIENRIK_EGG_2, -1)
        q:give_adena(player, reward, true)
      elseif eggs1 > 0 and eggs2 > 0 then
        htmltext = "31067-08.htm"
        reward = reward + (eggs1 * 34 + eggs2 * 1025 + 2000)
        q:take_items(player, LIENRIK_EGG_1, -1)
        q:take_items(player, LIENRIK_EGG_2, -1)
        q:give_adena(player, reward, true)
      end
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local npcId = npc:id()
  local random = rnd(100)
  local chance = (npcId == 20786 or npcId == 21644) and 44 or 58
  if random < chance then
    q:give_items(player, LIENRIK_EGG_1, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  elseif random < chance + 4 then
    q:give_items(player, LIENRIK_EGG_2, 1)
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
