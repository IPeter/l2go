-- The ferry between Giran and Talking Island (Mobius
-- vehicles/BoatGiranTalking.java). Tickets 3946 and 3945. Mobius never
-- docks this boat at Giran, and never takes or frees the Talking Island
-- dock when it leaves Giran.

local boat = require("lib.boat")
local say = boat.say

-- Time: 868 s.
local GIRAN_TO_TALKING = {
  { 51914, 189023, -3610, 150, 800 },
  { 60567, 189789, -3610, 150, 800 },
  { 63732, 197457, -3610, 200, 800 },
  { 63732, 219946, -3610, 250, 800 },
  { 62008, 222240, -3610, 250, 1200 },
  { 56115, 226791, -3610, 250, 1200 },
  { 40384, 226432, -3610, 300, 800 },
  { 37760, 226432, -3610, 300, 800 },
  { 27153, 226791, -3610, 300, 800 },
  { 12672, 227535, -3610, 300, 800 },
  { -1808, 228280, -3610, 300, 800 },
  { -22165, 230542, -3610, 300, 800 },
  { -42523, 235205, -3610, 300, 800 },
  { -68451, 259560, -3610, 250, 800 },
  { -70848, 261696, -3610, 200, 800 },
  { -83344, 261610, -3610, 200, 800 },
  { -88344, 261660, -3610, 180, 800 },
  { -92344, 261660, -3610, 180, 800 },
  { -94242, 261659, -3610, 150, 800 },
}

local TALKING_DOCK = { -96622, 261660, -3610, 150, 800 }

-- Time: 1398 s.
local TALKING_TO_GIRAN = {
  { -113925, 261660, -3610, 150, 800 },
  { -126107, 249116, -3610, 180, 800 },
  { -126107, 234499, -3610, 180, 800 },
  { -126107, 219882, -3610, 180, 800 },
  { -109414, 204914, -3610, 180, 800 },
  { -92807, 204914, -3610, 180, 800 },
  { -80425, 216450, -3610, 250, 800 },
  { -68043, 227987, -3610, 250, 800 },
  { -63744, 231168, -3610, 250, 800 },
  { -60844, 231369, -3610, 250, 1800 },
  { -44915, 231369, -3610, 200, 800 },
  { -28986, 231369, -3610, 200, 800 },
  { 8233, 207624, -3610, 200, 800 },
  { 21470, 201503, -3610, 180, 800 },
  { 40058, 195383, -3610, 180, 800 },
  { 43022, 193793, -3610, 150, 800 },
  { 45986, 192203, -3610, 150, 800 },
  { 48950, 190613, -3610, 150, 800 },
}

