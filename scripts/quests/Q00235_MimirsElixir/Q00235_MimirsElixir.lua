-- Translated by questport from quests/Q00235_MimirsElixir (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(235, "Q00235_MimirsElixir", "Mimir's Elixir")

-- NPCs
local JOAN = 30718
local LADD = 30721
local MIXING_URN = 31149
-- Items
local STAR_OF_DESTINY = 5011
local PURE_SILVER = 6320
local TRUE_GOLD = 6321
local SAGE_STONE = 6322
local BLOOD_FIRE = 6318
local MIMIR_ELIXIR = 6319
local MAGISTER_MIXING_STONE = 5905
-- Reward
local SCROLL_ENCHANT_WEAPON_A = 729

q:register_quest_items(PURE_SILVER, TRUE_GOLD, SAGE_STONE, BLOOD_FIRE, MAGISTER_MIXING_STONE, MIMIR_ELIXIR)
q:add_start_npc(LADD)
q:add_talk_id(LADD, JOAN, MIXING_URN)
q:add_kill_id(20965, 21090)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30721-06.htm" then
    st:start_quest()
  elseif event == "30721-12.htm" then
    if q:has_quest_items(player, TRUE_GOLD) then
      st:set_cond(6, true)
      q:give_items(player, MAGISTER_MIXING_STONE, 1)
    end
  elseif event == "30721-16.htm" then
    if q:has_quest_items(player, MIMIR_ELIXIR) then
      player:magic_skill_use(player, 4339, 1, 1, 1)
      q:take_items(player, MAGISTER_MIXING_STONE, -1)
      q:take_items(player, MIMIR_ELIXIR, -1)
      q:take_items(player, STAR_OF_DESTINY, -1)
      q:give_items(player, SCROLL_ENCHANT_WEAPON_A, 1)
      player:social_action(3)
      st:exit_quest(false, true)
    end
  elseif event == "30718-03.htm" then
    st:set_cond(3, true)
  elseif event == "31149-02.htm" then
    if not q:has_quest_items(player, MAGISTER_MIXING_STONE) then
      htmltext = "31149-havent.htm"
    end
  elseif event == "31149-03.htm" then
    if not q:has_quest_items(player, MAGISTER_MIXING_STONE, PURE_SILVER) then
      htmltext = "31149-havent.htm"
    end
  elseif event == "31149-05.htm" then
    if not q:has_quest_items(player, MAGISTER_MIXING_STONE, PURE_SILVER, TRUE_GOLD) then
      htmltext = "31149-havent.htm"
    end
  elseif event == "31149-07.htm" then
    if not q:has_quest_items(player, MAGISTER_MIXING_STONE, PURE_SILVER, TRUE_GOLD, BLOOD_FIRE) then
      htmltext = "31149-havent.htm"
    end
  elseif event == "31149-success.htm" then
    if q:has_quest_items(player, MAGISTER_MIXING_STONE, PURE_SILVER, TRUE_GOLD, BLOOD_FIRE) then
      st:set_cond(8, true)
      q:take_items(player, PURE_SILVER, -1)
      q:take_items(player, TRUE_GOLD, -1)
      q:take_items(player, BLOOD_FIRE, -1)
      q:give_items(player, MIMIR_ELIXIR, 1)
    else
      htmltext = "31149-havent.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:level() < 75 then
      htmltext = "30721-01b.htm"
    elseif not q:has_quest_items(player, STAR_OF_DESTINY) then
      htmltext = "30721-01a.htm"
    else
      htmltext = "30721-01.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == LADD then
      if cond == 1 then
        if q:has_quest_items(player, PURE_SILVER) then
          htmltext = "30721-08.htm"
          st:set_cond(2, true)
        else
          htmltext = "30721-07.htm"
        end
      elseif cond < 5 then
        htmltext = "30721-10.htm"
      elseif cond == 5 and q:has_quest_items(player, TRUE_GOLD) then
        htmltext = "30721-11.htm"
      elseif cond == 6 or cond == 7 then
        htmltext = "30721-13.htm"
      elseif cond == 8 and q:has_quest_items(player, MIMIR_ELIXIR) then
        htmltext = "30721-14.htm"
      end
    elseif sw2 == JOAN then
      if cond == 2 then
        htmltext = "30718-01.htm"
      elseif cond == 3 then
        htmltext = "30718-04.htm"
      elseif cond == 4 and q:has_quest_items(player, SAGE_STONE) then
        htmltext = "30718-05.htm"
        st:set_cond(5, true)
        q:take_items(player, SAGE_STONE, -1)
        q:give_items(player, TRUE_GOLD, 1)
      elseif cond > 4 then
        htmltext = "30718-06.htm"
      end
    elseif sw2 == MIXING_URN then
      -- The urn gives the same first htm. Bypasses' events will do all the job.
      htmltext = "31149-01.htm"
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local sw = npc:id()
  if sw == 20965 then
    if st:is_cond(3) and rnd(10) < 2 then
      q:give_items(player, SAGE_STONE, 1)
      st:set_cond(4, true)
    end
  elseif sw == 21090 then
    if st:is_cond(6) and rnd(10) < 2 then
      q:give_items(player, BLOOD_FIRE, 1)
      st:set_cond(7, true)
    end
  end
end)
