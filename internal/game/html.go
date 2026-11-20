package game

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// htmlPrefix is how Mobius names the HTML folder in page paths
// ("data/html/default/30006.htm"); the missing-page text and the GM debug
// lines show paths with it, as Mobius does.
const htmlPrefix = "data/html/"

// htmlLoader reads the dialog pages of the HTML folder (Settings.HTMLDir)
// the way Mobius HtmCache does: UTF-8, HTML comments removed, tabs and line
// feeds removed. Pages are read once, when first asked for, and kept, and so
// is the fact that a page is missing (Mobius's lazy cache). Loop-only.
type htmlLoader struct {
	dir   string
	pages map[string]htmlEntry // by clean slash path relative to dir
}

type htmlEntry struct {
	content string
	ok      bool
}

func newHTMLLoader(dir string) *htmlLoader {
	return &htmlLoader{dir: dir, pages: make(map[string]htmlEntry)}
}

// cleanHTMLPath checks a page path relative to the HTML folder and returns
// it cleaned. Mobius refuses any path with ".." (HtmCache.getHtm); absolute
// paths are refused too.
func cleanHTMLPath(rel string) (string, bool) {
	if rel == "" || strings.Contains(rel, "..") || strings.ContainsRune(rel, '\\') {
		return "", false
	}
	p := path.Clean(rel)
	if path.IsAbs(p) || p == "." {
		return "", false
	}
	return p, true
}

// get returns the page at rel (a slash path relative to the HTML folder),
// or false when there is no such page.
func (l *htmlLoader) get(rel string) (string, bool) {
	p, ok := cleanHTMLPath(rel)
	if !ok {
		return "", false
	}
	if e, ok := l.pages[p]; ok {
		return e.content, e.ok
	}
	content, ok := readHTMLFile(filepath.Join(l.dir, filepath.FromSlash(p)))
	l.pages[p] = htmlEntry{content: content, ok: ok}
	return content, ok
}

// exists reports whether the page at rel exists (Mobius getHtmlPath checks
// the file).
func (l *htmlLoader) exists(rel string) bool {
	_, ok := l.get(rel)
	return ok
}

// readHTMLFile reads one page file like Mobius HtmCache.loadFile: only
// .htm and .html files, read as UTF-8 (invalid bytes become U+FFFD), with
// HTML comments and then tabs and line feeds removed (carriage returns
// stay).
func readHTMLFile(name string) (string, bool) {
	lower := strings.ToLower(name)
	if !strings.HasSuffix(lower, ".htm") && !strings.HasSuffix(lower, ".html") {
		return "", false
	}
	b, err := os.ReadFile(name) //nolint:gosec // the path is checked by the callers
	if err != nil {
		return "", false
	}
	s := strings.ToValidUTF8(string(b), "�")
	s = stripHTMLComments(s)
	s = strings.NewReplacer("\t", "", "\n", "").Replace(s)
	return s, true
}

// stripHTMLComments removes every "<!--" ... "-->", shortest match first,
// like the Java replaceAll("(?s)<!--.*?-->", ""). An unclosed comment stays.
func stripHTMLComments(s string) string {
	if !strings.Contains(s, "<!--") {
		return s
	}
	var b strings.Builder
	for {
		i := strings.Index(s, "<!--")
		if i < 0 {
			break
		}
		j := strings.Index(s[i+4:], "-->")
		if j < 0 {
			break
		}
		b.WriteString(s[:i])
		s = s[i+4+j+3:]
	}
	b.WriteString(s)
	return b.String()
}

// missingPage is the page Mobius shows for a missing file
// (AbstractHtmlPacket.setFile).
func missingPage(rel string) string {
	return "<html><body>My Text is missing:<br>" + htmlPrefix + rel + "</body></html>"
}

// htmlPage is a page on its way to a player, with Mobius's
// NpcHtmlMessage steps: the text is prepared (PrepareHTML) when the page is
// made, then placeholders are replaced, then it is sent.
type htmlPage struct {
	html string
	// npc is the NPC the page belongs to (its links are checked against its
	// range), 0 for none.
	npc world.ObjectID
	// itemID is the item a page opened from an item belongs to, else 0.
	itemID int32
}

// newPage makes a page from its text, as Mobius setHtml stores it.
func newPage(npc world.ObjectID, html string) *htmlPage {
	return &htmlPage{html: serverpackets.PrepareHTML(html), npc: npc}
}

// replace replaces every placeholder with value (Mobius
// AbstractHtmlPacket.replace, a literal replace of all occurrences).
func (pg *htmlPage) replace(placeholder, value string) *htmlPage {
	pg.html = strings.ReplaceAll(pg.html, placeholder, value)
	return pg
}

// replaceObjectID fills %objectId% with the object ID of the page's NPC.
func (pg *htmlPage) replaceObjectID(id world.ObjectID) *htmlPage {
	return pg.replace("%objectId%", strconv.Itoa(int(id)))
}

