-- Zaken's door, ported from Mobius ai/bosses/Zaken/ZakenDoor (Skache).
--
-- At game midnight the door to Zaken's lair (21240006) opens, and 5 real
-- minutes later it closes again.

local s = script.new("ZakenDoor")

local ZAKEN_DOOR = 21240006
local CLOSE_DELAY_MS = 300000 -- 5 minutes in real time

s:on_day_night_change(function(is_night)
  if not is_night then
    return
  end
  if not doors.open(ZAKEN_DOOR) then
    log.warn("ZakenDoor: the door is not in the door data", "door", ZAKEN_DOOR)
    return
  end
  log.info("Zaken's door opened, game time 00:00", "door", ZAKEN_DOOR)
  s:start_quest_timer("close", CLOSE_DELAY_MS, nil, nil)
end)

s:on_event(function(event)
  if event == "close" then
    if doors.close(ZAKEN_DOOR) then
      log.info("Zaken's door closed", "door", ZAKEN_DOOR)
    else
      log.warn("ZakenDoor: the door is not in the door data", "door", ZAKEN_DOOR)
    end
  end
end)
