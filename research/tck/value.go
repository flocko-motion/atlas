package tck

import (
	"fmt"
	"math"
	"sort"
	"strings"

	graphdb "github.com/mstrYoda/goraphdb"
)

// Kind enumerates the value types in the openCypher TCK value model.
type Kind int

const (
	KNull Kind = iota
	KBool
	KInt
	KFloat
	KString
	KList
	KMap
	KNode
	KRel
	KPath
)

// Value is a canonical, engine-agnostic representation of a Cypher value.
// Both the TCK expected-value strings and goraphdb's actual return values are
// normalised into this form so they can be compared structurally.
type Value struct {
	Kind   Kind
	Bool   bool
	Int    int64
	Float  float64
	Str    string
	List   []Value
	Map    map[string]Value // map entries, or node/relationship properties
	Labels []string         // node labels, or the single relationship type
	Path   []Value          // alternating node/rel values for a path
}

// FromActual converts a value returned by goraphdb into the canonical model.
func FromActual(v any) Value {
	switch x := v.(type) {
	case nil:
		return Value{Kind: KNull}
	case bool:
		return Value{Kind: KBool, Bool: x}
	case int:
		return Value{Kind: KInt, Int: int64(x)}
	case int32:
		return Value{Kind: KInt, Int: int64(x)}
	case int64:
		return Value{Kind: KInt, Int: x}
	case uint64:
		return Value{Kind: KInt, Int: int64(x)}
	case float32:
		return Value{Kind: KFloat, Float: float64(x)}
	case float64:
		// goraphdb returns whole floats as float64; the TCK treats 1 and 1.0
		// as distinct types, so we keep the float kind as-is.
		return Value{Kind: KFloat, Float: x}
	case string:
		return Value{Kind: KString, Str: x}
	case *graphdb.Node:
		return Value{Kind: KNode, Labels: append([]string(nil), x.Labels...), Map: propsToMap(x.Props)}
	case graphdb.Node:
		return Value{Kind: KNode, Labels: append([]string(nil), x.Labels...), Map: propsToMap(x.Props)}
	case *graphdb.Edge:
		return Value{Kind: KRel, Labels: []string{x.Label}, Map: propsToMap(x.Props)}
	case graphdb.Edge:
		return Value{Kind: KRel, Labels: []string{x.Label}, Map: propsToMap(x.Props)}
	case graphdb.Props:
		return Value{Kind: KMap, Map: propsToMap(x)}
	case map[string]any:
		return Value{Kind: KMap, Map: propsToMap(x)}
	case []any:
		l := make([]Value, len(x))
		for i, e := range x {
			l[i] = FromActual(e)
		}
		return Value{Kind: KList, List: l}
	default:
		// Unknown engine type: stringify so comparison fails loudly rather
		// than silently matching.
		return Value{Kind: KString, Str: fmt.Sprintf("<unconvertible %T: %v>", v, v)}
	}
}

// ToGo converts a canonical value back into a plain Go value suitable for
// passing to goraphdb as a query parameter.
func (a Value) ToGo() any {
	switch a.Kind {
	case KNull:
		return nil
	case KBool:
		return a.Bool
	case KInt:
		return a.Int
	case KFloat:
		return a.Float
	case KString:
		return a.Str
	case KList:
		out := make([]any, len(a.List))
		for i, e := range a.List {
			out[i] = e.ToGo()
		}
		return out
	case KMap:
		out := make(map[string]any, len(a.Map))
		for k, v := range a.Map {
			out[k] = v.ToGo()
		}
		return out
	default:
		return nil
	}
}

func propsToMap(p map[string]any) map[string]Value {
	m := make(map[string]Value, len(p))
	for k, v := range p {
		m[k] = FromActual(v)
	}
	return m
}

// Equal reports structural equality under the TCK comparison rules.
// When ignoreListOrder is true, lists are compared as multisets (recursively).
func (a Value) Equal(b Value, ignoreListOrder bool) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case KNull:
		return true
	case KBool:
		return a.Bool == b.Bool
	case KInt:
		return a.Int == b.Int
	case KFloat:
		if math.IsNaN(a.Float) && math.IsNaN(b.Float) {
			return true // TCK: NaN compares equal to NaN in result tables
		}
		return a.Float == b.Float
	case KString:
		return a.Str == b.Str
	case KList:
		return listEqual(a.List, b.List, ignoreListOrder)
	case KMap:
		return mapEqual(a.Map, b.Map, ignoreListOrder)
	case KNode:
		return sameLabels(a.Labels, b.Labels) && mapEqual(a.Map, b.Map, ignoreListOrder)
	case KRel:
		return sameLabels(a.Labels, b.Labels) && mapEqual(a.Map, b.Map, ignoreListOrder)
	case KPath:
		return listEqual(a.Path, b.Path, false)
	}
	return false
}

func listEqual(a, b []Value, ignoreOrder bool) bool {
	if len(a) != len(b) {
		return false
	}
	if !ignoreOrder {
		for i := range a {
			if !a[i].Equal(b[i], ignoreOrder) {
				return false
			}
		}
		return true
	}
	return multisetEqual(a, b, ignoreOrder)
}

// multisetEqual matches each element of a to a distinct element of b.
func multisetEqual(a, b []Value, ignoreOrder bool) bool {
	used := make([]bool, len(b))
	for _, av := range a {
		found := false
		for j, bv := range b {
			if !used[j] && av.Equal(bv, ignoreOrder) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func mapEqual(a, b map[string]Value, ignoreListOrder bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || !av.Equal(bv, ignoreListOrder) {
			return false
		}
	}
	return true
}

func sameLabels(a, b []string) bool {
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

// String renders a Value in TCK-ish notation, for diagnostics.
func (a Value) String() string {
	switch a.Kind {
	case KNull:
		return "null"
	case KBool:
		if a.Bool {
			return "true"
		}
		return "false"
	case KInt:
		return fmt.Sprintf("%d", a.Int)
	case KFloat:
		return fmt.Sprintf("%g", a.Float)
	case KString:
		return "'" + a.Str + "'"
	case KList:
		parts := make([]string, len(a.List))
		for i, e := range a.List {
			parts[i] = e.String()
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case KMap:
		return "{" + mapBody(a.Map) + "}"
	case KNode:
		s := "("
		for _, l := range a.Labels {
			s += ":" + l
		}
		if len(a.Map) > 0 {
			if len(a.Labels) > 0 {
				s += " "
			}
			s += "{" + mapBody(a.Map) + "}"
		}
		return s + ")"
	case KRel:
		t := ""
		if len(a.Labels) > 0 {
			t = ":" + a.Labels[0]
		}
		if len(a.Map) > 0 {
			t += " {" + mapBody(a.Map) + "}"
		}
		return "[" + t + "]"
	case KPath:
		parts := make([]string, len(a.Path))
		for i, e := range a.Path {
			parts[i] = e.String()
		}
		return "<" + strings.Join(parts, "") + ">"
	}
	return "?"
}

func mapBody(m map[string]Value) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + m[k].String()
	}
	return strings.Join(parts, ", ")
}