// sendPage records the actions of pg in p's link cache and sends it (Mobius
// NpcHtmlMessage: written, then runImpl caches the actions of the page as
// sent). An item page goes to the item scope, any other to the NPC scope.
func (s *Server) sendPage(p *Player, pg *htmlPage) {
	scope := HTMLScopeNpc
	if pg.itemID != 0 {
		scope = HTMLScopeNpcItem
	}
	p.HTML.Set(scope, pg.npc, pg.html)
	p.Send(serverpackets.NewNpcHtmlMessagePacket(serverpackets.NpcHtmlMessage{
		NpcObjectID: int32(pg.npc),
		HTML:        pg.html,
		ItemID:      pg.itemID,
	}))
}

// htmlFile returns the page at rel of the HTML folder for p (Mobius
// HtmCache.getHtm). A GM is told the path first in the chat, as Mobius does
// with GMDebugHtmlPaths on (its default), also when the page is missing.
func (s *Server) htmlFile(p *Player, rel string) (string, bool) {
	debugHTMLPath(p, "html/"+rel)
	return s.dialog.html.get(rel)
}

// debugHTMLPath tells a GM which page file is read: Mobius sends the path
// below its data folder as general chat from "HTML".
func debugHTMLPath(p *Player, shown string) {
	if p == nil || !p.IsGM() {
		return
	}
	p.Send(serverpackets.NewCreatureSayPacket(serverpackets.CreatureSay{
		Type:       serverpackets.ChatGeneral,
		SenderName: "HTML",
		Text:       shown,
	}))
}

// pageFile makes a page from the file rel of the HTML folder, or Mobius's
// "My Text is missing" page when there is none (NpcHtmlMessage.setFile).
func (s *Server) pageFile(p *Player, npc world.ObjectID, rel string) *htmlPage {
	content, ok := s.htmlFile(p, rel)
	if !ok {
		s.log.Warn("game: missing html page", "page", htmlPrefix+rel)
		content = missingPage(rel)
	}
	return newPage(npc, content)
}

// ShowPage shows p the page file rel of the HTML folder (a path such as
// "help/main.htm"), not bound to an NPC, with %playername% filled: its links
// work wherever p stands. A missing file shows Mobius's missing-page text.
// Loop-only.
func (s *Server) ShowPage(p *Player, rel string) {
	s.sendPage(p, s.pageFile(p, 0, rel).replace("%playername%", p.Name()))
}

// ShowHTML shows p the page html (a whole "<html>...</html>" text or a body
// that gets wrapped), bound to the NPC with object ID npc (0 for none): its
// links are refused once p is out of the NPC's interaction range. Other
// features use it for their pages (GM info pages, for example). Loop-only.
func (s *Server) ShowHTML(p *Player, npc world.ObjectID, html string) {
	s.sendPage(p, newPage(npc, html))
}

// readScriptPage reads a page file a script names, from the script's
// folder dir, then from scripts/<rel>/ of the HTML folder (rel is the
// script's folder relative to the scripts folder; decision Q11), then from
// the HTML folder itself (Mobius Quest.getHtm also falls back to the data
// folder; a "data/html/" prefix names the HTML folder). Script pages are
// read on every use, so edits show without a restart.
func (s *Server) readScriptPage(p *Player, dir, rel, name string) (string, bool) {
	content, ok := s.readScriptDirPage(p, dir, name)
	if ok {
		return content, true
	}
	if clean, okName := cleanHTMLPath(name); okName && rel != "" && !strings.HasPrefix(name, htmlPrefix) {
		page := "scripts/" + rel + "/" + clean
		if content, ok := s.dialog.html.get(page); ok {
			debugHTMLPath(p, "html/"+page)
			return content, true
		}
	}
	if rel, ok := strings.CutPrefix(name, htmlPrefix); ok {
		return s.htmlFile(p, rel)
	}
	return s.htmlFile(p, name)
}

// readScriptDirPage reads a page from the script's folder dir.
func (s *Server) readScriptDirPage(p *Player, dir, name string) (string, bool) {
	if strings.HasPrefix(name, htmlPrefix) {
		return "", false
	}
	if dir != "" {
		if clean, ok := cleanHTMLPath(name); ok {
			file := filepath.Join(dir, filepath.FromSlash(clean))
			shown := file
			if rel, err := filepath.Rel(s.gs.Scripts.Dir, file); err == nil && !strings.HasPrefix(rel, "..") {
				shown = "scripts/" + filepath.ToSlash(rel)
			}
			debugHTMLPath(p, shown)
			if content, ok := readHTMLFile(file); ok {
				return content, true
			}
		}
	}
	return "", false
}

// checkHTMLDir logs a warning when the HTML folder is missing: the dialogs
// still work, with Mobius's missing-page texts.
func (s *Server) checkHTMLDir() {
	if _, err := os.Stat(s.settings.HTMLDir); errors.Is(err, fs.ErrNotExist) {
		s.log.Warn("game: the html folder does not exist; NPC dialogs show missing-page texts (make data copies the pages)", "dir", s.settings.HTMLDir)
	}
}
