-- Translated by questport from quests/Q00247_PossessorOfAPreciousSoul4 (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Possessor Of A PreciousSoul part 4 (247)
local J = require("lib.java")

local q = quest.new(247, "Q00247_PossessorOfAPreciousSoul4", "Possessor of a Precious Soul - 4")
q:requires("subclass")

-- NPCs
local CARADINE = 31740
local LADY_OF_THE_LAKE = 31745
-- Items
local CARADINE_LETTER = 7679
local NOBLESS_TIARA = 7694

q:add_start_npc(CARADINE)
q:add_talk_id(CARADINE, LADY_OF_THE_LAKE)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31740-03.htm" then
    repeat
      if not st:is_created() or not player:is_sub_class_active() or player:level() < 75 or not q:has_quest_items(player, CARADINE_LETTER) then
        break
      end
      st:start_quest()
      q:take_items(player, CARADINE_LETTER, 1)
    until true
  elseif event == "TELEPORT" then
    repeat
      if not st:is_started() then
        break
      end
      st:set_cond(2)
      player:teleport(143209, 43968, -3038)
    until true
  elseif event == "31745-05.htm" then
    repeat
      if not st:is_started() or not st:is_cond(2) or not player:is_sub_class_active() or player:level() < 75 then
        break
      end
      player:set_noble(true)
      q:give_items(player, NOBLESS_TIARA, 1)
      q:add_exp_and_sp(player, 93836, 0)
      player:social_action(3)
      st:exit_quest(false, true)
    until true
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if q:has_quest_items(player, CARADINE_LETTER) then
      htmltext = (not player:is_sub_class_active() or player:level() < 75) and "31740-02.htm" or "31740-01.htm"
    end
  elseif sw == State.STARTED then
    repeat
      if not player:is_sub_class_active() then
        break
      end
      local cond = st:get_cond()
      local sw2 = npc:id()
      if sw2 == CARADINE then
        if cond == 1 then
          htmltext = "31740-04.htm"
        elseif cond == 2 then
          htmltext = "31740-06.htm"
        end
      elseif sw2 == LADY_OF_THE_LAKE then
        if cond == 2 then
          htmltext = player:level() < 75 and "31745-06.htm" or "31745-01.htm"
        end
      end
    until true
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
