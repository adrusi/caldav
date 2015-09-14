package icalendar

import (
	"testing"
)

func fieldEq(a, b Field) bool {
	if a.Name != b.Name || a.Value != b.Value {
		return false
	}
	if len(a.Params) != len(b.Params) {
		return false
	}
	for k := range a.Params {
		if len(b.Params[k]) != len(a.Params[k]) {
			return false
		}
		for i := range a.Params[k] {
			if b.Params[k][i] != a.Params[k][i] {
				return false
			}
		}
	}
	return true
}

func Test_readField(t *testing.T) {
	cases := map[string]interface{}{
		// Some correct examples from the RFC
		"ATTENDEE;RSVP=TRUE;ROLE=REQ-PARTICIPANT:MAILTO:jsmith@host.com": Field{
			"ATTENDEE",
			map[string][]string{
				"RSVP": []string{"TRUE"},
				"ROLE": []string{"REQ-PARTICIPANT"},
			},
			"MAILTO:jsmith@host.com",
		},
		"RDATE;VALUE=DATE:19970304,19970504,19970704,19970904": Field{
			"RDATE",
			map[string][]string{"VALUE": []string{"DATE"}},
			"19970304,19970504,19970704,19970904",
		},
		"DESCRIPTION;ALTREP=\"http://www.wiz.org\":The Fall'98 ...": Field{
			"DESCRIPTION",
			map[string][]string{"ALTREP": []string{"http://www.wiz.org"}},
			"The Fall'98 ...",
		},
		"ATTENDEE;DELEGATED-TO=\"mailto:jdoe@example.com\"," +
			"\"mailto:jqpublic@example.com\":mailto:jsmith@example.com": Field{
			"ATTENDEE",
			map[string][]string{"DELEGATED-TO": []string{
				"mailto:jdoe@example.com",
				"mailto:jqpublic@example.com",
			}},
			"mailto:jsmith@example.com",
		},
		// Errors
		";VALUE=DATE:19970304":        noName,
		"RDATE;VALUE=DATE":            noValue,
		"RDATE;VALUE=DATE:":           nil, // ensure empty values are OK
		"R_DATE;VALUE=DATE":           invalidCharInName,
		"RDATE;VALUE":                 unexpectedEOI,
		"RDATE;VALUE=":                noValue, // make sure that empty params are ok
		"RDATE;VALUE:19970304":        invalidParam,
		"RDATE;=DATE:19970304":        emptyParamName,
		"RDATE;VALUE=\b:19980304":     illegalCharInParam,
		"RDATE;VALUE=\"DATE:19970304": invalidQuoted,
	}
	for str, expect := range cases {
		field, err := readField([]byte(str))
		switch expectedField := expect.(type) {
		case Field:
			if err != nil {
				t.Errorf("\nunexpected error in case '%s':\n%s\n", str, err)
			} else if !fieldEq(field, expectedField) {
				t.Errorf("\nfield mismatch in case '%s':\nexpected: %#v\ngot:      %#v\n",
					str, expectedField, field)
			}
		default:
			var expectedErr error
			if expect == nil {
				expectedErr = nil
			} else {
				expectedErr = expect.(error)
			}
			if err != expectedErr {
				t.Errorf("\nerror mismatch in case '%s':\nexpected: %s\ngot:      %s\n",
					str, expectedErr, err)
			}
		}
	}
}
