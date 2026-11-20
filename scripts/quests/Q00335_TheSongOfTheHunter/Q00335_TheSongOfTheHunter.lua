-- Translated by questport from quests/Q00335_TheSongOfTheHunter (Mobius CT_0 Interlude); owned by L2Go from here on.
-- Adapted from FirstTeam Interlude
--
-- Finished by hand. Mobius took this quest over from FirstTeam without
-- adapting it, and it cannot be played there:
--   * FirstTeam numbered the states CREATED 1, STARTED 2, COMPLETED 3;
--     Mobius has 0, 1, 2. The state checks use State.* now, with the
--     FirstTeam meaning (1 → CREATED, 2 → STARTED).
--   * onTalk only read an existing state, but FirstTeam's quest window
--     created it at the start NPC; Grey creates it now (q:get_quest_state
--     with create at GREY).
-- getActiveWeaponItem() is the item in the right hand (two-handed weapons
-- sit there too): q:get_item_equipped(player, Paperdoll.RHAND).
-- packInt's throws become error() inside the pcall that stands for
-- genList's try/catch, as in Java.
local J = require("lib.java")

local q = quest.new(335, "Q00335_TheSongOfTheHunter", "Song of the Hunter")

local GREY = 30744
local TOR = 30745
local CYBELLIN = 30746
local BREKA_ORC_WARRIOR = 20271
local WINDSUS = 20553
local TARLK_BUGBEAR_WARRIOR = 20571
local BREKA_OVERLORD_HAKA = 27140
local BREKA_OVERLORD_JAKA = 27141
local BREKA_OVERLORD_MARKA = 27142
local WINDSUS_ALEPH = 27143
local TARLK_RAIDER_ATHU = 27144
local TARLK_RAIDER_LANKA = 27145
local TARLK_RAIDER_TRISKA = 27146
local TARLK_RAIDER_MOTURA = 27147
local TARLK_RAIDER_KALATH = 27148
local CYBELLINS_DAGGER = 3471
local CIRCLE_HUNTER_LICENSE1 = 3692
local CIRCLE_HUNTER_LICENSE2 = 3693
local LAUREL_LEAF_PIN = 3694
local TEST_INSTRUCTIONS1 = 3695
local TEST_INSTRUCTIONS2 = 3696
local CYBELLINS_REQUEST = 3697
local GUARDIAN_BASILISK_SCALE = 3709
local KARUT_WEED = 3710
local HAKAS_HEAD = 3711
local JAKAS_HEAD = 3712
local MARKAS_HEAD = 3713
local WINDSUS_ALEPH_SKIN = 3714
local INDIGO_RUNESTONE = 3715
local SPORESEA_SEED = 3716
local TIMAK_ORC_TOTEM = 3717
local TRISALIM_SILK = 3718
local AMBROSIUS_FRUIT = 3719
local BALEFIRE_CRYSTAL = 3720
local IMPERIAL_ARROWHEAD = 3721
local ATHUS_HEAD = 3722
local LANKAS_HEAD = 3723
local TRISKAS_HEAD = 3724
local MOTURAS_HEAD = 3725
local KALATHS_HEAD = 3726
local Q_BLOOD_CRYSTAL = {3708, 3698, 3699, 3700, 3701, 3702, 3703, 3704, 3705, 3706, 3707}
local Q_BLOOD_CRYSTAL_LIZARDMEN = {20578, 20579, 20580, 20581, 20582, 20641, 20642, 20643, 20644, 20645}
local ITEMS_1ST_CIRCLE = {
  {{GUARDIAN_BASILISK_SCALE}, {40}, {20550, 75}},
  {{KARUT_WEED}, {20}, {20581, 50}},
  {{HAKAS_HEAD, JAKAS_HEAD, MARKAS_HEAD}, {3}},
  {{WINDSUS_ALEPH_SKIN}, {1}, {WINDSUS_ALEPH, 100}},
  {{INDIGO_RUNESTONE}, {20}, {20563, 50}, {20565, 50}},
  {{SPORESEA_SEED}, {30}, {20555, 70}},
}
local ITEMS_2ND_CIRCLE = {
  {{TIMAK_ORC_TOTEM}, {20}, {20586, 50}},
  {{TRISALIM_SILK}, {20}, {20560, 50}, {20561, 50}},
  {{AMBROSIUS_FRUIT}, {30}, {20591, 75}, {20597, 75}},
  {{BALEFIRE_CRYSTAL}, {20}, {20675, 50}},
  {{IMPERIAL_ARROWHEAD}, {20}, {20660, 50}},
  {{ATHUS_HEAD, LANKAS_HEAD, TRISKAS_HEAD, MOTURAS_HEAD, KALATHS_HEAD}, {5}},
}

