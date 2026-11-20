-- Translated by questport from quests/Q00504_CompetitionForTheBanditStronghold (Mobius CT_0 Interlude); owned by L2Go from here on.
-- @author LordWinter
-- Needs clan halls (siegable halls, M9+). The Mobius static field holding
-- the Bandit Stronghold is looked up when needed instead, so the script
-- loads before clan halls exist.
local J = require("lib.java")

local q = quest.new(504, "Q00504_CompetitionForTheBanditStronghold", "Competition for the Bandit Stronghold")
q:requires("clan_halls")

local MESSENGER = 35437
local TARLK_AMULET = 4332
local TROPHY_OF_ALLIANCE = 5009
local MOBS = {20570, 20571, 20572, 20573, 20574}

-- banditStronghold is Mobius's BANDIT_STRONGHOLD: siegable hall 35.
local function banditStronghold()
  return clan_halls.get_siegable_hall(35)
end

q:add_start_npc(MESSENGER)
q:add_talk_id(MESSENGER)
for _, mob in ipairs(MOBS) do
  q:add_kill_id(mob)
end

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  if st == nil then
    return htmltext
  end
  local sw = st:get_state()
  if sw == State.CREATED then
    if #banditStronghold():siege_attackers() >= 5 then
      htmltext = "35437-00.htm"
    else
      htmltext = "35437-01.htm"
      st:start_quest()
    end
  elseif sw == State.STARTED then
    if q:get_quest_items_count(player, TARLK_AMULET) < 30 then
      htmltext = "35437-02.htm"
    else
      q:take_items(player, TARLK_AMULET, 30)
      q:reward_items(player, TROPHY_OF_ALLIANCE, 1)
      st:exit_quest(true)
      htmltext = "35437-03.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local st = q:get_quest_state(killer, false)
  if st == nil then
    return
  end
  if st:is_started() and st:is_cond(1) then
    q:give_items(killer, TARLK_AMULET, 1)
    if q:get_quest_items_count(killer, TARLK_AMULET) < 30 then
      q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_MIDDLE)
    else
      st:set_cond(2, true)
    end
  end
end)
