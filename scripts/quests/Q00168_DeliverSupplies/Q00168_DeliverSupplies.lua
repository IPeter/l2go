-- Translated by questport from quests/Q00168_DeliverSupplies (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(168, "Q00168_DeliverSupplies", "Deliver Supplies")

-- NPCs
local JENNA = 30349
local ROSELYN = 30355
local KRISTIN = 30357
local HARANT = 30360
-- Items
local JENNA_LETTER = 1153
local SENTRY_BLADE_1 = 1154
local SENTRY_BLADE_2 = 1155
local SENTRY_BLADE_3 = 1156
local OLD_BRONZE_SWORD = 1157

q:register_quest_items(JENNA_LETTER, SENTRY_BLADE_1, SENTRY_BLADE_2, SENTRY_BLADE_3, OLD_BRONZE_SWORD)
q:add_start_npc(JENNA)
q:add_talk_id(JENNA, ROSELYN, KRISTIN, HARANT)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30349-03.htm" then
    st:start_quest()
    q:give_items(player, JENNA_LETTER, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.DARK_ELF then
      htmltext = "30349-00.htm"
    elseif player:level() < 3 then
      htmltext = "30349-01.htm"
    else
      htmltext = "30349-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == JENNA then
      if cond == 1 then
        htmltext = "30349-04.htm"
      elseif cond == 2 then
        htmltext = "30349-05.htm"
        st:set_cond(3, true)
        q:take_items(player, SENTRY_BLADE_1, 1)
      elseif cond == 3 then
        htmltext = "30349-07.htm"
      elseif cond == 4 then
        htmltext = "30349-06.htm"
        q:take_items(player, OLD_BRONZE_SWORD, 2)
        q:give_adena(player, 820, true)
        st:exit_quest(false, true)
      end
    elseif sw2 == HARANT then
      if cond == 1 then
        htmltext = "30360-01.htm"
        st:set_cond(2, true)
        q:take_items(player, JENNA_LETTER, 1)
        q:give_items(player, SENTRY_BLADE_1, 1)
        q:give_items(player, SENTRY_BLADE_2, 1)
        q:give_items(player, SENTRY_BLADE_3, 1)
      elseif cond == 2 then
        htmltext = "30360-02.htm"
      end
    elseif sw2 == ROSELYN then
      if cond == 3 then
        if q:has_quest_items(player, SENTRY_BLADE_2) then
          htmltext = "30355-01.htm"
          q:take_items(player, SENTRY_BLADE_2, 1)
          q:give_items(player, OLD_BRONZE_SWORD, 1)
          if q:get_quest_items_count(player, OLD_BRONZE_SWORD) == 2 then
            st:set_cond(4, true)
          end
        else
          htmltext = "30355-02.htm"
        end
      elseif cond == 4 then
        htmltext = "30355-02.htm"
      end
    elseif sw2 == KRISTIN then
      if cond == 3 then
        if q:has_quest_items(player, SENTRY_BLADE_3) then
          htmltext = "30357-01.htm"
          q:take_items(player, SENTRY_BLADE_3, 1)
          q:give_items(player, OLD_BRONZE_SWORD, 1)
          if q:get_quest_items_count(player, OLD_BRONZE_SWORD) == 2 then
            st:set_cond(4, true)
          end
        else
          htmltext = "30357-02.htm"
        end
      elseif cond == 4 then
        htmltext = "30357-02.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