local REQUESTS1, REQUESTS2, calcItemsConds, delItemsConds, getBloodCrystalLevel, bloodCrystal2Adena, genList, formatList
local getCurrentRequest, isValidRequest, packInt, unpackInt

local Request = {}
Request.__index = Request

function Request.new(requestid, requestitem, requestcount, rewardadena, txt)
  local self = setmetatable({}, Request)
  self.request_id = 0
  self.request_item = 0
  self.request_count = 0
  self.reward_adena = 0
  self.droplist = {}
  self.spawnlist = {}
  self.request_id = requestid
  self.request_item = requestitem
  self.request_count = requestcount
  self.reward_adena = rewardadena
  self.text = txt
  return self
end

function Request:addDrop(killMobId, chance)
  self.droplist[killMobId] = chance
  return self
end

function Request:addSpawn(killMobId, spawnMobId, chance)
  local ok, err = pcall(function()
    self.spawnlist[killMobId] = {spawnMobId, chance}
  end)
  if not ok then
  end
  return self
end

function Request:complete(st)
  local player = st:get_player()
  if q:get_quest_items_count(player, self.request_item) < self.request_count then
    return false
  end
  q:take_items(player, self.request_id, -1)
  q:take_items(player, self.request_item, -1)
  q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
  q:give_items(player, LAUREL_LEAF_PIN, 1)
  q:give_adena(player, self.reward_adena, true)
  st:unset("list")
  return true
end

function calcItemsConds(player, itemConds)
  local result = 0
  for _, itemCond in ipairs(itemConds) do
    local count = 0
    for _, i in ipairs(itemCond[1]) do
      count = count + q:get_quest_items_count(player, i)
    end
    if count >= itemCond[2][1] then
      result = result + 1
    end
  end
  return result
end

function delItemsConds(player, itemConds)
  for _, itemCond in ipairs(itemConds) do
    for _, i in ipairs(itemCond[1]) do
      q:take_items(player, i, -1)
    end
  end
end

function getBloodCrystalLevel(player)
  for i = #Q_BLOOD_CRYSTAL - 1, 0, -1 do
    if q:get_quest_items_count(player, Q_BLOOD_CRYSTAL[i + 1]) > 0 then
      return i
    end
  end
  return -1
end

function bloodCrystal2Adena(player, bloodCrystalLevel)
  if bloodCrystalLevel < 2 then
    return false
  end
  for _, i in ipairs(Q_BLOOD_CRYSTAL) do
    q:take_items(player, i, -1)
  end
  q:give_adena(player, 3400 * J.int(math.pow(2, bloodCrystalLevel - 2)), true)
  return true
end

function genList(st)
  -- final int grade_c = 12;
  -- final int grade_b = 6;
  -- final int grade_a = 3;
  if st:get("list") == nil or st:get("list") == "" then
    local laurelLeafPinCount = q:get_quest_items_count(st:get_player(), LAUREL_LEAF_PIN)
    local list = J.array(0, 5)
    if laurelLeafPinCount < 4 then
      if laurelLeafPinCount == 0 or rnd(100) < 80 then
        for i = 0, 4 do
          list[i + 1] = rnd(12)
        end
      else
        list[1] = 12 + rnd(6)
        list[2] = rnd(12)
        list[3] = rnd(6)
        list[4] = 6 + rnd(6)
        list[5] = rnd(12)
      end
    elseif rnd(100) < 20 then
      list[1] = 12 + rnd(6)
      list[2] = rnd(100) < 5 and 18 + rnd(3) or rnd(12)
      list[3] = rnd(6)
      list[4] = 6 + rnd(6)
      list[5] = rnd(12)
    else
      list[1] = rnd(12)
      list[2] = rnd(100) < 5 and 18 + rnd(3) or rnd(12)
      list[3] = rnd(6)
      list[4] = 6 + rnd(6)
      list[5] = rnd(12)
    end
    local sortFlag
    repeat
      sortFlag = false
      for j = 1, #list - 1 do
        if list[j + 1] < list[j] then
          local tmp = list[j + 1]
          list[j + 1] = list[j]
          list[j] = tmp
          sortFlag = true
        end
      end
    until not sortFlag
    local packedlist = 0
    local ok, err = pcall(function()
      packedlist = packInt(list, 5)
    end)
    if not ok then
    end
    st:set("list", tostring(packedlist))
  end
