package main

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// The NPC dialogs are NCSoft's text, so they are copied from the local
// Mobius checkout into the git-ignored data/html folder instead of being
// committed (decision W15).
const (
	// htmlSrcDir is copied whole to htmlOutDir.
	htmlSrcDir = "dist/game/data/html"
	// scriptsSrcDir holds the quest and script pages; its .htm and .html
	// files are copied to htmlOutDir/scripts, keeping their paths, where
	// Mobius reads them as data/scripts/<path>.
	scriptsSrcDir = "dist/game/data/scripts"
	htmlOutDir    = "html"
)

// genHTML copies the HTML pages: dist/game/data/html/** to html/** and
// dist/game/data/scripts/**/*.htm(l) to html/scripts/**.
func genHTML(o options) error {
	pages, err := copyTree(o, htmlSrcDir, htmlOutDir, func(string) bool { return true })
	if err != nil {
		return err
	}
	scripts, err := copyTree(o, scriptsSrcDir, htmlOutDir+"/scripts", isHTMLFile)
	if err != nil {
		return err
	}
	if pages == 0 || scripts == 0 {
		return fmt.Errorf("html: found %d pages in %s and %d in %s; want some in both", pages, htmlSrcDir, scripts, scriptsSrcDir)
	}
	fmt.Fprintf(o.log, "html: %d pages from %s, %d from %s\n", pages, htmlSrcDir, scripts, scriptsSrcDir)
	return nil
}

// isHTMLFile reports whether name is an .htm or .html file.
func isHTMLFile(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	return ext == ".htm" || ext == ".html"
}

// copyTree adds the files below src that keep accepts to the output under
// dst, with the same relative paths, and returns how many it added.
func copyTree(o options, src, dst string, keep func(name string) bool) (int, error) {
	n := 0
	err := fs.WalkDir(o.src, src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !keep(p) {
			return nil
		}
		b, err := fs.ReadFile(o.src, p)
		if err != nil {
			return err
		}
		n++
		return o.out.add(dst+strings.TrimPrefix(p, src), b)
	})
	if err != nil {
		return 0, fmt.Errorf("copy %s: %w", src, err)
	}
	return n, nil
}
