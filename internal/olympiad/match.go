package olympiad

import "time"

// The result of a match (Mobius OlympiadGame.validateWinner) and the
// countdowns of a match (OlympiadGameTask).

// Side is one noble of a match as validateWinner sees it.
type Side struct {
	// Noble is the noble's row; Validate changes its points and counts.
	Noble *Noble
	Name  string
	// Defaulted: the noble did not meet the requirements when the match was
	// to start (checkDefaulted).
	Defaulted bool
	// Crashed: the noble was gone when the match was made, or disconnected
	// after reaching the stadium (_playerOne == null ||
	// _playerOneDisconnected).
	Crashed bool
	// Present: the noble is in the game now (World.getPlayer and isOnline).
	Present bool
	// Dead and HP (current HP + CP) are the noble's state at the end.
	Dead bool
	HP   float64
	// Damage is the damage the noble dealt in the fight.
	Damage int
}

// health is Mobius playerOneHp: HP + CP, 0 when dead.
func (s Side) health() float64 {
	if s.Dead {
		return 0
	}
	return s.HP
}

// ResultKind is how a match ended.
type ResultKind int

// The ends of a match.
const (
	// ResultDefault: a noble did not meet the requirements; it loses a third
	// of its points (at most MaxPoints), and nobody counts a match.
	ResultDefault ResultKind = iota
	// ResultCrash: a noble left the game during the match; it loses, without
	// a reward.
	ResultCrash
	// ResultWin: one noble won.
	ResultWin
	// ResultTie: both stood, or both are gone.
	ResultTie
)

// Notice is a system message validateWinner broadcasts: to the two nobles,
// and with ToAll to the spectators too. Name and Num are $s1 and $s2.
type Notice struct {
	Msg   Message
	Name  string
	Num   int
	ToAll bool
}

// Result is what Validate decided.
type Result struct {
	Kind ResultKind
	// Winner is 0 or 1, -1 without one.
	Winner int
	// PointDiff is the points the match moves.
	PointDiff int
	// Reward is how many reward items the winner gets (ResultWin only).
	Reward int
	// Fight is the olympiad_fights winner to save: 1, 2, 0 for a tie, -1 to
	// save nothing (Mobius saves only the fights it validated in the
	// stadium, ties of two absent nobles excepted).
	Fight   int
	Notices []Notice
	// Countdown: the "moved back to town" countdown follows (not after a
	// default or a crash).
	Countdown bool
}

// PointDiff is Mobius's pointDiff: a fifth (non-classed) or a third
// (classed) of the lower points of the two, at most MaxPoints.
func (c Config) PointDiff(t CompetitionType, a, b int) int {
	div := 5
	if t == Classed {
		div = 3
	}
	return min(min(a, b)/div, c.MaxPoints)
}