end

function formatList(st, requests)
  local result = "<html><head><body>Guild Member Tor:<br>%reply%<br>%reply%<br>%reply%<br>%reply%<br>%reply%<br></body></html>"
  local listpacked = unpackInt(st:get_int("list"), 5)
  for i = 0, 5 do
    local s = "<a action=\"bypass Script Q00335_TheSongOfTheHunter 30745-request-" .. requests[listpacked[i + 1] + 1].request_id .. "\">" .. requests[listpacked[i + 1] + 1].text .. "</a>"
    result = J.replace_first(result, "%reply%", s)
  end
  return result
end

function getCurrentRequest(player, requests)
  for _, r in ipairs(requests) do
    if q:get_quest_items_count(player, r.request_id) > 0 then
      return r
    end
  end
  return nil
end

function isValidRequest(id)
  for _, r in ipairs(REQUESTS1) do
    if r.request_id == id then
      return true
    end
  end
  for _, r in ipairs(REQUESTS2) do
    if r.request_id == id then
      return true
    end
  end
  return false
end

function packInt(a, bits)
  local m = J.idiv(32, bits)
  if #a > m then
    error("Overflow")
  end
  local result = 0
  local mval = J.int(math.pow(2, bits))
  for i = 0, m - 1 do
    result = J.shl(result, bits)
    local next_
    if #a > i then
      next_ = a[i + 1]
      if next_ >= mval or next_ < 0 then
        error("Overload, value is out of range")
      end
    else
      next_ = 0
    end
    result = result + next_
  end
  return result
end

function unpackInt(value, bits)
  local m = J.idiv(32, bits)
  local mval = J.int(math.pow(2, bits))
  local result = J.array(0, m)
  local a = value
  for i = m, 1, -1 do
    local next_ = a
    a = J.shr(a, bits)
    result[i] = next_ - a * mval
  end
  return result
end

