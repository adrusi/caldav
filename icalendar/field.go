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

// Returns the empty string when absent or invalid
func (f Field) AltRep() string {
	if val, has := f.Params["ALTREP"]; has && len(val) == 1 {
		return val[0]
	}
	return ""
}

// Returns the empty string when absent or invalid
func (f Field) CommonName() string {
	if val, has := f.Params["CN"]; has && len(val) == 1 {
		return val[0]
	}
	return ""
}

type UserType string

const (
	UTIndividual UserType = "INDIVIDUAL"
	UTGroup               = "GROUP"
	UTResource            = "RESOURCE"
	UTRoom                = "ROOM"
	UTUnknown             = "UNKNOWN"
	// X-* usertypes also allowed
)

func (f Field) UserType() UserType {
	if val, has := f.Params["CUTYPE"]; has && len(val) == 1 {
		return UserType(val[0])
	}
	return UTIndividual
}

func (f Field) Delegators() []string {
	if val, has := f.Params["DELEGATED-FROM"]; has {
		return val
	}
	return make([]string, 0, 0)
}

func (f Field) Delegatees() []string {
	if val, has := f.Params["DELEGATED-TO"]; has {
		return val
	}
	return make([]string, 0, 0)
}

// Returns the empty string when absent or invalid
func (f Field) DirEntryRef() string {
	if val, has := f.Params["DIR"]; has && len(val) == 1 {
		return val[0]
	}
	return ""
}

func (f Field) Encoding() string {
	if val, has := f.Params["ENCODING"]; has && len(val) == 1 {
		return val[0]
	}
	return "8BIT"
}

// Returns application/octet-stream if no fmtype is specified. It may be useful
// to ignore this default if you have a filetype detector.
func (f Field) FormatType() string {
	if val, has := f.Params["FMTTYPE"]; has && len(val) == 1 {
		return val[0]
	}
	return "application/octet-steam" // least specific MIME type
}

type FreeBusyType string

const (
	FBFree            FreeBusyType = "FREE"
	FBBusy                         = "BUSY"
	FBBusyUnavailable              = "BUSY-UNAVAILABLE"
	FBBusyTentative                = "BUSY-TENTATIVE"
)

func (f Field) FreeBusyType() FreeBusyType {
	if val, has := f.Params["FBTYPE"]; has && len(val) == 1 {
		return FreeBusyType(val[0])
	}
	return FBBusy
}

func (f Field) Language() string {
	// TODO there's probably some type better than string to represent language
	// tags.
	if val, has := f.Params["LANGUAGE"]; has && len(val) == 1 {
		return val[0]
	}
	return "x-Unknown" // Is there a standard tag for unknown language?
	// See RFC5646
}

func (f Field) Members() []string {
	if val, has := f.Params["MEMBER"]; has {
		return val
	}
	return make([]string, 0, 0)
}

type ParticipantStatus string

const (
	PSAccepted    ParticipantStatus = "ACCEPTED"
	PSDeclined                      = "DECLINED"
	PSTentative                     = "TENTATIVE"
	PSDelegated                     = "DELEGATED"
	PSCompleted                     = "COMPLETED"
	PSInProcess                     = "IN-PROCESS"
	PSNeedsAction                   = "NEEDS-ACTION"
)

func (f Field) ParticipantStatus() ParticipantStatus {
	if val, has := f.Params["PARTSTAT"]; has && len(val) == 1 {
		return ParticipantStatus(val[0])
	}
	return PSNeedsAction
}

func (f Field) ThisAndFuture() bool {
	val, has := f.Params["RANGE"]
	return has && len(val) == 1 && val[0] == "THISANDFUTURE"
}

type AlarmTriggerRelationship int

const (
	ATRStart AlarmTriggerRelationship = iota
	ATREnd
)

func (f Field) AlarmTrigerRelationship() AlarmTriggerRelationship {
	if val, has := f.Params["RELATED"]; has && len(val) == 1 {
		if val[0] == "END" {
			return ATREnd
		}
	}
	return ATRStart
}

type RelationshipType string

const (
	RTParent  RelationshipType = "PARENT"
	RTChild                    = "CHILD"
	RTSibling                  = "SIBLING"
)

func (f Field) RelationshipType() RelationshipType {
	if val, has := f.Params["RELTYPE"]; has && len(val) == 1 {
		return RelationshipType(val[0])
	}
	return RTParent
}

type ParticipantRole string

const (
	PRChair          ParticipantRole = "CHAIR"
	PRReqParticipant                 = "REQ-PARTICIPANT"
	PROptParticipant                 = "OPT-PARTICIPANT"
	PRNonParticipant                 = "NON-PARTICIPANT"
)

func (f Field) ParticipantRole() ParticipantRole {
	if val, has := f.Params["ROLE"]; has && len(val) == 1 {
		return ParticipantRole(val[0])
	}
	return PRReqParticipant
}

func (f Field) Rsvp() bool {
	if val, has := f.Params["RSVP"]; has && len(val) == 1 && val[0] == "TRUE" {
		return true
	}
	return false
}

func (f Field) SentBy() string {
	if val, has := f.Params["SENT-BY"]; has && len(val) == 1 {
		return val[0]
	}
	return ""
}

func (f Field) TimeZone() *time.Location {
	if val, has := f.Params["TZID"]; has && len(val) == 1 {
		loc, err := time.LoadLocation(val[0])
		if err == nil {
			return loc
		}
	}
	loc, _ := time.LoadLocation("UTC")
	return loc
}

type DataType string

const (
	DTBinary     DataType = "BINARY"
	DTBoolean             = "BOOLEAN"
	DTCalAddress          = "CAL-ADDRESS"
	DTDate                = "DATE"
	DTDateTime            = "DATE-TIME"
	DTDuration            = "DURATION"
	DTFloat               = "FLOAT"
	DTInteger             = "INTEGER"
	DTPeriod              = "PERIOD"
	DTRecur               = "RECUR"
	DTText                = "TEXT"
	DTUri                 = "URI"
	DTUtcOffset           = "UTC-OFFSET"
)

func (f Field) DataType() DataType {
	if val, has := f.Params["VALUE"]; has && len(val) == 1 {
		return DataType(val[0])
	}
	// TODO the default value should depend on the fieldname
	return DTText
}
