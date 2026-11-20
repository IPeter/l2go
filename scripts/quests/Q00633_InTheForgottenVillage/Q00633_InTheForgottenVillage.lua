-- Translated by questport from quests/Q00633_InTheForgottenVillage (Mobius CT_0 Interlude); owned by L2Go from here on.
local J = require("lib.java")

local q = quest.new(633, "Q00633_InTheForgottenVillage", "In the Forgotten Village")

-- NPCs
local MINA = 31388
-- Items
local RIB_BONE = 7544
local ZOMBIE_LIVER = 7545
-- Monsters / Drop chances
local MOBS = {}
local UNDEADS = {}

MOBS[21557] = 328000 -- Bone Snatcher
MOBS[21558] = 328000 -- Bone Snatcher
MOBS[21559] = 337000 -- Bone Maker
MOBS[21560] = 337000 -- Bone Shaper
MOBS[21563] = 342000 -- Bone Collector
MOBS[21564] = 348000 -- Skull Collector
MOBS[21565] = 351000 -- Bone Animator
MOBS[21566] = 359000 -- Skull Animator
MOBS[21567] = 359000 -- Bone Slayer
MOBS[21572] = 365000 -- Bone Sweeper
MOBS[21574] = 383000 -- Bone Grinder
MOBS[21575] = 383000 -- Bone Grinder
MOBS[21580] = 385000 -- Bone Caster
MOBS[21581] = 395000 -- Bone Puppeteer
MOBS[21583] = 397000 -- Bone Scavenger
MOBS[21584] = 401000 -- Bone Scavenger
UNDEADS[21553] = 347000 -- Trampled Man
UNDEADS[21554] = 347000 -- Trampled Man
UNDEADS[21561] = 450000 -- Sacrificed Man
UNDEADS[21578] = 501000 -- Behemoth Zombie
UNDEADS[21596] = 359000 -- Requiem Lord
UNDEADS[21597] = 370000 -- Requiem Behemoth
UNDEADS[21598] = 441000 -- Requiem Behemoth
UNDEADS[21599] = 395000 -- Requiem Priest
UNDEADS[21600] = 408000 -- Requiem Behemoth
UNDEADS[21601] = 411000 -- Requiem Behemoth

q:register_quest_items(RIB_BONE, ZOMBIE_LIVER)
q:add_start_npc(MINA)
q:add_talk_id(MINA)
q:add_kill_id(J.keys(MOBS))
q:add_kill_id(J.keys(UNDEADS))

q:on_event(function(event, npc, player)
  local htmltext = event
  local st = q:get_quest_state(player, false)
  if st == nil then
    return htmltext
  end
  if event == "31388-04.htm" then
    st:start_quest()
  elseif event == "31388-10.htm" then
    q:take_items(player, RIB_BONE, -1)
    st:exit_quest(true, true)
  elseif event == "31388-09.htm" then
    if q:get_quest_items_count(player, RIB_BONE) >= 200 then
      htmltext = "31388-08.htm"
      q:take_items(player, RIB_BONE, 200)
      q:give_adena(player, 25000, true)
      q:add_exp_and_sp(player, 305235, 0)
    end
    st:set_cond(1, true)
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  local htmltext = q:get_no_quest_msg(player)
  local st = q:get_quest_state(player, true)
  local sw = st:get_state()
  if sw == State.CREATED then
    htmltext = player:level() < 65 and "31388-03.htm" or "31388-01.htm"
  elseif sw == State.STARTED then
    local cond = st:get_cond()
    if cond == 1 then
      htmltext = "31388-06.htm"
    elseif cond == 2 then
      htmltext = "31388-05.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local npcId = npc:id()
  local undeadChance = UNDEADS[npcId]
  if undeadChance ~= nil then
    local st = q:get_random_party_member_state(player, -1, 3, npc)
    if st == nil or not st:is_started() then
      return
    end
    if rnd(1000000) < undeadChance then
      local partyMember = st:get_player()
      q:give_items(partyMember, ZOMBIE_LIVER, 1)
      q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
    end
  elseif MOBS[npcId] ~= nil then
    local partyMember = q:get_random_party_member(player, 1)
    if partyMember == nil then
      return
    end
    local st = q:get_quest_state(partyMember, false)
    if rnd(1000000) < MOBS[npcId] then
      q:give_items(partyMember, RIB_BONE, 1)
      if q:get_quest_items_count(partyMember, RIB_BONE) < 200 then
        q:play_sound(partyMember, QuestSound.ITEMSOUND_QUEST_ITEMGET)
      else
        st:set_cond(2, true)
      end
    end
  end
end)