REQUESTS1 = {
  Request.new(3727, 3769, 40, 4500, "C: 40 Totems of Kadesh"):addDrop(20578, 80):addDrop(20579, 83),
  Request.new(3728, 3770, 50, 7500, "C: 50 Jade Necklaces of Timak"):addDrop(20586, 89):addDrop(20588, 100),
  Request.new(3729, 3771, 50, 7300, "C: 50 Enchanted Golem Shards"):addDrop(20565, 100),
  Request.new(3730, 3772, 30, 5500, "C: 30 Pieces Monster Eye Meat"):addDrop(20556, 50),
  Request.new(3731, 3773, 40, 5000, "C: 40 Eggs of Dire Wyrm"):addDrop(20557, 80),
  Request.new(3732, 3774, 100, 6500, "C: 100 Claws of Guardian Basilisk"):addDrop(20550, 150),
  Request.new(3733, 3775, 50, 4400, "C: 50 Revenant Chains"):addDrop(20552, 100),
  Request.new(3734, 3776, 30, 5200, "C: 30 Windsus Tusks"):addDrop(WINDSUS, 50),
  Request.new(3735, 3777, 100, 7600, "C: 100 Skulls of Grandis"):addDrop(20554, 200),
  Request.new(3736, 3778, 50, 4900, "C: 50 Taik Obsidian Amulets"):addDrop(20631, 100):addDrop(20632, 93),
  Request.new(3737, 3779, 30, 7600, "C: 30 Heads of Karul Bugbear"):addDrop(20600, 50),
  Request.new(3738, 3780, 40, 7200, "C: 40 Ivory Charms of Tamlin"):addDrop(20601, 62):addDrop(20602, 80),
  Request.new(3739, 3781, 1, 4500, "B: Situation Preparation - Leto Chief"):addSpawn(20582, 27157, 10):addDrop(27157, 100),
  Request.new(3740, 3782, 50, 9500, "B: 50 Enchanted Gargoyle Horns"):addDrop(20567, 50),
  Request.new(3741, 3783, 50, 5800, "B: 50 Coiled Serpent Totems"):addDrop(20269, 93):addDrop(BREKA_ORC_WARRIOR, 100),
  Request.new(3742, 3784, 1, 4500, "B: Situation Preparation - Sorcerer Catch of Leto"):addSpawn(20581, 27156, 10):addDrop(27156, 100),
  Request.new(3743, 3785, 1, 7000, "B: Situation Preparation - Timak Raider Kaikee"):addSpawn(20586, 27158, 10):addDrop(27158, 100),
  Request.new(3744, 3786, 30, 10000, "B: 30 Kronbe Venom Sacs"):addDrop(20603, 50),
  Request.new(3745, 3787, 30, 18000, "A: 30 Eva's Charm"):addDrop(20562, 50),
  Request.new(3746, 3788, 1, 12000, "A: Titan's Tablet"):addSpawn(20554, 27160, 10):addDrop(27160, 100),
  Request.new(3747, 3789, 1, 15000, "A: Book of Shunaiman"):addSpawn(20600, 27164, 10):addDrop(27164, 100),
}
REQUESTS2 = {
  Request.new(3748, 3790, 40, 6200, "C: 40 Rotting Tree Spores"):addDrop(20558, 67),
  Request.new(3749, 3791, 40, 5900, "C: 40 Trisalim Venom Sacs"):addDrop(20560, 66):addDrop(20561, 75),
  Request.new(3750, 3792, 50, 7200, "C: 50 Totems of Taik Orc"):addDrop(20633, 53):addDrop(20634, 99),
  Request.new(3751, 3793, 40, 7200, "C: 40 Harit Barbed Necklaces"):addDrop(20641, 88):addDrop(20642, 88):addDrop(20643, 91),
  Request.new(3752, 3794, 20, 8700, "C: 20 Coins of Ancient Empire"):addDrop(20661, 50):addSpawn(20661, 27149, 5):addDrop(20662, 52):addSpawn(20662, 27149, 5):addDrop(27149, 300),
  Request.new(3753, 3795, 30, 11600, "C: 30 Skins of Farkran"):addDrop(20667, 90),
  Request.new(3754, 3796, 40, 6200, "C: 40 Tempest Shards"):addDrop(20589, 49):addSpawn(20589, 27149, 5):addDrop(27149, 500),
  Request.new(3755, 3797, 40, 7800, "C: 40 Tsunami Shards"):addDrop(20590, 51):addSpawn(20590, 27149, 5):addDrop(27149, 500),
  Request.new(3756, 3798, 40, 7800, "C: 40 Manes of Pan Ruem"):addDrop(20592, 80):addDrop(20598, 100),
  Request.new(3757, 3799, 40, 7000, "C: 40 Hamadryad Shard"):addDrop(20594, 64):addSpawn(20594, 27149, 5):addDrop(27149, 500),
  Request.new(3758, 3800, 30, 7100, "C: 30 Manes of Vanor Silenos"):addDrop(20682, 70):addDrop(20683, 85):addDrop(20684, 90),
  Request.new(3759, 3801, 30, 13400, "C: 30 Totems of Tarlk Bugbears"):addDrop(TARLK_BUGBEAR_WARRIOR, 63),
  Request.new(3760, 3802, 1, 8200, "B: Situation Preparation - Overlord Okun of Timak"):addSpawn(20588, 27159, 10):addDrop(27159, 100),
  Request.new(3761, 3803, 1, 5300, "B: Situation Preparation - Overlord Kakran of Taik"):addSpawn(20634, 27161, 10):addDrop(27161, 100),
  Request.new(3762, 3804, 40, 8800, "B: 40 Narcissus Soulstones"):addDrop(20639, 86):addSpawn(20639, 27149, 5):addDrop(27149, 500),
  Request.new(3763, 3805, 20, 11000, "B: 20 Eyes of Deprived"):addDrop(20664, 77),
  Request.new(3764, 3806, 20, 8800, "B: 20 Unicorn Horns"):addDrop(20593, 68):addDrop(20599, 86),
  Request.new(3765, 3807, 1, 5500, "B: Golden Mane of Silenos"):addSpawn(20686, 27163, 10):addDrop(27163, 100),
  Request.new(3766, 3808, 20, 16000, "A: 20 Skulls of Executed Person"):addDrop(20659, 73),
  Request.new(3767, 3809, 1, 18000, "A: Bust of Travis"):addSpawn(20662, 27162, 10):addDrop(27162, 100),
  Request.new(3768, 3810, 10, 18000, "A: 10 Swords of Cadmus"):addDrop(20676, 64),
}

