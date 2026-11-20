-- Path to a Warrior (Q401), ported by hand from Mobius CT_0 Interlude
-- quests/Q00401_PathOfTheWarrior. Auron (Gludin) sends a level 19 Human
-- Fighter to Simplon (Gludio): ten Rusted Bronze Swords from Tracker
-- Skeletons, then twenty Poison Spider Legs taken with the Rusted Bronze
-- Sword in hand. The reward is the Medallion of Warrior.
--
-- The pages are Mobius's, served from
-- data/html/scripts/quests/Q00401_PathOfTheWarrior/.

local q = quest.new(401, "Q00401_PathOfTheWarrior", "Path to a Warrior")

-- Items
local AURON_LETTER = 1138
local WARRIOR_GUILD_MARK = 1139
local RUSTED_BRONZE_SWORD_1 = 1140
local RUSTED_BRONZE_SWORD_2 = 1141
local RUSTED_BRONZE_SWORD_3 = 1142
local SIMPLON_LETTER = 1143
local POISON_SPIDER_LEG = 1144
local MEDALLION_OF_WARRIOR = 1145

-- NPCs
local AURON = 30010
local SIMPLON = 30253

-- Monsters
local TRACKER_SKELETON = 20035
local TRACKER_SKELETON_LEADER = 20042
local VENOMOUS_SPIDER = 20038
local ARACHNID_TRACKER = 20043

q:register_quest_items(AURON_LETTER, WARRIOR_GUILD_MARK, RUSTED_BRONZE_SWORD_1, RUSTED_BRONZE_SWORD_2,
  RUSTED_BRONZE_SWORD_3, SIMPLON_LETTER, POISON_SPIDER_LEG)
q:add_start_npc(AURON)
q:add_talk_id(AURON, SIMPLON)
q:add_kill_id(TRACKER_SKELETON, VENOMOUS_SPIDER, TRACKER_SKELETON_LEADER, ARACHNID_TRACKER)

q:on_event(function(event, npc, player)
  local st = q:get_quest_state(player, false)
  if st == nil then
    return event
  end
  if event == "30010-05.htm" then
    if player:class_id() ~= ClassId.FIGHTER then
      return player:class_id() == ClassId.WARRIOR and "30010-03.htm" or "30010-02b.htm"
    elseif player:level() < 19 then
      return "30010-02.htm"
    elseif q:has_quest_items(player, MEDALLION_OF_WARRIOR) then
      return "30010-04.htm"
    end
  elseif event == "30010-06.htm" then
    st:start_quest()
    q:give_items(player, AURON_LETTER, 1)
  elseif event == "30253-02.htm" then
    st:set_cond(2, true)
    q:take_items(player, AURON_LETTER, 1)
    q:give_items(player, WARRIOR_GUILD_MARK, 1)
  elseif event == "30010-11.htm" then
    st:set_cond(5, true)
    q:take_items(player, RUSTED_BRONZE_SWORD_2, 1)
    q:take_items(player, SIMPLON_LETTER, 1)
    q:give_items(player, RUSTED_BRONZE_SWORD_3, 1)
  end
  return event
end)

local function talk_auron(st, player, cond)
  if cond == 1 then
    return "30010-07.htm"
  elseif cond == 2 or cond == 3 then
    return "30010-08.htm"
  elseif cond == 4 then
    return "30010-09.htm"
  elseif cond == 5 then
    return "30010-12.htm"
  elseif cond == 6 then
    q:take_items(player, POISON_SPIDER_LEG, -1)
    q:take_items(player, RUSTED_BRONZE_SWORD_3, 1)
    q:give_items(player, MEDALLION_OF_WARRIOR, 1)
    q:add_exp_and_sp(player, 3200, 1500)
    player:social_action(3)
    st:exit_quest(true, true)
    return "30010-13.htm"
  end
end

local function talk_simplon(st, player, cond)
  if cond == 1 then
    return "30253-01.htm"
  elseif cond == 2 then
    if not q:has_quest_items(player, RUSTED_BRONZE_SWORD_1) then
      return "30253-03.htm"
    elseif q:get_quest_items_count(player, RUSTED_BRONZE_SWORD_1) <= 9 then
      return "30253-03b.htm"
    end
  elseif cond == 3 then
    st:set_cond(4, true)
    q:take_items(player, RUSTED_BRONZE_SWORD_1, 10)
    q:take_items(player, WARRIOR_GUILD_MARK, 1)
    q:give_items(player, RUSTED_BRONZE_SWORD_2, 1)
    q:give_items(player, SIMPLON_LETTER, 1)
    return "30253-04.htm"
  elseif cond == 4 then
    return "30253-05.htm"
  end
end

q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  local state = st:get_state()
  if state == State.CREATED then
    return "30010-01.htm"
  elseif state == State.STARTED then
    local page
    if npc:id() == AURON then
      page = talk_auron(st, player, st:get_cond())
    elseif npc:id() == SIMPLON then
      page = talk_simplon(st, player, st:get_cond())
    end
    if page ~= nil then
      return page
    end
  end
  return q:get_no_quest_msg(player)
end)

-- collect gives one quest item and plays the item sound, or moves the
-- quest to cond once the count reaches need.
local function collect(st, player, item, need, cond)
  if q:get_quest_items_count(player, item) >= need then
    return
  end
  q:give_items(player, item, 1)
  if q:get_quest_items_count(player, item) >= need then
    st:set_cond(cond, true)
  else
    q:play_sound(player, QuestSound.ITEMSOUND_QUEST_ITEMGET)
  end
end

q:on_kill(function(npc, player, is_summon)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  local id = npc:id()
  if id == TRACKER_SKELETON or id == TRACKER_SKELETON_LEADER then
    if st:is_cond(2) and rnd(10) < 4 then
      collect(st, player, RUSTED_BRONZE_SWORD_1, 10, 3)
    end
  elseif id == VENOMOUS_SPIDER or id == ARACHNID_TRACKER then
    if st:is_cond(5) and q:get_item_equipped(player, Paperdoll.RHAND) == RUSTED_BRONZE_SWORD_3 then
      collect(st, player, POISON_SPIDER_LEG, 20, 6)
    end
  end
end)
