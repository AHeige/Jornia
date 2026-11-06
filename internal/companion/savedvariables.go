package companion

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// A bounded data-only Lua literal reader. Never execute a SavedVariables file.
// Only WispDB = { ... } is accepted; calls, expressions and other globals are rejected.
type savedReader struct {
	s         string
	i, values int
}

func (p *savedReader) space() {
	for p.i < len(p.s) {
		if strings.ContainsRune(" \t\r\n", rune(p.s[p.i])) {
			p.i++
			continue
		}
		if strings.HasPrefix(p.s[p.i:], "--") {
			end := strings.IndexByte(p.s[p.i:], '\n')
			if end < 0 {
				p.i = len(p.s)
				return
			}
			p.i += end + 1
			continue
		}
		break
	}
}
func (p *savedReader) take(c byte) bool {
	p.space()
	if p.i < len(p.s) && p.s[p.i] == c {
		p.i++
		return true
	}
	return false
}
func (p *savedReader) word() string {
	p.space()
	start := p.i
	for p.i < len(p.s) {
		c := p.s[p.i]
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || p.i > start && c >= '0' && c <= '9') {
			break
		}
		p.i++
	}
	return p.s[start:p.i]
}
func (p *savedReader) quoted() (string, error) {
	p.space()
	if p.i >= len(p.s) || (p.s[p.i] != '"' && p.s[p.i] != '\'') {
		return "", errors.New("expected quoted string")
	}
	quote := p.s[p.i]
	p.i++
	var out strings.Builder
	for p.i < len(p.s) {
		c := p.s[p.i]
		p.i++
		if c == quote {
			if !utf8.ValidString(out.String()) {
				return "", errors.New("invalid UTF-8")
			}
			return out.String(), nil
		}
		if c == '\n' || c == '\r' {
			return "", errors.New("unescaped newline")
		}
		if c != '\\' {
			out.WriteByte(c)
			continue
		}
		if p.i == len(p.s) {
			break
		}
		c = p.s[p.i]
		p.i++
		switch c {
		case '\\', '"', '\'':
			out.WriteByte(c)
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case 'a':
			out.WriteByte(7)
		case 'b':
			out.WriteByte(8)
		case 'f':
			out.WriteByte(12)
		case 'v':
			out.WriteByte(11)
		case '\n':
			out.WriteByte('\n')
		case '\r':
			if p.i < len(p.s) && p.s[p.i] == '\n' {
				p.i++
			}
			out.WriteByte('\n')
		default:
			if c < '0' || c > '9' {
				return "", errors.New("unsupported escape")
			}
			n := int(c - '0')
			for k := 0; k < 2 && p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9'; k++ {
				n = n*10 + int(p.s[p.i]-'0')
				p.i++
			}
			if n > 255 {
				return "", errors.New("invalid byte escape")
			}
			out.WriteByte(byte(n))
		}
	}
	return "", errors.New("unterminated string")
}
func (p *savedReader) value(depth int) (any, error) {
	p.values++
	if depth > 20 || p.values > 250000 {
		return nil, errors.New("SavedVariables limit exceeded")
	}
	p.space()
	if p.i >= len(p.s) {
		return nil, errors.New("incomplete SavedVariables")
	}
	c := p.s[p.i]
	if c == '"' || c == '\'' {
		return p.quoted()
	}
	if c == '{' {
		p.i++
		named := map[string]any{}
		indexed := map[int]any{}
		next := 1
		for !p.take('}') {
			p.space()
			var key string
			index := 0
			if p.take('[') {
				p.space()
				if p.i < len(p.s) && (p.s[p.i] == '"' || p.s[p.i] == '\'') {
					v, err := p.quoted()
					if err != nil {
						return nil, err
					}
					key = v
				} else {
					v, err := p.value(depth + 1)
					if err != nil {
						return nil, err
					}
					n, ok := v.(json.Number)
					if !ok {
						return nil, errors.New("invalid table index")
					}
					index, err = strconv.Atoi(string(n))
					if err != nil || index < 1 || index > 20000 {
						return nil, errors.New("invalid table index")
					}
				}
				if !p.take(']') || !p.take('=') {
					return nil, errors.New("invalid table key")
				}
			} else {
				save := p.i
				word := p.word()
				if word != "" && p.take('=') {
					key = word
				} else {
					p.i = save
					index = next
					next++
				}
			}
			v, err := p.value(depth + 1)
			if err != nil {
				return nil, err
			}
			if index > 0 {
				if _, ok := indexed[index]; ok {
					return nil, errors.New("duplicate key")
				}
				indexed[index] = v
			} else {
				if _, ok := named[key]; ok {
					return nil, errors.New("duplicate key")
				}
				named[key] = v
			}
			if !p.take(',') && !p.take(';') {
				if !p.take('}') {
					return nil, errors.New("missing table separator")
				}
				break
			}
		}
		if len(indexed) > 0 {
			if len(named) > 0 {
				return nil, errors.New("mixed table")
			}
			array := make([]any, len(indexed))
			for i := range array {
				v, ok := indexed[i+1]
				if !ok {
					return nil, errors.New("sparse table")
				}
				array[i] = v
			}
			return array, nil
		}
		return named, nil
	}
	if c == '-' || c >= '0' && c <= '9' {
		start := p.i
		p.i++
		for p.i < len(p.s) && strings.ContainsRune("0123456789.eE+-", rune(p.s[p.i])) {
			p.i++
		}
		n := p.s[start:p.i]
		if !json.Valid([]byte(n)) {
			return nil, errors.New("invalid number")
		}
		return json.Number(n), nil
	}
	switch p.word() {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "nil":
		return nil, nil
	}
	return nil, errors.New("executable or unsupported Lua rejected")
}
func ParseSavedVariables(raw string) ([]Export, error) {
	if len(raw) > 4<<20 || !utf8.ValidString(raw) {
		return nil, errors.New("Wisp-journalen är för stor eller felkodad")
	}
	p := savedReader{s: strings.TrimPrefix(raw, "\ufeff")}
	if p.word() != "WispDB" || !p.take('=') {
		return nil, errors.New("filen är inte en Wisp-journal")
	}
	value, err := p.value(0)
	if err != nil {
		return nil, err
	}
	p.space()
	if p.i != len(p.s) {
		return nil, errors.New("extra Lua content rejected")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("invalid WispDB")
	}
	if fmt.Sprint(object["schema"]) != "1" {
		return nil, errors.New("unsupported WispDB schema")
	}
	installation, _ := object["installation_id"].(string)
	characters, ok := object["characters"].(map[string]any)
	if !ok || len(characters) > 100 || installation == demoInstallation {
		return nil, errors.New("invalid journal characters")
	}
	packets := []Export{}
	for key, v := range characters {
		c, ok := v.(map[string]any)
		if !ok || c["id"] != key {
			return nil, errors.New("invalid character identity")
		}
		if empty, ok := c["events"].(map[string]any); ok && len(empty) == 0 {
			continue
		}
		events, ok := c["events"].([]any)
		if !ok || len(events) > 2000 {
			return nil, errors.New("invalid events")
		}
		name, _ := c["name"].(string)
		realm, _ := c["realm"].(string)
		for start := 0; start < len(events); start += 100 {
			end := start + 100
			if end > len(events) {
				end = len(events)
			}
			b, err := json.Marshal(map[string]any{"schema": 1, "installation_id": installation, "character": Character{ID: key, Name: name, Realm: realm}, "events": events[start:end]})
			if err != nil {
				return nil, err
			}
			pack, err := ParseExport(string(b))
			if err != nil {
				return nil, err
			}
			packets = append(packets, pack)
		}
	}
	return packets, nil
}