q:add_start_npc(GREY)
q:add_talk_id(GREY, CYBELLIN, TOR)
q:add_kill_id(BREKA_OVERLORD_HAKA)
q:add_kill_id(BREKA_OVERLORD_JAKA)
q:add_kill_id(BREKA_OVERLORD_MARKA)
q:add_kill_id(TARLK_RAIDER_ATHU)
q:add_kill_id(TARLK_RAIDER_LANKA)
q:add_kill_id(TARLK_RAIDER_TRISKA)
q:add_kill_id(TARLK_RAIDER_MOTURA)
q:add_kill_id(TARLK_RAIDER_KALATH)
q:add_kill_id(Q_BLOOD_CRYSTAL_LIZARDMEN)
local questItems = {}
for _, ItemsCond in ipairs(ITEMS_1ST_CIRCLE) do
  for _, i in ipairs(ItemsCond[1]) do
    table.insert(questItems, i)
  end
  for i = 2, #ItemsCond - 1 do
    q:add_kill_id(ItemsCond[i + 1][1])
  end
end
for _, ItemsCond in ipairs(ITEMS_2ND_CIRCLE) do
  for _, i in ipairs(ItemsCond[1]) do
    table.insert(questItems, i)
  end
  for i = 2, #ItemsCond - 1 do
    q:add_kill_id(ItemsCond[i + 1][1])
  end
end
for _, r in ipairs(REQUESTS1) do
  table.insert(questItems, r.request_id)
  table.insert(questItems, r.request_item)
  for id in J.spairs(r.droplist) do
    q:add_kill_id(id)
  end
  for id in J.spairs(r.spawnlist) do
    q:add_kill_id(id)
  end
end
for _, r in ipairs(REQUESTS2) do
  table.insert(questItems, r.request_id)
  table.insert(questItems, r.request_item)
  for id in J.spairs(r.droplist) do
    q:add_kill_id(id)
  end
  for id in J.spairs(r.spawnlist) do
    q:add_kill_id(id)
  end
end
table.insert(questItems, CIRCLE_HUNTER_LICENSE1)
table.insert(questItems, CIRCLE_HUNTER_LICENSE2)
table.insert(questItems, LAUREL_LEAF_PIN)
table.insert(questItems, TEST_INSTRUCTIONS1)
table.insert(questItems, TEST_INSTRUCTIONS2)
table.insert(questItems, CYBELLINS_REQUEST)
table.insert(questItems, CYBELLINS_DAGGER)
for _, i in ipairs(Q_BLOOD_CRYSTAL) do
  table.insert(questItems, i)
end
q:register_quest_items(questItems)

