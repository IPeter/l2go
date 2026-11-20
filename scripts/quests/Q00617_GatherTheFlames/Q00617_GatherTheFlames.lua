-- Translated by questport from quests/Q00617_GatherTheFlames (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(617, "Q00617_GatherTheFlames", "Gather the Flames")

-- NPCs
local HILDA = 31271
local VULCAN = 31539
local ROONEY = 32049
-- Items
local TORCH = 7264
-- Drop chances
local CHANCES = {}
-- Rewards
local REWARD = {6881, 6883, 6885, 6887, 6891, 6893, 6895, 6897, 6899, 7580}

CHANCES[21381] = 510000
CHANCES[21653] = 510000
CHANCES[21387] = 530000
CHANCES[21655] = 530000
CHANCES[21390] = 560000
CHANCES[21656] = 690000
CHANCES[21389] = 550000
CHANCES[21388] = 530000
CHANCES[21383] = 510000
CHANCES[21392] = 560000
CHANCES[21382] = 600000
CHANCES[21654] = 520000
CHANCES[21384] = 640000
CHANCES[21394] = 510000
CHANCES[21395] = 560000
CHANCES[21385] = 520000
CHANCES[21391] = 550000
CHANCES[21393] = 580000
CHANCES[21657] = 570000
CHANCES[21386] = 520000
CHANCES[21652] = 490000
CHANCES[21378] = 490000
CHANCES[21376] = 480000
CHANCES[21377] = 480000
CHANCES[21379] = 590000
CHANCES[21380] = 490000

q:register_quest_items(TORCH)
q:add_start_npc(VULCAN, HILDA)
q:add_talk_id(VULCAN, HILDA, ROONEY)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31539-03.htm" or event == "31271-03.htm" then
    st:start_quest()
  elseif event == "31539-05.htm" then
    if q:get_quest_items_count(player, TORCH) >= 1000 then
      htmltext = "31539-07.htm"
      q:take_items(player, TORCH, 1000)
      q:give_items(player, rnd.entry(REWARD), 1)
    end
  elseif event == "31539-08.htm" then
    q:take_items(player, TORCH, -1)
    st:exit_quest(true, true)
  elseif J.is_numeric(event) then
    if q:get_quest_items_count(player, TORCH) >= 1200 then
      htmltext = "32049-03.htm"
      q:take_items(player, TORCH, 1200)
      q:give_items(player, J.parse_int(event), 1)
    else
      htmltext = "32049-02.htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = npc:id() .. (player:level() >= 74 and "-01.htm" or "-02.htm")
  elseif sw == State.STARTED then
    local sw2 = npc:id()
    if sw2 == VULCAN then
      htmltext = q:get_quest_items_count(player, TORCH) >= 1000 and "31539-04.htm" or "31539-05.htm"
    elseif sw2 == HILDA then
      htmltext = "31271-04.htm"
    elseif sw2 == ROONEY then
      htmltext = q:get_quest_items_count(player, TORCH) >= 1200 and "32049-01.htm" or "32049-02.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, killer, isSummon)
  local partyMember = q:get_random_party_member_state(killer, State.STARTED)
  if partyMember == nil then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    q:give_item_randomly(partyMember, npc, TORCH, 1, 1, 0, 1, true)
  end
end)
