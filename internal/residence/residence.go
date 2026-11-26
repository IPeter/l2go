// Package residence holds the rules castles and clan halls share (Plan 14,
// M11i, decision X2): their owner clan, the castle tax and treasury, the
// siege calendar and the siege engine's registration lists, mid victory and
// reputation rules. It is pure: times come from the caller (the world
// clock), and the game (internal/game) adapts the rules to the loop, the
// packets and the storage.
//
// The rules follow Mobius's mechanics/siege package (Siege, Castle,
// SiegeClan) and its managers (SiegeManager, CastleManager,
// MercTicketManager). The siege registry (Registry) is the part of Mobius's
// Siegable that castles and contestable clan halls share.
package residence

// System messages of the siege rules (Mobius SystemMessageId).
const (
	MsgEnteredCombatZone       = 283  // You have entered a combat zone.
	MsgLeftCombatZone          = 284  // You have left a combat zone.
	MsgEngravedRuler           = 285  // Clan $s1 has succeeded in engraving the ruler!
	MsgVictoriousOverSiege     = 291  // Clan $s1 is victorious over $s2's castle siege!
	MsgAnnouncedSiegeTime      = 292  // $s1 has announced the castle siege time.
	MsgRegistrationTermEnded   = 293  // The registration term for $s1 has ended.
	MsgSiegeCanceledNoClans    = 295  // $s1's siege was canceled because there were no clans that participated.
	MsgHoursUntilConclusion    = 358  // $s1 hour(s) until castle siege conclusion.
	MsgMinutesUntilConclusion  = 359  // $s1 minute(s) until castle siege conclusion.
	MsgSecondsLeft             = 360  // Castle siege $s1 second(s) left!
	MsgAlreadyRequested        = 638  // You have already requested a Siege Battle.
	MsgAnotherSiegeSameDay     = 639  // ... already submitted a request for another Siege Battle.
	MsgClanLevelTooLow         = 645  // Only clans of level 4 or higher may register for a castle siege.
	MsgNoMoreAttackers         = 648  // No more registrations may be accepted for the attacker side.
	MsgNoMoreDefenders         = 649  // No more registrations may be accepted for the defender side.
	MsgNotTimeToApprove        = 659  // ... registrations cannot be accepted or rejected.
	MsgNotTimeToRegister       = 660  // ... registration and cancellation cannot be done.
	MsgOwnerIsDefender         = 688  // The clan that owns the castle is automatically registered on the defending side.
	MsgOwnsAnotherCastle       = 689  // A clan that owns a castle cannot participate in another siege.
	MsgAllyOfOwner             = 690  // ... part of an alliance with the clan that owns the castle.
	MsgSiegeStarted            = 711  // The siege of $s1 has started.
	MsgSiegeFinished           = 712  // The siege of $s1 has finished.
	MsgRegistrationDeadline    = 845  // The deadline to register for the siege of $s1 has passed.
	MsgSiegeCanceledNoInterest = 846  // The siege of $s1 has been canceled due to lack of interest.
	MsgSiegeDraw               = 856  // The siege of $s1 has ended in a draw.
	MsgReputationLost          = 1772 // ... $s1 points have been deducted from your clan's reputation score ...
	MsgReputationWon           = 1773 // ... $s1 points have been added to your clan's reputation score.
)
