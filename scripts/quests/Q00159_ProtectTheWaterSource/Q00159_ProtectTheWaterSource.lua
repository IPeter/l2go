-- Translated by questport from quests/Q00159_ProtectTheWaterSource (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(159, "Q00159_ProtectTheWaterSource", "Protect the Water Source")

-- Items
local PLAGUE_DUST = 1035
local HYACINTH_CHARM_1 = 1071
local HYACINTH_CHARM_2 = 1072

q:register_quest_items(PLAGUE_DUST, HYACINTH_CHARM_1, HYACINTH_CHARM_2)
q:add_start_npc(30154) -- Asterios
q:add_talk_id(30154)
q:add_kill_id(27017) -- Plague Zombie

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30154-04.htm" then
    st:start_quest()
    q:give_items(player, HYACINTH_CHARM_1, 1)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ELF then
      htmltext = "30154-00.htm"
    elseif player:level() < 12 then
      htmltext = "30154-02.htm"
    else
      htmltext = "30154-03.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "30154-05.htm"
    elseif cond == 2 then
      htmltext = "30154-06.htm"
      st:set_cond(3, true)
      q:take_items(player, PLAGUE_DUST, -1)
      q:take_items(player, HYACINTH_CHARM_1, 1)
      q:give_items(player, HYACINTH_CHARM_2, 1)
    elseif cond == 3 then
      htmltext = "30154-07.htm"
    elseif cond == 4 then
      htmltext = "30154-08.htm"
      q:take_items(player, HYACINTH_CHARM_2, 1)
      q:take_items(player, PLAGUE_DUST, -1)
      q:give_adena(player, 18250, true)
      st:exit_quest(false, true)
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local qs = q:get_quest_state(killer, false)
  if qs ~= nil then
    local sw = qs:get_cond()
    if sw == 1 then
      if rnd(100) < 40 and q:has_quest_items(killer, HYACINTH_CHARM_1) and not q:has_quest_items(killer, PLAGUE_DUST) then
        q:give_items(killer, PLAGUE_DUST, 1)
        qs:set_cond(2, true)
      end
    elseif sw == 3 then
      local dust = q:get_quest_items_count(killer, PLAGUE_DUST)
      if rnd(100) < 40 and dust < 5 and q:has_quest_items(killer, HYACINTH_CHARM_2) then
        q:give_items(killer, PLAGUE_DUST, 1)
        dust = dust + 1
        if dust >= 5 then
          qs:set_cond(4, true)
        else
          q:play_sound(killer, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  end
end)
