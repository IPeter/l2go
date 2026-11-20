-- Translated by questport from quests/Q00160_NerupasRequest (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(160, "Q00160_NerupasRequest", "Nerupa's Request")

-- NPCs
local NERUPA = 30370
local UNOREN = 30147
local CREAMEES = 30149
local JULIA = 30152
-- Items
local SILVERY_SPIDERSILK = 1026
local UNOREN_RECEIPT = 1027
local CREAMEES_TICKET = 1028
local NIGHTSHADE_LEAF = 1029
-- Reward
local LESSER_HEALING_POTION = 1060

q:register_quest_items(SILVERY_SPIDERSILK, UNOREN_RECEIPT, CREAMEES_TICKET, NIGHTSHADE_LEAF)
q:add_start_npc(NERUPA)
q:add_talk_id(NERUPA, UNOREN, CREAMEES, JULIA)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30370-04.htm" then
    st:start_quest()
    q:give_items(player, SILVERY_SPIDERSILK, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "30370-00.htm"
    elseif player:level() < 3 then
      htmltext = "30370-02.htm"
    else
      htmltext = "30370-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == NERUPA then
      if cond < 4 then
        htmltext = "30370-05.htm"
      elseif cond == 4 then
        htmltext = "30370-06.htm"
        q:take_items(player, NIGHTSHADE_LEAF, 1)
        q:reward_items(player, LESSER_HEALING_POTION, 5)
        q:add_exp_and_sp(player, 1000, 0)
        st:exit_quest(false, true)
      end
    elseif sw2 == UNOREN then
      if cond == 1 then
        htmltext = "30147-01.htm"
        st:set_cond(2, true)
        q:take_items(player, SILVERY_SPIDERSILK, 1)
        q:give_items(player, UNOREN_RECEIPT, 1)
      elseif cond == 2 then
        htmltext = "30147-02.htm"
      elseif cond == 4 then
        htmltext = "30147-03.htm"
      end
    elseif sw2 == CREAMEES then
      if cond == 2 then
        htmltext = "30149-01.htm"
        st:set_cond(3, true)
        q:take_items(player, UNOREN_RECEIPT, 1)
        q:give_items(player, CREAMEES_TICKET, 1)
      elseif cond == 3 then
        htmltext = "30149-02.htm"
      elseif cond == 4 then
        htmltext = "30149-03.htm"
      end
    elseif sw2 == JULIA then
      if cond == 3 then
        htmltext = "30152-01.htm"
        st:set_cond(4, true)
        q:take_items(player, CREAMEES_TICKET, 1)
        q:give_items(player, NIGHTSHADE_LEAF, 1)
      elseif cond == 4 then
        htmltext = "30152-02.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