q:on_event(function(event, npc, player)
  local st = player:get_quest_state(q:name())
  if st == nil then
    return event
  end
  local htmltext = event
  local state = st:get_state()
  -- L2Go: FirstTeam's state numbers (1 created, 2 started) as State.*.
  if "30744_03.htm" == htmltext then
    if state == State.CREATED then
      if q:get_quest_items_count(player, TEST_INSTRUCTIONS1) == 0 then
        q:give_items(player, TEST_INSTRUCTIONS1, 1)
      end
      st:start_quest()
    end
  elseif "30744_09.htm" == htmltext then
    if state == State.STARTED then
      if getCurrentRequest(player, REQUESTS1) ~= nil then
        return "30744_09a.htm"
      end
      if q:get_quest_items_count(player, TEST_INSTRUCTIONS2) == 0 then
        q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
        q:give_items(player, TEST_INSTRUCTIONS2, 1)
      end
    end
  elseif "30744_16.htm" == htmltext then
    if state == State.STARTED then
      if q:get_quest_items_count(player, LAUREL_LEAF_PIN) >= 20 then
        q:give_adena(player, 20000, true)
        htmltext = "30744_17.htm"
      end
      st:exit_quest(true, true)
    end
  elseif "30746_03.htm" == htmltext then
    if state == State.STARTED then
      if q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE1) == 0 and q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE2) == 0 then
        return nil
      end
      if q:get_quest_items_count(player, CYBELLINS_DAGGER) == 0 then
        q:give_items(player, CYBELLINS_DAGGER, 1)
      end
      if q:get_quest_items_count(player, CYBELLINS_REQUEST) == 0 then
        q:give_items(player, CYBELLINS_REQUEST, 1)
      end
      for _, i in ipairs(Q_BLOOD_CRYSTAL) do
        q:take_items(player, i, -1)
      end
      q:play_sound(player, QuestSound.ITEMSOUND_QUEST_MIDDLE)
      q:give_items(player, Q_BLOOD_CRYSTAL[2], 1)
    end
  elseif "30746_06.htm" == htmltext then
    if state == State.STARTED and not bloodCrystal2Adena(player, getBloodCrystalLevel(player)) then
      return nil
    end
  elseif "30746_10.htm" == htmltext then
    if state == State.STARTED then
      q:take_items(player, CYBELLINS_DAGGER, -1)
      q:take_items(player, CYBELLINS_REQUEST, -1)
      for _, i in ipairs(Q_BLOOD_CRYSTAL) do
        q:take_items(player, i, -1)
      end
    end
  elseif "30745_02.htm" == htmltext then
    if state == State.STARTED and q:get_quest_items_count(player, TEST_INSTRUCTIONS2) > 0 then
      return "30745_03.htm"
    end
  elseif "30745_05b.htm" == htmltext then
    if state == State.STARTED then
      if q:get_quest_items_count(player, LAUREL_LEAF_PIN) > 0 then
        q:take_items(player, LAUREL_LEAF_PIN, 1)
      end
      for _, r in ipairs(REQUESTS1) do
        q:take_items(player, r.request_id, -1)
        q:take_items(player, r.request_item, -1)
      end
      for _, r in ipairs(REQUESTS2) do
        q:take_items(player, r.request_id, -1)
        q:take_items(player, r.request_item, -1)
      end
    end
  elseif state == State.STARTED then
    if "30745-list1" == htmltext then
      genList(st)
      return formatList(st, REQUESTS1)
    end
    if "30745-list2" == htmltext then
      genList(st)
      return formatList(st, REQUESTS2)
    end
    if J.starts_with(htmltext, "30745-request-") then
      htmltext = J.replace_first(htmltext, "30745-request-", "")
      local requestId
      local ok, err = pcall(function()
        requestId = J.parse_int(htmltext)
      end)
      if not ok then
        return nil
      end
      if not isValidRequest(requestId) then
        return nil
      end
      q:give_items(player, requestId, 1)
      return "30745-" .. requestId .. ".htm"
    end
  end
  return htmltext
end)

q:on_talk(function(npc, player)
  -- L2Go: FirstTeam's quest window made the state at the start NPC.
  local st = q:get_quest_state(player, npc:id() == GREY)
  local htmltext = q:get_no_quest_msg(player)
  if st == nil then
    return htmltext
  end
  local state = st:get_state()
  local npcId = npc:id()
  if state == State.CREATED then
    if npcId ~= GREY then
      return htmltext
    end
    if st:get_player():level() < 35 then
      st:exit_quest(true)
      return "30744_01.htm"
    end
    st:set_cond(0)
    st:unset("list")
    return "30744_02.htm"
  end
  if state ~= State.STARTED then
    return htmltext
  end
  if npcId == GREY then
    if q:get_quest_items_count(player, TEST_INSTRUCTIONS1) > 0 then
      if calcItemsConds(player, ITEMS_1ST_CIRCLE) < 3 then
        return "30744_05.htm"
      end
      delItemsConds(player, ITEMS_1ST_CIRCLE)
      q:take_items(player, TEST_INSTRUCTIONS1, -1)
      q:give_items(player, CIRCLE_HUNTER_LICENSE1, 1)
      st:set_cond(2, true)
      return "30744_06.htm"
    end
    if q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE1) > 0 then
      if st:get_player():level() < 45 then
        return "30744_07.htm"
      end
      if q:get_quest_items_count(player, TEST_INSTRUCTIONS2) == 0 then
        return "30744_08.htm"
      end
    end
    if q:get_quest_items_count(player, TEST_INSTRUCTIONS2) > 0 then
      if calcItemsConds(player, ITEMS_2ND_CIRCLE) < 3 then
        return "30744_11.htm"
      end
      delItemsConds(player, ITEMS_2ND_CIRCLE)
      q:take_items(player, TEST_INSTRUCTIONS2, -1)
      q:take_items(player, CIRCLE_HUNTER_LICENSE1, -1)
      q:give_items(player, CIRCLE_HUNTER_LICENSE2, 1)
      st:set_cond(3, true)
      return "30744_12.htm"
    elseif q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE2) > 0 then
      return "30744_14.htm"
    end
  end
  if npcId == CYBELLIN then
    if q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE1) == 0 and q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE2) == 0 then
      return "30746_01.htm"
    end
    if q:get_quest_items_count(player, CYBELLINS_REQUEST) == 0 then
      return "30746_02.htm"
    end
    local bloodCrystalLevel = getBloodCrystalLevel(player)
    if bloodCrystalLevel == -1 then
      return "30746_08.htm"
    end
    if bloodCrystalLevel == 0 then
      return "30746_09.htm"
    end
    if bloodCrystalLevel == 1 then
      return "30746_04.htm"
    end
    if bloodCrystalLevel > 1 and bloodCrystalLevel < 10 then
      return "30746_05.htm"
    end
    if bloodCrystalLevel == 10 and bloodCrystal2Adena(player, bloodCrystalLevel) then
      return "30746_05a.htm"
    end
  end
  if npcId == TOR then
    if q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE1) == 0 and q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE2) == 0 then
      return "30745_01a.htm"
    end
    if q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE1) > 0 then
      local request = getCurrentRequest(player, REQUESTS1)
      if request ~= nil then
        return request:complete(st) and "30745_06a.htm" or "30745_05.htm"
      end
      if st:get_player():level() < 45 then
        return "30745_01b.htm"
      end
      return q:get_quest_items_count(player, TEST_INSTRUCTIONS2) > 0 and "30745_03.htm" or "30745_03a.htm"
    elseif q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE2) > 0 then
      local request = getCurrentRequest(player, REQUESTS2)
      if request == nil then
        return "30745_03b.htm"
      end
      return request:complete(st) and "30745_06b.htm" or "30745_05.htm"
    end
  end
  return htmltext
