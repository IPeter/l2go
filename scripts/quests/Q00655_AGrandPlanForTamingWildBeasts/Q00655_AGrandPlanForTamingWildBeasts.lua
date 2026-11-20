-- Translated by questport from quests/Q00655_AGrandPlanForTamingWildBeasts (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author LordWinter
local J = require("lib.java")

local q = quest.new(655, "Q00655_AGrandPlanForTamingWildBeasts", "A Grand Plan for Taming Wild Beasts")
q:requires("clan_halls")

local MESSENGER = 35627
local STONE = 8084
local TRAINER_LICENSE = 8293
-- The Wild Beast Reserve (siegable clan hall 63).
local BEAST_STRONGHOLD_ID = 63

-- Mobius's static helper, called by the Wild Beast Reserve siege script
-- (not ported yet) when a Crystal of Purity is picked up.
local function checkCrystalofPurity(player)
  local st = player:get_quest_state("Q00655_AGrandPlanForTamingWildBeasts")
  if st ~= nil and st:is_cond(1) and q:get_quest_items_count(player, STONE) < 10 then
    q:give_items(player, STONE, 1)
  end
end

q:add_start_npc(MESSENGER)
q:add_talk_id(MESSENGER)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  if st == nil then
    return htmltext
  end
  local sw = st:get_state()
  if sw == State.CREATED then
    -- Mobius looks the hall up once in a static field; clan_halls does not
    -- exist before the clan hall milestone (q:requires above), so the
    -- lookup happens here, where the quest is never reached until then.
    local hall = clan_halls.get_siegable_hall(BEAST_STRONGHOLD_ID)
    if #hall:siege_attackers() >= 5 then
      htmltext = "35627-00.htm"
    else
      htmltext = "35627-01.htm"
      st:start_quest()
    end
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, STONE) < 10 then
      htmltext = "35627-02.htm"
    else
      q:take_items(player, STONE, 10)
      q:give_items(player, TRAINER_LICENSE, 1)
      st:exit_quest(true, true)
      htmltext = "35627-03.htm"
    end
  end
  return htmltext
end)
