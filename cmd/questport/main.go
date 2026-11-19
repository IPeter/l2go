// Command questport translates the L2J Mobius quest scripts (Java) into
// Lua scripts for the L2Go quest API.
//
// Usage:
//
//	questport -mobius <dir> -out <dir> [-quest Q00303_CollectArrowheads ...] [-report]
//
// -mobius is the Mobius Interlude checkout (or its quests folder). Every
// quest becomes <out>/<QuestName>/<QuestName>.lua; what could not be
// translated becomes a J.todo marker, listed in <out>/REPORT.txt.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// questList collects repeated -quest flags.
type questList []string

func (q *questList) String() string { return strings.Join(*q, ",") }

func (q *questList) Set(v string) error {
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			*q = append(*q, s)
		}
	}
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "questport:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("questport", flag.ContinueOnError)
	mobius := fs.String("mobius", "", "Mobius Interlude checkout or its dist/game/data/scripts/quests folder")
	out := fs.String("out", "", "output folder")
	report := fs.Bool("report", false, "print the coverage report")
	parseOnly := fs.Bool("parse-only", false, "only parse the Java files and report syntax errors")
	var quests questList
	fs.Var(&quests, "quest", "translate only this quest (repeatable, or comma separated)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *mobius == "" {
		return fmt.Errorf("-mobius is required")
	}
	dir, err := questsDir(*mobius)
	if err != nil {
		return err
	}
	names, err := questNames(dir, quests)
	if err != nil {
		return err
	}
	if *parseOnly {
		return parseAll(dir, names)
	}
	if *out == "" {
		return fmt.Errorf("-out is required")
	}
	rep, err := translateAll(dir, names, *out)
	if err != nil {
		return err
	}
	text := rep.String()
	if err := os.WriteFile(filepath.Join(*out, "REPORT.txt"), []byte(text), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	if *report {
		fmt.Print(text)
	} else {
		fmt.Print(rep.Summary())
	}
	return nil
}

// questsDir finds the quests folder under a Mobius checkout.
func questsDir(mobius string) (string, error) {
	for _, d := range []string{
		filepath.Join(mobius, "dist", "game", "data", "scripts", "quests"),
		filepath.Join(mobius, "L2J_Mobius_CT_0_Interlude", "dist", "game", "data", "scripts", "quests"),
		mobius,
	} {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), "Q") {
				return d, nil
			}
		}
	}
	return "", fmt.Errorf("no quests folder found under %s", mobius)
}

// questNames lists the quest folders to translate: all of them, or the
// requested ones.
func questNames(dir string, only []string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read quests folder: %w", err)
	}
	var all []string
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "Q") {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), e.Name()+".java")); err == nil {
			all = append(all, e.Name())
		}
	}
	slices.Sort(all)
	if len(only) == 0 {
		return all, nil
	}
	var names []string
	for _, q := range only {
		found := ""
		for _, n := range all {
			if n == q || strings.HasPrefix(n, q+"_") {
				found = n
				break
			}
		}
		if found == "" {
			return nil, fmt.Errorf("quest %s not found in %s", q, dir)
		}
		names = append(names, found)
	}
	return names, nil
}

// parseAll parses every quest and prints the syntax errors.
func parseAll(dir string, names []string) error {
	failed := 0
	for _, n := range names {
		path := filepath.Join(dir, n, n+".java")
		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if _, err := parseJava(path, string(src)); err != nil {
			fmt.Println(err)
			failed++
		}
	}
	fmt.Printf("parsed %d files, %d failed\n", len(names), failed)
	return nil
}
