-- Translated by questport from quests/Q00385_YokeOfThePast (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(385, "Q00385_YokeOfThePast", "Yoke of the Past")

-- NPCs
local GATEKEEPER_ZIGGURAT = {
  31095,
  31096,
  31097,
  31098,
  31099,
  31100,
  31101,
  31102,
  31103,
  31104,
  31105,
  31106,
  31107,
  31108,
  31109,
  31110,
  31114,
  31115,
  31116,
  31117,
  31118,
  31119,
  31120,
  31121,
  31122,
  31123,
  31124,
  31125,
  31126,
}
-- Item
local ANCIENT_SCROLL = 5902
-- Reward
local BLANK_SCROLL = 5965
-- Drop chances
local CHANCES = {}

CHANCES[21208] = 70000
CHANCES[21209] = 80000
CHANCES[21210] = 110000
CHANCES[21211] = 110000
CHANCES[21213] = 140000
CHANCES[21214] = 190000
CHANCES[21215] = 190000
CHANCES[21217] = 240000
CHANCES[21218] = 300000
CHANCES[21219] = 300000
CHANCES[21221] = 370000
CHANCES[21222] = 460000
CHANCES[21223] = 450000
CHANCES[21224] = 500000
CHANCES[21225] = 540000
CHANCES[21226] = 660000
CHANCES[21227] = 640000
CHANCES[21228] = 700000
CHANCES[21229] = 750000
CHANCES[21230] = 910000
CHANCES[21231] = 860000
CHANCES[21236] = 120000
CHANCES[21237] = 140000
CHANCES[21238] = 190000
CHANCES[21239] = 190000
CHANCES[21240] = 220000
CHANCES[21241] = 240000
CHANCES[21242] = 300000
CHANCES[21243] = 300000
CHANCES[21244] = 340000
CHANCES[21245] = 370000
CHANCES[21246] = 460000
CHANCES[21247] = 450000
CHANCES[21248] = 500000
CHANCES[21249] = 540000
CHANCES[21250] = 660000
CHANCES[21251] = 640000
CHANCES[21252] = 700000
CHANCES[21253] = 750000
CHANCES[21254] = 910000
CHANCES[21255] = 860000

q:register_quest_items(ANCIENT_SCROLL)
q:add_start_npc(GATEKEEPER_ZIGGURAT)
q:add_talk_id(GATEKEEPER_ZIGGURAT)
q:add_kill_id(J.keys(CHANCES))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "05.htm" then
    st:start_quest()
  elseif event == "10.htm" then
    st:exit_quest(true, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 20 and "02.htm" or "01.htm"
  elseif sw == State.STARTED then
    if not q:has_quest_items(player, ANCIENT_SCROLL) then
      htmltext = "08.htm"
    else
      htmltext = "09.htm"
      local count = q:get_quest_items_count(player, ANCIENT_SCROLL)
      q:take_items(player, ANCIENT_SCROLL, -1)
      q:reward_items(player, BLANK_SCROLL, count)
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_random_party_member_state(player, -1, 3, npc)
  if st == nil then
    return
  end
  if rnd(1000000) < CHANCES[npc:id()] then
    local partyMember = st:get_player()
    q:give_items(partyMember, ANCIENT_SCROLL, 1)
    q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end)