-- Mobius: the last point of TALKING_TO_GIRAN.
local GIRAN_DOCK = TALKING_TO_GIRAN[#TALKING_TO_GIRAN]

local GI, T = GIRAN_DOCK, TALKING_DOCK

boat.route {
  name = "GiranTalking",
  boat = 2,
  start = { 48950, 190613, -3610, 60800 },
  steps = {
    -- 0
    { say = say(GI, T, "THE_FERRY_WILL_LEAVE_FOR_TALKING_ISLAND_HARBOR_IN_FIVE_MINUTES"), wait = 240000 },
    -- 1
    { say = say(GI, T, "THE_FERRY_WILL_LEAVE_FOR_TALKING_ISLAND_HARBOR_IN_ONE_MINUTE"), wait = 40000 },
    -- 2
    { say = say(GI, T, "THE_FERRY_WILL_BE_LEAVING_SOON_FOR_TALKING_ISLAND_HARBOR"), wait = 20000 },
    -- 3
    {
      say = say(GI, T, "THE_FERRY_IS_LEAVING_FOR_TALKING_ISLAND_HARBOR",
        "THE_FERRY_FROM_GIRAN_HARBOR_WILL_BE_ARRIVING_AT_TALKING_ISLAND_IN_APPROXIMATELY_15_MINUTES"),
      horn = GI,
      pay = { item = boat.TICKET_GIRAN_TO_TALKING, count = 1, oust = { 46763, 187041, -3451 } },
      path = GIRAN_TO_TALKING,
      wait = 250000,
    },
    -- 4
    { say = say(T, GI, "THE_FERRY_FROM_GIRAN_HARBOR_WILL_BE_ARRIVING_AT_TALKING_ISLAND_IN_APPROXIMATELY_10_MINUTES"), wait = 300000 },
    -- 5
    { say = say(T, GI, "THE_FERRY_FROM_GIRAN_HARBOR_WILL_BE_ARRIVING_AT_TALKING_ISLAND_IN_APPROXIMATELY_5_MINUTES"), wait = 240000 },
    -- 6: the end of the path runs the next step.
    { say = say(T, GI, "THE_FERRY_FROM_GIRAN_HARBOR_WILL_BE_ARRIVING_AT_TALKING_ISLAND_IN_APPROXIMATELY_1_MINUTE") },
    -- 7
    {
      await = boat.await(boat.TALKING_ISLAND, T, GI, "THE_FERRY_FROM_GIRAN_HARBOR_TO_TALKING_ISLAND_HAS_BEEN_DELAYED"),
      path = { TALKING_DOCK },
    },
    -- 8
    {
      dock = boat.TALKING_ISLAND,
      say = say(T, GI, "THE_FERRY_ARRIVED_AT_TALKING_ISLAND_HARBOR", "THE_FERRY_WILL_LEAVE_FOR_GIRAN_HARBOR_AFTER_ANCHORING_FOR_TEN_MINUTES"),
      horn = T,
      wait = 300000,
    },
    -- 9
    { say = say(T, GI, "THE_FERRY_WILL_LEAVE_FOR_GIRAN_HARBOR_IN_FIVE_MINUTES"), wait = 240000 },
    -- 10
    { say = say(T, GI, "THE_FERRY_WILL_LEAVE_FOR_GIRAN_HARBOR_IN_ONE_MINUTE"), wait = 40000 },
    -- 11
    { say = say(T, GI, "THE_FERRY_WILL_BE_LEAVING_SOON_FOR_GIRAN_HARBOR"), wait = 20000 },
    -- 12
    {
      undock = boat.TALKING_ISLAND,
      say = say(T, GI, "THE_FERRY_IS_LEAVING_FOR_GIRAN_HARBOR"),
      horn = T,
      pay = { item = boat.TICKET_TALKING_TO_GIRAN, count = 1, oust = { -96777, 258970, -3623 } },
      path = TALKING_TO_GIRAN,
      wait = 200000,
    },
    -- 13
    { say = say(GI, T, "THE_FERRY_FROM_TALKING_ISLAND_WILL_BE_ARRIVING_AT_GIRAN_HARBOR_IN_APPROXIMATELY_20_MINUTES"), wait = 300000 },
    -- 14
    { say = say(GI, T, "THE_FERRY_FROM_TALKING_ISLAND_WILL_BE_ARRIVING_AT_GIRAN_HARBOR_IN_APPROXIMATELY_15_MINUTES"), wait = 300000 },
    -- 15
    { say = say(GI, T, "THE_FERRY_FROM_TALKING_ISLAND_WILL_BE_ARRIVING_AT_GIRAN_HARBOR_IN_APPROXIMATELY_10_MINUTES"), wait = 300000 },
    -- 16
    { say = say(GI, T, "THE_FERRY_FROM_TALKING_ISLAND_WILL_BE_ARRIVING_AT_GIRAN_HARBOR_IN_APPROXIMATELY_5_MINUTES"), wait = 240000 },
    -- 17: the end of the path runs the next step.
    { say = say(GI, T, "THE_FERRY_FROM_TALKING_ISLAND_WILL_BE_ARRIVING_AT_GIRAN_HARBOR_IN_APPROXIMATELY_1_MINUTE") },
    -- 18
    {
      say = say(GI, T, "THE_FERRY_HAS_ARRIVED_AT_GIRAN_HARBOR", "THE_FERRY_WILL_LEAVE_FOR_TALKING_ISLAND_HARBOR_AFTER_ANCHORING_FOR_TEN_MINUTES"),
      horn = GI,
      wait = 300000,
    },
  },
}
