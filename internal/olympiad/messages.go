package olympiad

// Message is a system message of the Olympiad (Mobius SystemMessageId).
type Message int32

// The system messages of the Olympiad and the heroes.
const (
	// MsgEarnedItem: "You have earned $s2 $s1(s)." (the winner's reward).
	MsgEarnedItem Message = 53
	// MsgMovedToStadium: "You will be moved to the Olympiad Stadium in $s1
	// second(s).".
	MsgMovedToStadium Message = 1492
	// MsgOpponentDefaulted: "Your opponent does not meet the requirements
	// to do battle; the match has been cancelled.".
	MsgOpponentDefaulted Message = 1494
	// MsgMatchStartsIn: "The Grand Olympiad match will start in $s1
	// second(s).".
	MsgMatchStartsIn Message = 1495
	// MsgMatchStarted: "The match has started, fight!".
	MsgMatchStarted Message = 1496
	// MsgYouWin: "Congratulations $s1, you win the match!".
	MsgYouWin Message = 1497
	// MsgTie: "There is no victor; the match ends in a tie.".
	MsgTie Message = 1498
	// MsgBackToTown: "You will be moved back to town in $s1 second(s).".
	MsgBackToTown Message = 1499
	// MsgNoSubclass: "You cannot participate in the Grand Olympiad Games
	// with a character in their subclass.".
	MsgNoSubclass Message = 1500
	// MsgOnlyNobles: "Only Noblesse can participate in the Olympiad.".
	MsgOnlyNobles Message = 1501
	// MsgRegisteredClassed: "You have been registered in the Grand
	// Olympiad Games waiting list for a class specific match.".
	MsgRegisteredClassed Message = 1503
	// MsgRegisteredNonClassed: the same for a non-class specific match.
	MsgRegisteredNonClassed Message = 1504
	// MsgRemovedFromList: "You have been removed from the Grand Olympiad
	// Games waiting list.".
	MsgRemovedFromList Message = 1505
	// MsgNotRegistered: "You are not currently registered on any Grand
	// Olympiad Games waiting list.".
	MsgNotRegistered Message = 1506
	// MsgPeriodStarted: "Period $s1 of the Grand Olympiad Games has
	// started!".
	MsgPeriodStarted Message = 1639
	// MsgPeriodEnded: "Period $s1 of the Grand Olympiad Games has now
	// ended.".
	MsgPeriodEnded Message = 1640
	// MsgCompetitionStarted: "Sharpen your swords, ... Battles in the Grand
	// Olympiad Games are now taking place!".
	MsgCompetitionStarted Message = 1641
	// MsgCompetitionEnded: "Much carnage has been left for the cleanup crew
	// of the Olympiad Stadium. Battles in the Grand Olympiad Games are now
	// over!".
	MsgCompetitionEnded Message = 1642
	// MsgNotInProgress: "The Grand Olympiad Games are not currently in
	// progress.".
	MsgNotInProgress Message = 1651
	// MsgEarnedPoints: "$s1 has earned $s2 points in the Grand Olympiad
	// Games.".
	MsgEarnedPoints Message = 1657
	// MsgLostPoints: "$s1 has lost $s2 points in the Grand Olympiad
	// Games.".
	MsgLostPoints Message = 1658
	// MsgAlreadyClassed: "You have already joined the waiting list for a
	// class specific match.".
	MsgAlreadyClassed Message = 1689
	// MsgAlreadyNonClassed: the same for a non-class specific match.
	MsgAlreadyNonClassed Message = 1690
	// MsgTooMuchStuff: "You can't join a Grand Olympiad Game match with that
	// much stuff on you! ...".
	MsgTooMuchStuff Message = 1691
	// MsgSubclassRemoved: "You have changed from your main class to a
	// subclass and therefore are removed from the Grand Olympiad Games
	// waiting list.".
	MsgSubclassRemoved Message = 1692
	// MsgNoObserveWhileWaiting: "You may not observe a Grand Olympiad Games
	// match while you are on the waiting list.".
	MsgNoObserveWhileWaiting Message = 1693
	// MsgCursedWeaponNoOlympiad: "If you possess $s1, you cannot
	// participate in the Olympiad.".
	MsgCursedWeaponNoOlympiad Message = 1750
	// MsgNoRegistrationLast10: "The request to participate in the game
	// cannot be made starting from 10 minutes before the end of the game.".
	MsgNoRegistrationLast10 Message = 1803
	// MsgDeadNoOlympiad: "You cannot participate in the Olympiad while
	// dead.".
	MsgDeadNoOlympiad Message = 1858
	// MsgRegistrationEnded: "The Grand Olympiad registration period has
	// ended.".
	MsgRegistrationEnded Message = 1919
)
