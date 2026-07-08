package tck

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// ParseValue parses a single openCypher TCK value-string into a canonical
// Value. The grammar (per the TCK string representation) covers:
//
//	null | true | false | integer | float | 'string'
//	[ list ] | { map } | ( node ) | [ relationship ] | < path >
func ParseValue(s string) (Value, error) {
	p := &valParser{src: s}
	p.ws()
	v, err := p.value()
	if err != nil {
		return Value{}, err
	}
	p.ws()
	if p.pos != len(p.src) {
		return Value{}, fmt.Errorf("trailing input at %d in %q", p.pos, s)
	}
	return v, nil
}

type valParser struct {
	src string
	pos int
}

func (p *valParser) ws() {
	for p.pos < len(p.src) && (p.src[p.pos] == ' ' || p.src[p.pos] == '\t' || p.src[p.pos] == '\n') {
		p.pos++
	}
}

func (p *valParser) peek() byte {
	if p.pos < len(p.src) {
		return p.src[p.pos]
	}
	return 0
}

func (p *valParser) value() (Value, error) {
	p.ws()
	switch c := p.peek(); {
	case c == '\'':
		return p.string()
	case c == '[':
		return p.listOrRel()
	case c == '{':
		return p.mapVal()
	case c == '(':
		return p.node()
	case c == '<':
		return p.path()
	default:
		return p.literal()
	}
}

func (p *valParser) string() (Value, error) {
	// opening quote
	p.pos++
	var b strings.Builder
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == '\\' && p.pos+1 < len(p.src) {
			nx := p.src[p.pos+1]
			switch nx {
			case '\'', '\\':
				b.WriteByte(nx)
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			default:
				b.WriteByte(nx)
			}
			p.pos += 2
			continue
		}
		if c == '\'' {
			p.pos++
			return Value{Kind: KString, Str: b.String()}, nil
		}
		b.WriteByte(c)
		p.pos++
	}
	return Value{}, fmt.Errorf("unterminated string in %q", p.src)
}

func (p *valParser) literal() (Value, error) {
	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == ',' || c == ']' || c == '}' || c == ')' || c == '>' || c == ' ' {
			break
		}
		p.pos++
	}
	tok := p.src[start:p.pos]
	switch tok {
	case "null":
		return Value{Kind: KNull}, nil
	case "true":
		return Value{Kind: KBool, Bool: true}, nil
	case "false":
		return Value{Kind: KBool, Bool: false}, nil
	case "NaN":
		return Value{Kind: KFloat, Float: math.NaN()}, nil
	case "Inf", "+Inf", "Infinity":
		return Value{Kind: KFloat, Float: math.Inf(1)}, nil
	case "-Inf", "-Infinity":
		return Value{Kind: KFloat, Float: math.Inf(-1)}, nil
	}
	if i, err := strconv.ParseInt(tok, 10, 64); err == nil {
		return Value{Kind: KInt, Int: i}, nil
	}
	if f, err := strconv.ParseFloat(tok, 64); err == nil {
		return Value{Kind: KFloat, Float: f}, nil
	}
	return Value{}, fmt.Errorf("unrecognised literal %q in %q", tok, p.src)
}

func (p *valParser) listOrRel() (Value, error) {
	// '[' is at the cursor. A relationship's first non-space char after the
	// bracket is ':' (e.g. [:KNOWS {..}]); a list starts with a value or ']'.
	save := p.pos
	p.pos++ // consume '['
	p.ws()
	isRel := p.peek() == ':'
	p.pos = save
	if isRel {
		return p.relationship()
	}
	return p.list()
}

func (p *valParser) list() (Value, error) {
	p.pos++ // '['
	out := Value{Kind: KList, List: []Value{}}
	p.ws()
	if p.peek() == ']' {
		p.pos++
		return out, nil
	}
	for {
		v, err := p.value()
		if err != nil {
			return Value{}, err
		}
		out.List = append(out.List, v)
		p.ws()
		switch p.peek() {
		case ',':
			p.pos++
		case ']':
			p.pos++
			return out, nil
		default:
			return Value{}, fmt.Errorf("expected , or ] in list at %d of %q", p.pos, p.src)
		}
	}
}

