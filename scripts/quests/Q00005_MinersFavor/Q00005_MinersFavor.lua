-- Translated by questport from quests/Q00005_MinersFavor (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(5, "Q00005_MinersFavor", "Miner's Favor")

-- NPCs
local BOLTER = 30554
local SHARI = 30517
local GARITA = 30518
local REED = 30520
local BRUNON = 30526
-- Items
local BOLTERS_LIST = 1547
local MINING_BOOTS = 1548
local MINERS_PICK = 1549
local BOOMBOOM_POWDER = 1550
local REDSTONE_BEER = 1551
local BOLTERS_SMELLY_SOCKS = 1552
-- Reward
local NECKLACE = 906

q:register_quest_items(BOLTERS_LIST, MINING_BOOTS, MINERS_PICK, BOOMBOOM_POWDER, REDSTONE_BEER, BOLTERS_SMELLY_SOCKS)
q:add_start_npc(BOLTER)
q:add_talk_id(BOLTER, SHARI, GARITA, REED, BRUNON)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30554-03.htm" then
    st:start_quest()
    q:give_items(player, BOLTERS_LIST, 1)
    q:give_items(player, BOLTERS_SMELLY_SOCKS, 1)
  elseif event == "30526-02.htm" then
    q:take_items(player, BOLTERS_SMELLY_SOCKS, 1)
    q:give_items(player, MINERS_PICK, 1)
    if q:has_quest_items(player, MINING_BOOTS, BOOMBOOM_POWDER, REDSTONE_BEER) then
      st:set_cond(2, true)
    else
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 2 and "30554-01.htm" or "30554-02.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local sw2 = npc:id()
    if sw2 == BOLTER then
      if cond == 1 then
        htmltext = "30554-04.htm"
      elseif cond == 2 then
        htmltext = "30554-06.htm"
        q:take_items(player, BOLTERS_LIST, 1)
        q:take_items(player, BOOMBOOM_POWDER, 1)
        q:take_items(player, MINERS_PICK, 1)
        q:take_items(player, MINING_BOOTS, 1)
        q:take_items(player, REDSTONE_BEER, 1)
        q:give_items(player, NECKLACE, 1)
        st:exit_quest(false, true)
      end
    elseif sw2 == SHARI then
      if cond == 1 and not q:has_quest_items(player, BOOMBOOM_POWDER) then
        htmltext = "30517-01.htm"
        q:give_items(player, BOOMBOOM_POWDER, 1)
        if q:has_quest_items(player, MINING_BOOTS, MINERS_PICK, REDSTONE_BEER) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      else
        htmltext = "30517-02.htm"
      end
    elseif sw2 == GARITA then
      if cond == 1 and not q:has_quest_items(player, MINING_BOOTS) then
        htmltext = "30518-01.htm"
        q:give_items(player, MINING_BOOTS, 1)
        if q:has_quest_items(player, MINERS_PICK, BOOMBOOM_POWDER, REDSTONE_BEER) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      else
        htmltext = "30518-02.htm"
      end
    elseif sw2 == REED then
      if cond == 1 and not q:has_quest_items(player, REDSTONE_BEER) then
        htmltext = "30520-01.htm"
        q:give_items(player, REDSTONE_BEER, 1)
        if q:has_quest_items(player, MINING_BOOTS, MINERS_PICK, BOOMBOOM_POWDER) then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      else
        htmltext = "30520-02.htm"
      end
    elseif sw2 == BRUNON then
      if cond == 1 and not q:has_quest_items(player, MINERS_PICK) then
        htmltext = "30526-01.htm"
      else
        htmltext = "30526-03.htm"
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
