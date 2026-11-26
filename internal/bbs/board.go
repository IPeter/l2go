// Package bbs holds the rules of the community board (Alt+B, Plan 14 M11o)
// that need no game state: how a page is split into ShowBoard parts, which
// board a bypass or a "Write" button goes to, the mail box rules and the
// pages Mobius builds in code. internal/game adapts them to the players,
// the clans and the store.
//
// The reference is L2J Mobius Interlude: CommunityBoardHandler,
// HtmlUtil.sendCBHtml and the handlers in
// dp/scripts/handlers/bypass/communityboard/.
package bbs

import (
	"strings"
	"unicode/utf16"
)

// The page part limits of HtmlUtil.sendCBHtml, in UTF-16 code units: a page
// shorter than PartLength goes in part 101; one shorter than 2 × PartLength
// is split into parts 101 and 102; one shorter than MaxPageLength into 101,
// 102 and 103. A longer page shows TooLongPage.
const (
	PartLength    = 4090
	MaxPageLength = 12270
)

// TooLongPage is what Mobius shows instead of a page too long for the
// three parts.
const TooLongPage = "<html><body><br><center>Error: HTML was too long!</center></body></html>"

// MissingPart is how Mobius writes a missing page part: ShowBoard(null,
// "102") concatenates the Java null into its content.
const MissingPart = "null"

// PartIDs are the IDs of the three page parts.
var PartIDs = [3]string{"101", "102", "103"}

// EditPageID is the ID of a page sent whole because a multiedit field of it
// is filled right after (HtmlUtil.sendCBHtml with fillMultiEdit).
const EditPageID = "1001"

// Parts splits a page into the texts of parts 101, 102 and 103 (Mobius
// HtmlUtil.sendCBHtml): MissingPart stands for a part the page does not
// fill. Lengths count UTF-16 code units, like the Java String. A character
// that takes a surrogate pair across a cut goes whole into the next part,
// where Java splits the pair.
func Parts(html string) [3]string {
	n := utf16Len(html)
	switch {
	case n < PartLength:
		return [3]string{html, MissingPart, MissingPart}
	case n < 2*PartLength:
		a, b := cutUnits(html, PartLength)
		return [3]string{a, b, MissingPart}
	case n < MaxPageLength:
		a, rest := cutUnits(html, PartLength)
		b, c := cutUnits(rest, PartLength)
		return [3]string{a, b, c}
	}
	return [3]string{TooLongPage, MissingPart, MissingPart}
}

// utf16Len is the length of s in UTF-16 code units (Java String.length).
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

// cutUnits cuts s after n UTF-16 code units.
func cutUnits(s string, n int) (string, string) {
	units := 0
	for i, r := range s {
		size := utf16.RuneLen(r)
		if units+size > n {
			return s[:i], s[i:]
		}
		units += size
	}
	return s, ""
}

// Abbreviate keeps the first n UTF-16 code units of s (Mobius
// MailBoard.abbreviate).
func Abbreviate(s string, n int) string {
	a, _ := cutUnits(s, n)
	return a
}

// Board is one of the board handlers.
type Board int

// The boards (Mobius dp/scripts/handlers/bypass/communityboard). The drop
// search board stays off (decision X8): it is only linked from the custom
// board, so its commands are not registered.
const (
	BoardNone Board = iota
	BoardHome
	BoardFavorite
	BoardRegion
	BoardClan
	BoardMemo
	BoardMail
	BoardFriends
	BoardHomepage
)

// commands are the commands each board registers (IParseBoardHandler
// getCommandList). The custom commands of HomeBoard (_bbsmultisell,
// _bbsteleport, _bbsbuff, ...) belong to the custom board and are left out.
var commands = []struct {
	command string
	board   Board
}{
	{"_bbshome", BoardHome},
	{"_bbstop", BoardHome},
	{"_bbsgetfav", BoardFavorite},
	{"bbs_add_fav", BoardFavorite},
	{"_bbsdelfav_", BoardFavorite},
	{"_bbsloc", BoardRegion},
	{"_bbsclan", BoardClan},
	{"_bbsclan_list", BoardClan},
	{"_bbsclan_clanhome", BoardClan},
	{"_bbsmemo", BoardMemo},
	{"_bbstopics", BoardMemo},
	{"_bbsmail", BoardMail},
	{"_maillist_0_1_0_", BoardMail},
	{"_friendlist", BoardFriends},
	{"_friendblocklist", BoardFriends},
	{"_friend", BoardFriends},
	{"_block", BoardFriends},
	{"_bbslink", BoardHomepage},
}

// Route returns the board of a bypass (Mobius
// CommunityBoardHandler.getHandler): the board of a registered command the
// bypass starts with, ignoring case. Mobius takes the first match in hash
// order, which only matters for "_bbstopics" (also starting with "_bbstop");
// Route takes the longest match, the memo board.
func Route(command string) (Board, bool) {
	lower := strings.ToLower(command)
	best, bestLen := BoardNone, -1
	for _, c := range commands {
		if strings.HasPrefix(lower, c.command) && len(c.command) > bestLen {
			best, bestLen = c.board, len(c.command)
		}
	}
	return best, bestLen >= 0
}

// WriteTarget returns the board a "Write" button's URL goes to (Mobius
// CommunityBoardHandler.handleWriteCommand). ok is false for a URL Mobius
// does not know, which shows NotImplementedPage. A target may still not take
// writes: the home board ("Topic") does not, and no board registers "Post".
func WriteTarget(url string) (b Board, ok bool) {
	switch url {
	case "Topic":
		return BoardHome, true
	case "Post":
		return BoardNone, true // "_bbspos": no board has it
	case "Region":
		return BoardRegion, true
	case "Notice", "Search", "CMail":
		return BoardClan, true
	case "Mail":
		return BoardMail, true
	}
	return BoardNone, false
}

// NotImplementedPage is the page for a "Write" URL Mobius does not know.
func NotImplementedPage(url string) string {
	return "<html><body><br><br><center>The command: " + url + " is not implemented yet.</center><br><br></body></html>"
}

// SplitJava splits s at every sep like the Java String.split with a literal
// one-character separator: trailing empty strings are removed, and a string
// without the separator is returned whole (also when empty).
func SplitJava(s, sep string) []string {
	if !strings.Contains(s, sep) {
		return []string{s}
	}
	parts := strings.Split(s, sep)
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// JavaTrim trims s like the Java String.trim: every character up to U+0020
// at both ends.
func JavaTrim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r <= ' ' })
}

// Tokens splits s like the Java StringTokenizer with one delimiter: empty
// tokens are skipped.
func Tokens(s string, delim rune) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == delim })
}

// IsNumeric reports whether s is a non-empty string of ASCII digits (Mobius
// StringUtil.isNumeric).
func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
