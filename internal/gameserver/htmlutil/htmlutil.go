// Package htmlutil finds the actions in the HTML pages the server sends: the
// commands a player can send back by clicking. The game keeps the actions
// of the last pages per player (the link cache) and drops any bypass or
// link that they did not offer.
//
// The rules are those of Mobius HtmlUtil.buildHtmlActionCache, which builds
// the cache, and Player.validateHtmlAction, which checks against it.
package htmlutil

import "strings"

// ParamChar starts a parameter that the client fills in from an input
// field of the page, such as "$amount". A bypass is cached up to and
// including its first ParamChar; it then allows every action that starts
// with the text before it.
const ParamChar = '$'

// LinkPrefix starts the cached form of a link action. RequestLinkHtml
// carries only the path: check it as LinkPrefix + path.
const LinkPrefix = "link "

// The markers Mobius looks for, matched without regard to ASCII case.
const (
	bypassMarker = `="bypass `
	linkMarker   = `="link `
	hideFlag     = "-h "
)

// Actions returns the actions of an HTML page in the order Mobius caches
// them: the bypasses, then the links, each in page order and with repeats
// kept. Pass the page exactly as it is sent, with its placeholders
// replaced.
//
// A bypass is the text from `="bypass ` to the next double quote, without
// the "-h " flag and anything before it, and trimmed. A link is LinkPrefix
// plus the trimmed text from `="link ` to the next double quote. The
// markers match in any ASCII case. A page whose last marker has no closing
// quote gives nothing from that marker on.
//
// Two cases differ from Mobius: an empty bypass is left out (Mobius caches
// it and then fails on every check against it), and an empty link or a
// link with ".." is skipped (Mobius loops forever on one).
func Actions(html string) []string {
	lower := asciiLower(html)
	actions := appendBypasses(nil, html, lower)
	return appendLinks(actions, html, lower)
}

// Matches reports whether a cached action allows the action a player sent
// (Mobius Player.validateHtmlAction). A cached action that ends in
// ParamChar allows every action that starts with the text before it,
// trimmed; any other cached action allows only itself. An empty cached
// action allows nothing (Mobius throws on one).
//
// As in Mobius, a cached action of just ParamChar allows every action.
func Matches(cached, action string) bool {
	n := len(cached)
	switch {
	case n == 0:
		return false
	case cached[n-1] == ParamChar:
		return strings.HasPrefix(action, javaTrim(cached[:n-1]))
	}
	return cached == action
}

// appendBypasses appends the bypass actions of html to dst. lower is html
// with its ASCII letters in lower case.
func appendBypasses(dst []string, html, lower string) []string {
	end := 0
	for {
		i := strings.Index(lower[end:], bypassMarker)
		if i < 0 {
			return dst
		}
		start := end + i + len(bypassMarker)
		j := strings.IndexByte(lower[start:], '"')
		if j < 0 {
			return dst
		}
		end = start + j

		from := start
		if h := strings.Index(lower[start:end], hideFlag); h >= 0 {
			from = start + h + len(hideFlag)
		}
		bypass := javaTrim(html[from:end])
		if p := strings.IndexByte(bypass, ParamChar); p >= 0 {
			bypass = bypass[:p+1]
		}
		if bypass != "" {
			dst = append(dst, bypass)
		}
	}
}

// appendLinks appends the link actions of html to dst. lower is html with
// its ASCII letters in lower case.
func appendLinks(dst []string, html, lower string) []string {
	end := 0
	for {
		i := strings.Index(lower[end:], linkMarker)
		if i < 0 {
			return dst
		}
		start := end + i + len(linkMarker)
		j := strings.IndexByte(lower[start:], '"')
		if j < 0 {
			return dst
		}
		end = start + j

		link := javaTrim(html[start:end])
		if link == "" || strings.Contains(link, "..") {
			continue
		}
		dst = append(dst, LinkPrefix+link)
	}
}

// asciiLower returns s with its ASCII letters in lower case and every other
// byte kept, so that an offset in the result is the same offset in s.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// javaTrim trims s like the Java String.trim: every character up to
// U+0020 at both ends.
func javaTrim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r <= ' ' })
}
