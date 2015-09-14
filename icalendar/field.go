package icalendar

import (
	"errors"
	"time"
)

type Field struct {
	Name   string
	Params map[string][]string
	Value  string
}

var (
	noName              = errors.New("Field has no name")
	noValue             = errors.New("Field has no value")
	invalidCharInName   = errors.New("Invalid character in name (must be alphanumeric or '-')")
	endOfParams         = errors.New("End of parameter list")
	emptyParamName      = errors.New("Empty parameter name")
	unexpectedEOI       = errors.New("Unexpected end of input while reading parameter")
	invalidParam        = errors.New("Invalid parameter")
	illegalCharInParam  = errors.New("Illegal character in parameter value")
	invalidQuoted       = errors.New("Invalid quoted value in parameter")
	illegalCharInQuoted = errors.New("Illegal character in quoted parameter value")
)

// See RFC 2445 s. 4.1. We use bytes instead of runes here because the RFC is
// written for ASCII. It will never-the-less handle UTF-8 text where legal just
// fine, save for the edge case where an incomplete multi-codepoint grapheme
// directly precedes a control character like the double quote. I see no way to
// fully support zalgo text in field values without violating the spec.
func readField(str []byte) (field Field, err error) {
	// Read in the field name.
	field.Name, err = readName(&str)
	if err != nil {
		return
	}
	if field.Name == "" {
		err = noName
		return
	}
	// Read in the parameter list
	field.Params = make(map[string][]string)
	for len(str) > 0 && str[0] != ':' {
		var key string
		var val []string
		key, val, err = readParam(&str)
		if err == endOfParams {
			break
		}
		if err != nil {
			return
		}
		field.Params[key] = val
	}
	if len(str) == 0 || str[0] != ':' {
		err = noValue
		return
	}
	str = str[1:]
	// Read in the value
	// TODO validate the value
	field.Value = string(str)
	// The error is implicitly reported if present
	return
}

func readName(str *[]byte) (name string, err error) {
	for i := 0; i < len(*str); i++ {
		switch c := (*str)[i]; {
		case c == ':', c == ';', c == '=':
			name = string((*str)[:i])
			*str = (*str)[i:]
			return
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '1' && c <= '9',
			c == '-':
			continue
		default:
			err = invalidCharInName
			return
		}
	}
	name = string(*str)
	*str = (*str)[len(*str):]
	return
}

func readParam(str *[]byte) (key string, vals []string, err error) {
	if (*str)[0] != ';' {
		err = endOfParams
		return
	}
	*str = (*str)[1:]
	// Read the key
	key, err = readName(str)
	if err != nil {
		return
	}
	if key == "" {
		err = emptyParamName
		return
	}
	if len(*str) == 0 {
		err = unexpectedEOI
		return
	}
	if (*str)[0] != '=' {
		err = invalidParam
		return
	}
	*str = (*str)[1:]
	// Read the value(s)
	for len(*str) > 0 {
		// quoted value
		if (*str)[0] == '"' {
			var val string
			val, err = readQuoted(str)
			if err != nil {
				return
			}
			vals = append(vals, val)
		} else {
			// unquoted value
			i := 0
		LOOP:
			for i < len(*str) {
				switch c := (*str)[i]; {
				case c == ',', c == ';', c == ':':
					break LOOP
				case c == '\t', c == '\n', c == '\v', c == '\r':
					break // out of the switch
				case c == '\x7f', c == '"', c < ' ':
					err = illegalCharInParam
					return
				default:
				}
				i++
			}
			vals = append(vals, string((*str)[:i]))
			*str = (*str)[i:]
		}
		if len(*str) > 0 && (*str)[0] == ',' {
			*str = (*str)[1:]
		} else {
			break
		}
	}
	return
}

func readQuoted(str *[]byte) (val string, err error) {
	if len(*str) < 2 || (*str)[0] != '"' {
		err = invalidQuoted
		return
	}
	*str = (*str)[1:]
	i := 0
	for {
		if i >= len(*str) {
			err = invalidQuoted
			return
		}
		if (*str)[i] == '"' {
			break
		}
		switch c := (*str)[i]; {
		case c == '\t', c == '\n', c == '\v', c == '\r':
			break // out of the switch
		case c == '\x7f', c < ' ':
			err = illegalCharInQuoted
			return
		default:
		}
		i++
	}
	val = string((*str)[:i])
	*str = (*str)[i+1:] // +1 because skip the quote
	return
}
