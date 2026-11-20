-- Translated by questport from quests/Q00636_TruthBeyondTheGate (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author Mobius, Skache
-- @note Based on python script
local J = require("lib.java")

local q = quest.new(636, "Q00636_TruthBeyondTheGate", "Truth Beyond the Gate")

-- NPCs
local ELIYAH = 31329
local FLAURON = 32010
-- Item
local VISITOR_MARK = 8064
local FADED_VISITOR_MARK = 8065

q:add_start_npc(ELIYAH)
q:add_talk_id(ELIYAH, FLAURON)
q:add_enter_zone_id(30100)

q:on_event(function(event, npc, player)
  local htmltext = event
  local qs = player:get_quest_state(q:name())
  if qs == nil then
    return htmltext
  end
  if event == "31329-04.htm" then
    qs:start_quest()
  elseif event == "32010-02.htm" then
    q:give_items(player, VISITOR_MARK, 1)
    qs:unset("cond")
    qs:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local qs = q:get_quest_state(player, true)
  local npcId = npc:id()
  local id = qs:get_state()
  local cond = qs:get_cond()
  if cond == 0 and id == State.CREATED then
    if npcId == ELIYAH then
      if player:level() > 72 then
        htmltext = "31329-02.htm"
      else
        htmltext = "31329-01.htm"
        qs:exit_quest(true)
      end
    end
  elseif id == State.STARTED then
    if npcId == ELIYAH then
      htmltext = "31329-05.htm"
    elseif npcId == FLAURON then
      if cond == 1 then
        htmltext = "32010-01.htm"
        qs:set_cond(2)
      else
        htmltext = "32010-03.htm"
      end
    end
  end
  return htmltext
end)

-- Entering the Pagan Temple fades the Visitor's Mark.
q:on_enter_zone(function(creature, zone)
  -- Mobius creature.isPlayer()
  if J.instance_of(creature, "Player") then
    local player = creature
    -- Mobius getInventory().getItemByItemId(VISITOR_MARK) != null, then
    -- destroyItemByItemId(VISITOR_MARK, 1), which is true when it had one.
    if player:item_count(VISITOR_MARK) > 0 then
      if player:take_items(VISITOR_MARK, 1) then
        q:give_items(player, FADED_VISITOR_MARK, 1)
      end
    end
  end
end)