// Validate is Mobius validateWinner without the aborted case (an aborted
// match validates nothing): it changes the nobles' points and counts and
// returns the messages to send.
func (c Config) Validate(t CompetitionType, s [2]Side) Result {
	one, two := s[0].Noble, s[1].Noble
	diff := c.PointDiff(t, one.Points, two.Points)
	res := Result{Winner: -1, PointDiff: diff, Fight: -1}
	if s[0].Defaulted || s[1].Defaulted {
		res.Kind = ResultDefault
		for _, x := range s {
			if x.Defaulted {
				lost := min(x.Noble.Points/3, c.MaxPoints)
				x.Noble.Points -= lost
				res.Notices = append(res.Notices, Notice{Msg: MsgLostPoints, Name: x.Name, Num: lost})
			}
		}
		return res
	}
	if s[0].Crashed || s[1].Crashed {
		res.Kind = ResultCrash
		switch {
		case s[0].Crashed && s[1].Crashed:
			one.Points -= diff
			one.Lost++
			two.Points -= diff
			two.Lost++
		default:
			w := 1
			if s[1].Crashed {
				w = 0
			}
			win, lose := s[w].Noble, s[1-w].Noble
			lose.Points -= diff
			lose.Lost++
			win.Points += diff
			win.Won++
			res.Winner = w
			res.Notices = append(res.Notices,
				Notice{Msg: MsgYouWin, Name: s[w].Name, ToAll: true},
				Notice{Msg: MsgEarnedPoints, Name: s[w].Name, Num: diff})
		}
		one.Done++
		two.Done++
		return res
	}

	res.Countdown = true
	hp1, hp2 := s[0].health(), s[1].health()
	reward := c.NonClassedReward
	if t == Classed {
		reward = c.ClassedReward
	}
	winsOver := func(w int) bool {
		me, other := s[w], s[1-w]
		hpMe, hpOther := hp1, hp2
		if w == 1 {
			hpMe, hpOther = hp2, hp1
		}
		return !other.Present || (hpOther == 0 && hpMe != 0) || (me.Damage > other.Damage && hpOther != 0 && hpMe != 0)
	}
	switch {
	case !s[0].Present && !s[1].Present:
		res.Kind = ResultTie
		one.Drawn++
		two.Drawn++
		res.Notices = append(res.Notices, Notice{Msg: MsgTie, ToAll: true})
	case winsOver(0), winsOver(1):
		w := 0
		if !winsOver(0) {
			w = 1
		}
		win, lose := s[w].Noble, s[1-w].Noble
		win.Points += diff
		lose.Points -= diff
		win.Won++
		lose.Lost++
		res.Kind, res.Winner, res.Reward, res.Fight = ResultWin, w, reward, w+1
		res.Notices = append(res.Notices,
			Notice{Msg: MsgYouWin, Name: s[w].Name, ToAll: true},
			Notice{Msg: MsgEarnedPoints, Name: s[w].Name, Num: diff},
			Notice{Msg: MsgLostPoints, Name: s[1-w].Name, Num: diff})
	default:
		res.Kind, res.Fight = ResultTie, 0
		d1 := min(one.Points/5, c.MaxPoints)
		d2 := min(two.Points/5, c.MaxPoints)
		one.Points -= d1
		two.Points -= d2
		one.Drawn++
		two.Drawn++
		res.Notices = append(res.Notices,
			Notice{Msg: MsgTie, ToAll: true},
			Notice{Msg: MsgLostPoints, Name: s[0].Name, Num: d1},
			Notice{Msg: MsgLostPoints, Name: s[1].Name, Num: d2})
	}
	one.Done++
	two.Done++
	return res
}

// Tick is one step of a countdown: the seconds the message shows, then the
// wait before the next step.
type Tick struct {
	Seconds int
	Wait    time.Duration
}

// countdown is Mobius's countdown loops: from start down by step while
// above 0, the step changing after the values in switches.
func countdown(start, step int, switches map[int]int) []Tick {
	var out []Tick
	for i := start; i > 0; i -= step {
		if s, ok := switches[i]; ok {
			step = s
		}
		out = append(out, Tick{Seconds: i, Wait: time.Duration(step) * time.Second})
	}
	return out
}

// TeleportCountdown is OlympiadGameTask.teleportCountdown: "You will be
// moved to the Olympiad Stadium in $s1 second(s)." from WaitTime, every
// minute, then at 60, 30, 15, 10, 5, 4, 3, 2 and 1.
func (c Config) TeleportCountdown() []Tick {
	return countdown(c.WaitTime, 60, map[int]int{60: 30, 30: 15, 15: 5, 5: 1})
}

// StartCountdown is runGame's "The Grand Olympiad match will start in $s1
// second(s)." at 60, 50, 40, 30, 20, 10, 5, 4, 3, 2 and 1 (the damage
// counts start over at 10).
func StartCountdown() []Tick { return countdown(60, 10, map[int]int{10: 5, 5: 1}) }

// BackCountdown is validateWinner's "You will be moved back to town in $s1
// second(s)." at 40, 30, 20, 10, 5, 4, 3, 2 and 1.
func BackCountdown() []Tick { return countdown(40, 10, map[int]int{10: 5, 5: 1}) }

// ArenaWait is the pause between the teleport into the stadium and the
// start countdown (runGame's Thread.sleep(5000)).
const ArenaWait = 5 * time.Second

// BattleCheck is how often a running fight is checked for a winner
// (runGame's 10 s sleeps, up to Battle).
const BattleCheck = 10 * time.Second
