-- Translated by questport from quests/Q00004_LongLiveThePaagrioLord (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(4, "Q00004_LongLiveThePaagrioLord", "Long live the Pa'agrio Lord!")

local NAKUSIN = 30578
local NPC_GIFTS = {}

NPC_GIFTS[30585] = 1542
NPC_GIFTS[30566] = 1541
NPC_GIFTS[30562] = 1543
NPC_GIFTS[30560] = 1544
NPC_GIFTS[30559] = 1545
NPC_GIFTS[30587] = 1546

q:register_quest_items(1541, 1542, 1543, 1544, 1545, 1546)
q:add_start_npc(30578) -- Nakusin
q:add_talk_id(30578, 30585, 30566, 30562, 30560, 30559, 30587)

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "30578-03.htm" then
    st:start_quest()
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    if player:race() ~= Race.ORC then
      htmltext = "30578-00.htm"
    elseif player:level() < 2 then
      htmltext = "30578-01.htm"
    else
      htmltext = "30578-02.htm"
    end
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    local npcId = npc:id()
    if npcId == NAKUSIN then
      if cond == 1 then
        htmltext = "30578-04.htm"
      elseif cond == 2 then
        htmltext = "30578-06.htm"
        q:give_items(player, 4, 1)
        for _, item in J.spairs(NPC_GIFTS) do
          q:take_items(player, item, -1)
        end
        st:exit_quest(false, true)
      end
    else
      local i = NPC_GIFTS[npcId]
      if q:has_quest_items(player, i) then
        htmltext = "30585-02.htm"
      else
        q:give_items(player, i, 1)
        htmltext = "30585-01.htm"
        local count = 0
        for _, item in J.spairs(NPC_GIFTS) do
          count = count + q:get_quest_items_count(player, item)
        end
        if count == 6 then
          st:set_cond(2, true)
        else
          q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
        end
      end
    end
  elseif sw == State.COMPLETED then
    htmltext = q:get_already_completed_msg(player)
  end
  return htmltext
end)
