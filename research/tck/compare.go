package tck

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/flocko-motion/rankedb/research/cypher-tck/adapter"
)

// ExpectedTable is a parsed TCK result table: a header row of column names
// followed by data rows whose cells are TCK value-strings.
type ExpectedTable struct {
	Columns []string
	Rows    [][]string
}

// CompareResult checks an engine result against an expected TCK table.
//
//	inOrder         — row order must match (Then ... in order)
//	ignoreListOrder — list elements compared as multisets
func CompareResult(actual *adapter.Result, exp ExpectedTable, inOrder, ignoreListOrder bool) error {
	if actual == nil {
		return fmt.Errorf("no result")
	}
	// Columns must match as a set.
	if !sameStringSet(actual.Columns, exp.Columns) {
		return fmt.Errorf("columns mismatch: got %v, want %v", actual.Columns, exp.Columns)
	}

	expRows := make([]map[string]Value, len(exp.Rows))
	for i, cells := range exp.Rows {
		row := make(map[string]Value, len(exp.Columns))
		for j, col := range exp.Columns {
			v, err := ParseValue(cells[j])
			if err != nil {
				return fmt.Errorf("parse expected cell [%d][%s]=%q: %w", i, col, cells[j], err)
			}
			row[col] = v
		}
		expRows[i] = row
	}

	actRows := make([]map[string]Value, len(actual.Rows))
	for i, r := range actual.Rows {
		row := make(map[string]Value, len(exp.Columns))
		for _, col := range exp.Columns {
			row[col] = FromActual(r[col])
		}
		actRows[i] = row
	}

	if len(actRows) != len(expRows) {
		return fmt.Errorf("row count: got %d, want %d\n  got:  %s\n  want: %s",
			len(actRows), len(expRows), renderRows(actRows, exp.Columns), renderRows(expRows, exp.Columns))
	}

	if inOrder {
		for i := range expRows {
			if !rowEqual(actRows[i], expRows[i], exp.Columns, ignoreListOrder) {
				return fmt.Errorf("row %d mismatch:\n  got:  %s\n  want: %s",
					i, renderRow(actRows[i], exp.Columns), renderRow(expRows[i], exp.Columns))
			}
		}
		return nil
	}

	// Unordered: match each expected row to a distinct actual row.
	used := make([]bool, len(actRows))
	for _, er := range expRows {
		found := false
		for j, ar := range actRows {
			if !used[j] && rowEqual(ar, er, exp.Columns, ignoreListOrder) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("missing expected row: %s\n  actual rows: %s",
				renderRow(er, exp.Columns), renderRows(actRows, exp.Columns))
		}
	}
	return nil
}

func rowEqual(a, b map[string]Value, cols []string, ignoreListOrder bool) bool {
	for _, c := range cols {
		if !a[c].Equal(b[c], ignoreListOrder) {
			return false
		}
	}
	return true
}

func renderRow(r map[string]Value, cols []string) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = r[c].String()
	}
	return "| " + strings.Join(parts, " | ") + " |"
}

func renderRows(rows []map[string]Value, cols []string) string {
	if len(rows) == 0 {
		return "(empty)"
	}
	parts := make([]string, len(rows))
	for i, r := range rows {
		parts[i] = renderRow(r, cols)
	}
	return strings.Join(parts, " ")
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

// CompareSideEffects diffs two snapshots and checks the expected TCK side
// effects. Supported keys: +nodes, -nodes, +relationships, -relationships,
// +labels, -labels, +properties, -properties. Label/property deltas are
// approximated from graph-state diffs (see adapter.Snapshot).
func CompareSideEffects(before, after adapter.Snapshot, expected map[string]int) error {
	got := map[string]int{}
	if d := after.Nodes - before.Nodes; d > 0 {
		got["+nodes"] = d
	} else if d < 0 {
		got["-nodes"] = -d
	}
	if d := after.Rels - before.Rels; d > 0 {
		got["+relationships"] = d
	} else if d < 0 {
		got["-relationships"] = -d
	}
	if d := after.Props - before.Props; d > 0 {
		got["+properties"] = d
	} else if d < 0 {
		got["-properties"] = -d
	}
	addLab, delLab := 0, 0
	for l := range after.Labels {
		if _, ok := before.Labels[l]; !ok {
			addLab++
		}
	}
	for l := range before.Labels {
		if _, ok := after.Labels[l]; !ok {
			delLab++
		}
	}
	if addLab > 0 {
		got["+labels"] = addLab
	}
	if delLab > 0 {
		got["-labels"] = delLab
	}

	for k, want := range expected {
		if got[k] != want {
			return fmt.Errorf("side effect %s: got %d, want %d (all observed: %s)", k, got[k], want, fmtMap(got))
		}
	}
	for k, g := range got {
		if _, ok := expected[k]; !ok && g != 0 {
			return fmt.Errorf("unexpected side effect %s=%d (expected: %s)", k, g, fmtMap(expected))
		}
	}
	return nil
}

func fmtMap(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + strconv.Itoa(m[k])
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
