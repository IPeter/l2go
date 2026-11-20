-- Translated by questport from quests/Q00431_WeddingMarch (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(431, "Q00431_WeddingMarch", "Wedding March")

-- NPC
local KANTABILON = 31042
-- Item
local SILVER_CRYSTAL = 7540
-- Reward
local WEDDING_ECHO_CRYSTAL = 7062

q:register_quest_items(SILVER_CRYSTAL)
q:add_start_npc(KANTABILON)
q:add_talk_id(KANTABILON)
q:add_kill_id(20786, 20787)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31042-02.htm" then
    st:start_quest()
  elseif event == "31042-05.htm" then
    if q:get_quest_items_count(player, SILVER_CRYSTAL) < 50 then
      htmltext = "31042-03.htm"
    else
      q:take_items(player, SILVER_CRYSTAL, -1)
      q:give_items(player, WEDDING_ECHO_CRYSTAL, 25)
      st:exit_quest(true, true)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 38 and "31042-00.htm" or "31042-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "31042-02.htm"
    elseif cond == 2 then
      htmltext = q:get_quest_items_count(player, SILVER_CRYSTAL) < 50 and "31042-03.htm" or "31042-04.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local qs = q:get_random_party_member_state(player, 1, 3, npc)
  if qs == nil then
    return
  end
  local partyMember = qs:get_player()
  local st = q:get_quest_state(partyMember, false)
  if st == nil then
    return
  end
  if rnd.bool() then
    q:give_items(partyMember, SILVER_CRYSTAL, 1)
    if q:get_quest_items_count(partyMember, SILVER_CRYSTAL) < 50 then
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    else
      st:set_cond(2, true)
    end
  end
end)
