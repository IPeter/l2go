-- Translated by questport from quests/Q00154_SacrificeToTheSea (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(154, "Q00154_SacrificeToTheSea", "Sacrifice to the Sea")

-- NPCs
local ROCKSWELL = 30312
local CRISTEL = 30051
local ROLFE = 30055
-- Items
local FOX_FUR = 1032
local FOX_FUR_YARN = 1033
local MAIDEN_DOLL = 1034
-- Reward
local EARING = 113

q:register_quest_items(FOX_FUR, FOX_FUR_YARN, MAIDEN_DOLL)
q:add_start_npc(ROCKSWELL)
q:add_talk_id(ROCKSWELL, CRISTEL, ROLFE)
q:add_kill_id(20481, 20544, 20545) -- Following Keltirs can be found near Talking Island.

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30312-04.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 2 and "30312-02.htm" or "30312-03.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == ROCKSWELL then
      if cond == 1 then
        htmltext = "30312-05.htm"
      elseif cond == 2 then
        htmltext = "30312-08.htm"
      elseif cond == 3 then
        htmltext = "30312-06.htm"
      elseif cond == 4 then
        htmltext = "30312-07.htm"
        q:take_items(player, MAIDEN_DOLL, -1)
        q:give_items(player, EARING, 1)
        q:add_exp_and_sp(player, 100, 0)
        st:exit_quest(false, true)
      end
    elseif sw2 == CRISTEL then
      if cond == 1 then
        htmltext = q:has_quest_items(player, FOX_FUR) and "30051-01.htm" or "30051-01a.htm"
      elseif cond == 2 then
        htmltext = "30051-02.htm"
        st:set_cond(3, true)
        q:take_items(player, FOX_FUR, -1)
        q:give_items(player, FOX_FUR_YARN, 1)
      elseif cond == 3 then
        htmltext = "30051-03.htm"
      elseif cond == 4 then
        htmltext = "30051-04.htm"
      end
    elseif sw2 == ROLFE then
      if cond < 3 then
        htmltext = "30055-03.htm"
      elseif cond == 3 then
        htmltext = "30055-01.htm"
        st:set_cond(4, true)
        q:take_items(player, FOX_FUR_YARN, 1)
        q:give_items(player, MAIDEN_DOLL, 1)
      elseif cond == 4 then
        htmltext = "30055-02.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_random_party_member_state(killer, 1, 3, npc)
  if qs ~= nil and q:give_item_randomly(qs:get_player(), npc, FOX_FUR, 1, 1, 10, 0.4, true) then
    qs:set_cond(2)
  end
end)
