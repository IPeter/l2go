package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// note is one J.todo marker or review remark in a translated script.
type note struct {
	line   int    // Java line
	key    string // what was not mapped, e.g. "Npc.getAI", for the totals
	reason string // shown in the marker
}

// questResult is the outcome of translating one quest.
type questResult struct {
	name     string
	out      string // output file, empty when the quest failed
	err      error  // parse failure
	todos    []note
	reviews  []note
	requires []string
	api      map[string]int // Lua API functions used, with counts
	saga     bool
}

// report collects the results of a run.
type report struct {
	results []*questResult
}

func (r *report) totals() (clean, withTodos, failed, markers int) {
	for _, q := range r.results {
		switch {
		case q.err != nil:
			failed++
		case len(q.todos) == 0:
			clean++
		default:
			withTodos++
		}
		markers += len(q.todos)
	}
	return
}

// Summary is the short report printed after a run.
func (r *report) Summary() string {
	clean, withTodos, failed, markers := r.totals()
	return fmt.Sprintf("questport: %d quests, %d without J.todo, %d with J.todo (%d markers), %d failed to parse\n",
		len(r.results), clean, withTodos, markers, failed)
}

// keyCount is how often one kind of note occurs.
type keyCount struct {
	key    string
	count  int // notes
	quests int // quests with at least one
}

// topKeys counts the notes by key, most common first.
func topKeys(results []*questResult, pick func(*questResult) []note) []keyCount {
	type agg struct {
		count  int
		quests map[string]bool
	}
	m := map[string]*agg{}
	for _, q := range results {
		for _, n := range pick(q) {
			a := m[n.key]
			if a == nil {
				a = &agg{quests: map[string]bool{}}
				m[n.key] = a
			}
			a.count++
			a.quests[q.name] = true
		}
	}
	out := make([]keyCount, 0, len(m))
	for k, a := range m {
		out = append(out, keyCount{key: k, count: a.count, quests: len(a.quests)})
	}
	slices.SortFunc(out, func(a, b keyCount) int {
		if c := cmp.Compare(b.count, a.count); c != 0 {
			return c
		}
		return cmp.Compare(a.key, b.key)
	})
	return out
}

// String is the full coverage report.
func (r *report) String() string {
	var b strings.Builder
	clean, withTodos, failed, markers := r.totals()
	b.WriteString("questport coverage report\n")
	b.WriteString("=========================\n\n")
	fmt.Fprintf(&b, "Quests:              %d\n", len(r.results))
	fmt.Fprintf(&b, "Without J.todo:      %d\n", clean)
	fmt.Fprintf(&b, "With J.todo:         %d\n", withTodos)
	fmt.Fprintf(&b, "Failed to parse:     %d\n", failed)
	fmt.Fprintf(&b, "J.todo markers:      %d\n", markers)
	reviews := 0
	for _, q := range r.results {
		reviews += len(q.reviews)
	}
	fmt.Fprintf(&b, "Review notes:        %d\n\n", reviews)

	b.WriteString("Most common unmapped constructs (markers, quests)\n")
	b.WriteString("-------------------------------------------------\n")
	for i, k := range topKeys(r.results, func(q *questResult) []note { return q.todos }) {
		if i == 60 {
			break
		}
		fmt.Fprintf(&b, "%5d %4d  %s\n", k.count, k.quests, k.key)
	}

	b.WriteString("\nReview notes by kind (notes, quests)\n")
	b.WriteString("------------------------------------\n")
	for _, k := range topKeys(r.results, func(q *questResult) []note { return q.reviews }) {
		fmt.Fprintf(&b, "%5d %4d  %s\n", k.count, k.quests, k.key)
	}

	if failed > 0 {
		b.WriteString("\nFailed to parse\n")
		b.WriteString("---------------\n")
		for _, q := range r.results {
			if q.err != nil {
				fmt.Fprintf(&b, "%s\n", q.err)
			}
		}
	}

	b.WriteString("\nRequired features (q:requires)\n")
	b.WriteString("------------------------------\n")
	byFeature := map[string][]string{}
	for _, q := range r.results {
		for _, f := range q.requires {
			byFeature[f] = append(byFeature[f], q.name)
		}
	}
	for _, f := range sortedKeys(byFeature) {
		fmt.Fprintf(&b, "%-12s %d: %s\n", f, len(byFeature[f]), strings.Join(byFeature[f], " "))
	}

	b.WriteString("\nLua API used (calls, quests)\n")
	b.WriteString("----------------------------\n")
	type use struct{ calls, quests int }
	api := map[string]*use{}
	for _, q := range r.results {
		for name, n := range q.api {
			u := api[name]
			if u == nil {
				u = &use{}
				api[name] = u
			}
			u.calls += n
			u.quests++
		}
	}
	for _, name := range sortedKeys(api) {
		fmt.Fprintf(&b, "%6d %4d  %s\n", api[name].calls, api[name].quests, name)
	}

	b.WriteString("\nPer quest (J.todo markers, review notes)\n")
	b.WriteString("----------------------------------------\n")
	for _, q := range r.results {
		switch {
		case q.err != nil:
			fmt.Fprintf(&b, "%-50s parse error\n", q.name)
			continue
		default:
			kind := ""
			if q.saga {
				kind = " (saga data)"
			}
			fmt.Fprintf(&b, "%-50s %3d %3d%s\n", q.name, len(q.todos), len(q.reviews), kind)
		}
		for _, n := range q.todos {
			fmt.Fprintf(&b, "    todo   java:%d: %s\n", n.line, n.reason)
		}
		for _, n := range q.reviews {
			fmt.Fprintf(&b, "    review java:%d: %s\n", n.line, n.reason)
		}
	}
	return b.String()
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
