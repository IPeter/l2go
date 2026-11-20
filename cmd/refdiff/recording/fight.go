package recording

import (
	"slices"
	"strconv"
	"strings"
)

// FightEntry is the name of the entry that stands for the swings of a fight.
const FightEntry = "Fight"

// fightPackets are the server packets a fight repeats a random number of
// times: the swings, the HP bars, the hit and miss messages, the chase and
// the server's position corrections, and the idle animations of NPCs
// around.
var fightPackets = map[string]bool{
	"Attack": true, "StatusUpdate": true, "MoveToPawn": true, "MoveToLocation": true,
	"ValidateLocation": true, "StopMove": true, "SetupGauge": true, "SocialAction": true,
}

// fightMessages are the system messages of single swings: 35 "You hit for
// $s1 damage.", 36 and 37 "$s1 hit you for $s2 damage.", 43 "You have
// missed." and 44 "Critical hit!".
var fightMessages = map[string]bool{"35": true, "36": true, "37": true, "43": true, "44": true}

// isFightPacket reports whether e belongs to the swings of a fight.
func isFightPacket(e Entry) bool {
	if e.Dir != DirServer {
		return false
	}
	if e.Name == "SystemMessage" {
		for _, f := range e.Fields {
			if f.Path == "ID" {
				return fightMessages[f.Value]
			}
		}
		return false
	}
	return fightPackets[e.Name]
}

// collapseFights replaces each fight in es by one Fight entry. A fight is a
// run of consecutive server packets of fightPackets (and swing messages)
// with at least one Attack in it. The number of swings, hits and misses of
// a fight is random, so its packets cannot be aligned one by one with
// another run; what stays comparable is who fought whom. The Fight entry
// has the fields Attackers and Targets (sorted canonical names) and the
// counts Swings and Packets, which the combat scenarios ignore. What ends
// the run, such as the experience message, Die or the drops, stays
// compared packet by packet.
func collapseFights(es []Entry) []Entry {
	var out []Entry
	for i := 0; i < len(es); {
		if !isFightPacket(es[i]) {
			out = append(out, es[i])
			i++
			continue
		}
		j := i
		for j < len(es) && isFightPacket(es[j]) {
			j++
		}
		if f, ok := fightEntry(es[i:j]); ok {
			out = append(out, f)
		} else {
			out = append(out, es[i:j]...)
		}
		i = j
	}
	return out
}

// fightEntry summarizes a run of fight packets, or reports false when the
// run has no Attack.
func fightEntry(run []Entry) (Entry, bool) {
	var attackers, targets []string
	swings := 0
	for _, e := range run {
		if e.Name != "Attack" {
			continue
		}
		swings++
		for _, f := range e.Fields {
			switch {
			case f.Path == "AttackerID":
				attackers = append(attackers, f.Value)
			case strings.HasPrefix(f.Path, "Hits[") && strings.HasSuffix(f.Path, "].TargetID"):
				targets = append(targets, f.Value)
			}
		}
	}
	if swings == 0 {
		return Entry{}, false
	}
	return Entry{
		Dir:  DirServer,
		Name: FightEntry,
		Fields: []Field{
			{Path: "Attackers", Value: nameSet(attackers)},
			{Path: "Targets", Value: nameSet(targets)},
			{Path: "Swings", Value: strconv.Itoa(swings)},
			{Path: "Packets", Value: strconv.Itoa(len(run))},
		},
	}, true
}

// nameSet returns the distinct names, sorted, separated by spaces.
func nameSet(names []string) string {
	slices.Sort(names)
	return strings.Join(slices.Compact(names), " ")
}
