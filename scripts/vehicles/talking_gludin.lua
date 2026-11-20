-- The ferry between Talking Island and Gludin (Mobius
-- vehicles/BoatTalkingGludin.java). Tickets 1074 and 1075.

local boat = require("lib.boat")
local say = boat.say

-- Time: 919 s.
local TALKING_TO_GLUDIN = {
  { -121385, 261660, -3610, 180, 800 },
  { -127694, 253312, -3610, 200, 800 },
  { -129274, 237060, -3610, 250, 800 },
  { -114688, 139040, -3610, 200, 800 },
  { -109663, 135704, -3610, 180, 800 },
  { -102151, 135704, -3610, 180, 800 },
  { -96686, 140595, -3610, 180, 800 },
  { -95686, 147718, -3610, 180, 800 },
  { -95686, 148718, -3610, 180, 800 },
  { -95686, 149718, -3610, 150, 800 },
}

local GLUDIN_DOCK = { -95686, 150514, -3610, 150, 800 }

-- Time: 780 s.
local GLUDIN_TO_TALKING = {
  { -95686, 155514, -3610, 180, 800 },
  { -95686, 185514, -3610, 250, 800 },
  { -60136, 238816, -3610, 200, 800 },
  { -60520, 259609, -3610, 180, 1800 },
  { -65344, 261460, -3610, 180, 1800 },
  { -83344, 261560, -3610, 180, 1800 },
  { -88344, 261660, -3610, 180, 1800 },
  { -92344, 261660, -3610, 150, 1800 },
  { -94242, 261659, -3610, 150, 1800 },
}

local TALKING_DOCK = { -96622, 261660, -3610, 150, 1800 }

local T, G = TALKING_DOCK, GLUDIN_DOCK

boat.route {
  name = "TalkingGludin",
  boat = 1,
  start = { -96622, 261660, -3610, 32768 },
  dock = boat.TALKING_ISLAND,
  steps = {
    -- 0
    { say = say(T, G, "THE_FERRY_WILL_LEAVE_FOR_GLUDIN_HARBOR_IN_FIVE_MINUTES"), wait = 240000 },
    -- 1
    {
      say = say(T, G, "THE_FERRY_WILL_LEAVE_FOR_GLUDIN_HARBOR_IN_ONE_MINUTE", "THOSE_WISHING_TO_RIDE_SHOULD_MAKE_HASTE_TO_GET_ON"),
      wait = 40000,
    },
    -- 2
    { say = say(T, G, "THE_FERRY_WILL_BE_LEAVING_SOON_FOR_GLUDIN_HARBOR"), wait = 20000 },
    -- 3
    {
      undock = boat.TALKING_ISLAND,
      say = say(T, G, "THE_FERRY_IS_LEAVING_FOR_GLUDIN_HARBOR"),
      horn = T,
      pay = { item = boat.TICKET_TALKING_TO_GLUDIN, count = 1, oust = { -96777, 258970, -3623 } },
      path = TALKING_TO_GLUDIN,
      wait = 300000,
    },
    -- 4
    { say = say(G, T, "THE_FERRY_FROM_TALKING_ISLAND_WILL_ARRIVE_AT_GLUDIN_HARBOR_IN_APPROXIMATELY_10_MINUTES"), wait = 300000 },
    -- 5
    { say = say(G, T, "THE_FERRY_FROM_TALKING_ISLAND_WILL_BE_ARRIVING_AT_GLUDIN_HARBOR_IN_APPROXIMATELY_5_MINUTES"), wait = 240000 },
    -- 6: the end of the path runs the next step.
    { say = say(G, T, "THE_FERRY_FROM_TALKING_ISLAND_WILL_BE_ARRIVING_AT_GLUDIN_HARBOR_IN_APPROXIMATELY_1_MINUTE") },
    -- 7
    {
      await = boat.await(boat.GLUDIN_HARBOR, G, T, "THE_FERRY_FROM_TALKING_ISLAND_TO_GLUDIN_HARBOR_HAS_BEEN_DELAYED"),
      path = { GLUDIN_DOCK },
    },
    -- 8
    {
      dock = boat.GLUDIN_HARBOR,
      say = say(G, T, "THE_FERRY_HAS_ARRIVED_AT_GLUDIN_HARBOR", "THE_FERRY_WILL_LEAVE_FOR_TALKING_ISLAND_HARBOR_AFTER_ANCHORING_FOR_TEN_MINUTES"),
      horn = G,
      wait = 300000,
    },
    -- 9
    { say = say(T, G, "THE_FERRY_WILL_LEAVE_FOR_TALKING_ISLAND_HARBOR_IN_FIVE_MINUTES"), wait = 240000 },
    -- 10
    {
      say = say(T, G, "THE_FERRY_WILL_LEAVE_FOR_TALKING_ISLAND_HARBOR_IN_ONE_MINUTE", "THOSE_WISHING_TO_RIDE_SHOULD_MAKE_HASTE_TO_GET_ON"),
      wait = 40000,
    },
    -- 11
    { say = say(T, G, "THE_FERRY_WILL_BE_LEAVING_SOON_FOR_TALKING_ISLAND_HARBOR"), wait = 20000 },
    -- 12
    {
      undock = boat.GLUDIN_HARBOR,
      say = say(T, G, "THE_FERRY_IS_LEAVING_FOR_TALKING_ISLAND_HARBOR"),
      horn = G,
      pay = { item = boat.TICKET_GLUDIN_TO_TALKING, count = 1, oust = { -90015, 150422, -3610 } },
      path = GLUDIN_TO_TALKING,
      wait = 150000,
    },
    -- 13
    { say = say(T, G, "THE_FERRY_FROM_GLUDIN_HARBOR_WILL_BE_ARRIVING_AT_TALKING_ISLAND_IN_APPROXIMATELY_10_MINUTES"), wait = 300000 },
    -- 14
    { say = say(T, G, "THE_FERRY_FROM_GLUDIN_HARBOR_WILL_BE_ARRIVING_AT_TALKING_ISLAND_IN_APPROXIMATELY_5_MINUTES"), wait = 240000 },
    -- 15: the end of the path runs the next step.
    { say = say(T, G, "THE_FERRY_FROM_GLUDIN_HARBOR_WILL_BE_ARRIVING_AT_TALKING_ISLAND_IN_APPROXIMATELY_1_MINUTE") },
    -- 16
    {
      await = boat.await(boat.TALKING_ISLAND, T, G, "THE_FERRY_FROM_GLUDIN_HARBOR_TO_TALKING_ISLAND_HAS_BEEN_DELAYED"),
      path = { TALKING_DOCK },
    },
    -- 17
    {
      dock = boat.TALKING_ISLAND,
      say = say(T, G, "THE_FERRY_ARRIVED_AT_TALKING_ISLAND_HARBOR", "THE_FERRY_WILL_LEAVE_FOR_GLUDIN_HARBOR_AFTER_ANCHORING_FOR_TEN_MINUTES"),
      horn = T,
      wait = 300000,
    },
  },
}
