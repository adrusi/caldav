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

func Test_validate(t *testing.T) {
	testCases := map[string]error{
		"DESCRIPTION;ALTREP=\"CID:part3.msg.970415T083000@example.com\":" +
			"Project XYZ Review Meeting will include the following agenda items:" +
			"(a) Market Overview\\, (b) Finances\\, (c) Project Management": nil,
		"DESCRIPTION;ALTREP=\"http://example.com\",\"http://example.org\":foo": expectedScalar,
		"ORGANIZER;CN=\"John Smith\":mailto:jsmith@example.com":                nil,
		"ORGANIZER;CN=\"John\",\"Smith\":mailto:jsmith@example.com":            expectedScalar,
		"ATTENDEE;CUTYPE=GROUP:mailto:ietf-calsch@example.org":                 nil,
		"ATTENDEE;CUTYPE=GROUP,UNKNOWN:mailtp:ietf-calsch@example.org":         expectedScalar,
		"ATTENDEE;CUTYPE=\"@#$\":mailtp:ietf-calsch@example.org":               invalidToken,
		"ATTENDEE;CUTYPE=X-USERTYPE:mailtp:ietf-calsch@example.org":            nil,
		"ATTENDEE;CUTYPE=TYPE-IANA-REGISTERED:mailtp:ietf-calsch@example.org":  nil,
		"ATTACH;FMTTYPE=text/plain;ENCODING=BASE64;VALUE=BINARY:TG9yZW" +
			"0gaXBzdW0gZG9sb3Igc2l0IGFtZXQsIGNvbnNlY3RldHVyIGFkaXBpc2ljaW" +
			"5nIGVsaXQsIHNlZCBkbyBlaXVzbW9kIHRlbXBvciBpbmNpZGlkdW50IHV0IG" +
			"xhYm9yZSBldCBkb2xvcmUgbWFnbmEgYWxpcXVhLiBVdCBlbmltIGFkIG1pbm" +
			"ltIHZlbmlhbSwgcXVpcyBub3N0cnVkIGV4ZXJjaXRhdGlvbiB1bGxhbWNvIG" +
			"xhYm9yaXMgbmlzaSB1dCBhbGlxdWlwIGV4IGVhIGNvbW1vZG8gY29uc2VxdW" +
			"F0LiBEdWlzIGF1dGUgaXJ1cmUgZG9sb3IgaW4gcmVwcmVoZW5kZXJpdCBpbi" +
			"B2b2x1cHRhdGUgdmVsaXQgZXNzZSBjaWxsdW0gZG9sb3JlIGV1IGZ1Z2lhdC" +
			"BudWxsYSBwYXJpYXR1ci4gRXhjZXB0ZXVyIHNpbnQgb2NjYWVjYXQgY3VwaW" +
			"RhdGF0IG5vbiBwcm9pZGVudCwgc3VudCBpbiBjdWxwYSBxdWkgb2ZmaWNpYS" +
			"BkZXNlcnVudCBtb2xsaXQgYW5pbSBpZCBlc3QgbGFib3J1bS4=": nil,
		"ATTACH;FMTTYPE=text/plain;ENCODING=BASE64,8BIT;VALUE=BINARY:TG9yZW":               expectedScalar,
		"ATTACH;FMTTYPE=text/plain;ENCODING=8BIT;VALUE=BINARY:TG9yZW":                      invalidEncoding,
		"ATTACH;FMTTYPE=text/plain;VALUE=BINARY:TG9yZW":                                    invalidEncoding,
		"ATTACH;FMTTYPE=text/plain;ENCODING=BASE2:TG9yZW":                                  invalidOption,
		"ATTACH;FMTTYPE=application/msword:ftp://example.com/pub/docs/agenda.do":           nil,
		"ATTACH;FMTTYPE=application/msword,text/html:ftp://example.com/pub/docs/agenda.do": expectedScalar,
		"ATTACH;FMTTYPE=jpg:ftp://example.com/pub/docs/agenda.do":                          invalidMime,
		"FREEBUSY;FBTYPE=BUSY:19980415T133000Z/19980415T170000Z":                           nil,
		"FREEBUSY;FBTYPE=BUSY,FREE:19980415T133000Z/19980415T170000Z":                      expectedScalar,
		"FREEBUSY;FBTYPE=\"$$$\":19980415T133000Z/19980415T170000Z":                        invalidToken,
		"FREEBUSY;FBTYPE=X-DEAD:19980415T133000Z/19980415T170000Z":                         nil,
		"FREEBUSY;FBTYPE=SOME-IANA-STATUS:19980415T133000Z/19980415T170000Z":               nil,
		"SUMMARY;LANGUAGE=en-US:Company Holiday Party":                                     nil,
		"LOCATION;LANGUAGE=en:Germany":                                                     nil,
		"LOCATION;LANGUAGE=no:Tyskland":                                                    nil,
		"LOCATION;LANGUAGE=no,en-US:Tyskland":                                              expectedScalar,
		"ATTENDEE;MEMBER=\"mailto:ietf-calsch@example.org\":mailto:jsmith@example.com":     nil,
		"ATTENDEE;MEMBER=\"mailto:projectA@example.com\"," +
			"\"mailto:projectB@example.com\":mailto:janedoe@example.com": nil,
		"ATTENDEE;PARTSTAT=DECLINED:mailto:jsmith@example.com":                    nil,
		"ATTENDEE;PARTSTAT=DECLINED,ACCEPTED:mailto:jsmith@example.com":           expectedScalar,
		"ATTENDEE;PARTSTAT=\"###\":mailto:jsmith@example.com":                     invalidToken,
		"ATTENDEE;PARTSTAT=X-PROBABLY-NOT:mailto:jsmith@example.com":              nil,
		"ATTENDEE;PARTSTAT=SOME-IANA-STATUS:mailto:jsmith@example.com":            nil,
		"RECURRENCE-ID;RANGE=THISANDFUTURE:19980401T133000Z":                      nil,
		"RECURRENCE-ID;RANGE=ONLYTHIS:19980401T133000Z":                           invalidOption,
		"RECURRENCE-ID;RANGE=THISANDFUTURE,THISANDFUTURE:19980401T133000Z":        expectedScalar,
		"TRIGGER;RELATED=END:PT5M":                                                nil,
		"TRIGGER;RELATED=START:PT5M":                                              nil,
		"TRIGGER;RELATED=MIDDLE:PT5M":                                             invalidOption,
		"TRIGGER;RELATED=START,END:PT5M":                                          expectedScalar,
		"RELATED-TO;RELTYPE=SIBLING:19960401-080045-4000F192713@example.com":      nil,
		"ATTENDEE;ROLE=CHAIR:mailto:mrbig@example.com":                            nil,
		"ATTENDEE;ROLE=\"***\":mailto:mrbig@example.com":                          invalidToken,
		"ATTENDEE;RSVP=TRUE:mailto:jsmith@example.com":                            nil,
		"ATTENDEE;RSVP=MAYBE:mailto:jsmith@example.com":                           invalidOption,
		"ORGANIZER;SENT-BY=\"mailto:sray@example.com\":mailto:jsmith@example.com": nil,
		"ORGANIZER;SENT-BY=\"mailto:sray@example.com\"," +
			"\"mailto:adrian@adrusi.com\":mailto:jsmith@example.com": expectedScalar,
		"DTSTART;TZID=America/New_York:19980119T020000":                nil,
		"DTEND;TZID=America/New_York:19980119T030000":                  nil,
		"DTEND;TZID=America/New_York,Europe/Amsterdam:19980119T030000": expectedScalar,
	}
	for testCase, expectedErr := range testCases {
		field, err := readField([]byte(testCase))
		if err != nil {
			t.Errorf("\nparsing error in case:\n%s\nthe error was: %s\n",
				testCase, err)
			continue
		}
		if err = field.validate(); err != expectedErr {
			t.Errorf("\nerror in case:\n%s\nexpected: %s\ngot:     %s\n",
				testCase, expectedErr, err)
		}
	}
}