func (p *valParser) relationship() (Value, error) {
	p.pos++ // '['
	p.ws()
	v := Value{Kind: KRel}
	if p.peek() == ':' {
		p.pos++
		v.Labels = []string{p.ident()}
	}
	p.ws()
	if p.peek() == '{' {
		m, err := p.mapVal()
		if err != nil {
			return Value{}, err
		}
		v.Map = m.Map
	}
	p.ws()
	if p.peek() != ']' {
		return Value{}, fmt.Errorf("expected ] closing relationship at %d of %q", p.pos, p.src)
	}
	p.pos++
	return v, nil
}

func (p *valParser) mapVal() (Value, error) {
	p.pos++ // '{'
	out := Value{Kind: KMap, Map: map[string]Value{}}
	p.ws()
	if p.peek() == '}' {
		p.pos++
		return out, nil
	}
	for {
		p.ws()
		key := p.ident()
		if key == "" {
			return Value{}, fmt.Errorf("expected map key at %d of %q", p.pos, p.src)
		}
		p.ws()
		if p.peek() != ':' {
			return Value{}, fmt.Errorf("expected : after key %q at %d of %q", key, p.pos, p.src)
		}
		p.pos++
		v, err := p.value()
		if err != nil {
			return Value{}, err
		}
		out.Map[key] = v
		p.ws()
		switch p.peek() {
		case ',':
			p.pos++
		case '}':
			p.pos++
			return out, nil
		default:
			return Value{}, fmt.Errorf("expected , or } in map at %d of %q", p.pos, p.src)
		}
	}
}

func (p *valParser) node() (Value, error) {
	p.pos++ // '('
	v := Value{Kind: KNode}
	p.ws()
	for p.peek() == ':' {
		p.pos++
		v.Labels = append(v.Labels, p.ident())
		p.ws()
	}
	if p.peek() == '{' {
		m, err := p.mapVal()
		if err != nil {
			return Value{}, err
		}
		v.Map = m.Map
		p.ws()
	}
	if p.peek() != ')' {
		return Value{}, fmt.Errorf("expected ) closing node at %d of %q", p.pos, p.src)
	}
	p.pos++
	return v, nil
}

// path: <(node)-[rel]->(node)...> — we collect the alternating node/rel
// values; direction arrows are structural and ignored for equality purposes
// beyond ordering.
func (p *valParser) path() (Value, error) {
	p.pos++ // '<'
	v := Value{Kind: KPath}
	for p.pos < len(p.src) && p.peek() != '>' {
		switch p.peek() {
		case '(':
			n, err := p.node()
			if err != nil {
				return Value{}, err
			}
			v.Path = append(v.Path, n)
		case '[':
			r, err := p.relationship()
			if err != nil {
				return Value{}, err
			}
			v.Path = append(v.Path, r)
		case '-', '<', '>':
			p.pos++ // arrow glyphs
		default:
			p.pos++
		}
	}
	if p.peek() != '>' {
		return Value{}, fmt.Errorf("unterminated path in %q", p.src)
	}
	p.pos++
	return v, nil
}

func (p *valParser) ident() string {
	start := p.pos
	// backtick-quoted identifiers
	if p.peek() == '`' {
		p.pos++
		s := p.pos
		for p.pos < len(p.src) && p.src[p.pos] != '`' {
			p.pos++
		}
		id := p.src[s:p.pos]
		if p.peek() == '`' {
			p.pos++
		}
		return id
	}
	for p.pos < len(p.src) {
		c := rune(p.src[p.pos])
		if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' {
			p.pos++
			continue
		}
		break
	}
	return p.src[start:p.pos]
}