end)

q:on_kill(function(npc, player, isPet)
  local st = q:get_quest_state(player, false)
  if st == nil or not st:is_started() then
    return
  end
  if st:get_state() ~= State.STARTED then
    return
  end
  local npcId = npc:id()
  local itemsCircle = nil
  if q:get_quest_items_count(player, TEST_INSTRUCTIONS1) > 0 then
    itemsCircle = ITEMS_1ST_CIRCLE
  elseif q:get_quest_items_count(player, TEST_INSTRUCTIONS2) > 0 then
    itemsCircle = ITEMS_2ND_CIRCLE
  end
  if itemsCircle ~= nil then
    for _, itemCond in ipairs(itemsCircle) do
      for i = 2, #itemCond - 1 do
        if npcId == itemCond[i + 1][1] and rnd(100) < itemCond[i + 1][2] and q:get_quest_items_count(player, itemCond[1][1]) < itemCond[2][1] then
          q:give_items(player, itemCond[1][1], 1)
        end
      end
    end
    if q:get_quest_items_count(player, TEST_INSTRUCTIONS1) > 0 then
      local hakasHeadCount = q:get_quest_items_count(player, HAKAS_HEAD)
      local jakasHeadCount = q:get_quest_items_count(player, JAKAS_HEAD)
      local markasHeadCount = q:get_quest_items_count(player, MARKAS_HEAD)
      if npcId == BREKA_ORC_WARRIOR then
        if hakasHeadCount == 0 and rnd(100) < 10 then
          q:add_spawn(BREKA_OVERLORD_HAKA, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
        elseif jakasHeadCount == 0 and rnd(100) < 10 then
          q:add_spawn(BREKA_OVERLORD_JAKA, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
        elseif markasHeadCount == 0 and rnd(100) < 10 then
          q:add_spawn(BREKA_OVERLORD_MARKA, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
        end
      elseif npcId == BREKA_OVERLORD_HAKA then
        if hakasHeadCount == 0 then
          q:give_items(player, HAKAS_HEAD, 1)
        end
      elseif npcId == BREKA_OVERLORD_JAKA then
        if jakasHeadCount == 0 then
          q:give_items(player, JAKAS_HEAD, 1)
        end
      elseif npcId == BREKA_OVERLORD_MARKA then
        if markasHeadCount == 0 then
          q:give_items(player, MARKAS_HEAD, 1)
        end
      elseif npcId == WINDSUS and q:get_quest_items_count(player, WINDSUS_ALEPH_SKIN) == 0 and rnd(100) < 10 then
        q:add_spawn(WINDSUS_ALEPH, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
      end
    elseif q:get_quest_items_count(player, TEST_INSTRUCTIONS2) > 0 then
      local athusHeadCount = q:get_quest_items_count(player, ATHUS_HEAD)
      local lankasHeadCount = q:get_quest_items_count(player, LANKAS_HEAD)
      local triskasHeadCount = q:get_quest_items_count(player, TRISKAS_HEAD)
      local moturasHeadCount = q:get_quest_items_count(player, MOTURAS_HEAD)
      local kalathsHeadCount = q:get_quest_items_count(player, KALATHS_HEAD)
      if npcId == TARLK_BUGBEAR_WARRIOR then
        if athusHeadCount == 0 and rnd(100) < 10 then
          q:add_spawn(TARLK_RAIDER_ATHU, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
        elseif lankasHeadCount == 0 and rnd(100) < 10 then
          q:add_spawn(TARLK_RAIDER_LANKA, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
        elseif triskasHeadCount == 0 and rnd(100) < 10 then
          q:add_spawn(TARLK_RAIDER_TRISKA, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
        elseif moturasHeadCount == 0 and rnd(100) < 10 then
          q:add_spawn(TARLK_RAIDER_MOTURA, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
        elseif kalathsHeadCount == 0 and rnd(100) < 10 then
          q:add_spawn(TARLK_RAIDER_KALATH, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
        end
      elseif npcId == TARLK_RAIDER_ATHU then
        if athusHeadCount == 0 then
          q:give_items(player, ATHUS_HEAD, 1)
        end
      elseif npcId == TARLK_RAIDER_LANKA then
        if lankasHeadCount == 0 then
          q:give_items(player, LANKAS_HEAD, 1)
        end
      elseif npcId == TARLK_RAIDER_TRISKA then
        if triskasHeadCount == 0 then
          q:give_items(player, TRISKAS_HEAD, 1)
        end
      elseif npcId == TARLK_RAIDER_MOTURA then
        if moturasHeadCount == 0 then
          q:give_items(player, MOTURAS_HEAD, 1)
        end
      elseif npcId == TARLK_RAIDER_KALATH and kalathsHeadCount == 0 then
        q:give_items(player, KALATHS_HEAD, 1)
      end
    end
  end
  if q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE1) > 0 or q:get_quest_items_count(player, CIRCLE_HUNTER_LICENSE2) > 0 then
    -- L2Go: getActiveWeaponItem() is the right-hand item.
    if q:get_quest_items_count(player, CYBELLINS_REQUEST) > 0 and q:get_item_equipped(st:get_player(), Paperdoll.RHAND) == CYBELLINS_DAGGER then
      local bloodCrystalLevel = getBloodCrystalLevel(player)
      if bloodCrystalLevel > 0 and bloodCrystalLevel < 10 then
        for _, lizardmen_id in ipairs(Q_BLOOD_CRYSTAL_LIZARDMEN) do
          if npcId == lizardmen_id then
            if rnd(100) < 50 then
              q:take_items(player, Q_BLOOD_CRYSTAL[bloodCrystalLevel + 1], -1)
              q:play_sound(player, bloodCrystalLevel < 6 and QuestSound.ITEMSOUND_QUEST_MIDDLE or QuestSound.ITEMSOUND_QUEST_JACKPOT)
              q:give_items(player, Q_BLOOD_CRYSTAL[bloodCrystalLevel + 2], 1)
            else
              for _, i in ipairs(Q_BLOOD_CRYSTAL) do
                q:take_items(player, i, -1)
              end
              q:give_items(player, Q_BLOOD_CRYSTAL[1], 1)
            end
          end
        end
      end
    end
    local request = getCurrentRequest(player, REQUESTS1)
    if request == nil then
      request = getCurrentRequest(player, REQUESTS2)
    end
    if request ~= nil then
      if request.droplist[npcId] ~= nil and rnd(100) < request.droplist[npcId] and q:get_quest_items_count(player, request.request_item) < request.request_count then
        q:give_items(player, request.request_item, 1)
      end
      if request.spawnlist[npcId] ~= nil and q:get_quest_items_count(player, request.request_item) < request.request_count then
        local spawnChance = request.spawnlist[npcId]
        if rnd(100) < spawnChance[2] then
          q:add_spawn(spawnChance[1], {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
          if spawnChance[1] == 27149 then
            npc:broadcast_say(ChatType.GENERAL, "Show me the pretty sparkling things! They're all mine!")
          end
        end
      end
    end
  end
  if (npcId == 27160 or npcId == 27162 or npcId == 27164) and rnd(100) < 50 then
    npc:broadcast_say(ChatType.GENERAL, "We'll take the property of the ancient empire!")
    q:add_spawn(27150, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
    q:add_spawn(27150, {x = npc:x(), y = npc:y(), z = npc:z(), heading = npc:heading()}, true, 300000)
  end
end)
