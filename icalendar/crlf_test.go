package icalendar

import (
	"bytes"
	"io"
	"testing"
)

func Test_nextLine(t *testing.T) {
	// TODO strip this of error-reporting test logic, as nextLine no longer
	// has any of its own errors to report (modulo EOF).
	type x []interface{}
	type example struct {
		input  string
		result x
	}
	testCases := []example{
		{"", x{}},
		{"a\r\nb\r\n c\r\n", x{"a\r\n", "b\r\n c\r\n"}},
		{"a\r\n b\r\nc\r\n", x{"a\r\n b\r\n", "c\r\n"}},
		{"a\r\nb\r\n c", x{"a\r\n", "b\r\n c\r\n"}},
	}
	for _, testCase := range testCases {
		buf := bytes.NewBufferString(testCase.input)
		iter := newfieldIter(buf)
		for _, result := range testCase.result {
			line, err := nextLineAux(&iter)
			if err == io.EOF {
				t.Errorf("\npremature EOF in case %#v\n", testCase.input)
				break
			}
			switch expectedLine := result.(type) {
			case string:
				if err != nil {
					t.Errorf("\nunexpected error in case %#v:\n%s\n",
						testCase.input, err)
					break
				} else if string(line) != expectedLine {
					t.Errorf("\nmismatch in case %#v:\nexpected: %#v\ngot:     %#v\n",
						testCase.input, expectedLine, string(line))
					break
				}
			default:
				expectedErr := expectedLine.(error)
				if err != expectedErr {
					t.Errorf("\nerror mismatch in case %#v:\nexpected: %s\ngot:     %s\n",
						testCase.input, expectedErr, err)
					break
				}
			}
		}
		if line, err := nextLineAux(&iter); err != io.EOF {
			if err != nil {
				t.Errorf("\nin case %#v:\nexpected EOF\ngot: %#v\n",
					testCase.input, string(line))
			} else {
				t.Errorf("\nin case %#v:\nexpected EOF\ngot error: '%s'\n",
					testCase.input, err)
			}
		}
	}
}

func nextLineAux(iter *fieldIter) (line []byte, err error) {
	if err = iter.src.Err(); err != nil {
		return
	}
	if iter.eof {
		err = io.EOF
		return
	}
	line, err = iter.nextLine()
	// implicitly report error
	return
}

func Test_nextField(t *testing.T) {
	type x []interface{}
	type example struct {
		input  string
		result x
	}
	testCases := []example{
		{"", x{}},
		{"DESCRIPTION:This\r\n  is a long\r\n", x{
			Field{Name: "DESCRIPTION", Value: "This is a long"},
		}},
	}
	for _, testCase := range testCases {
		buf := bytes.NewBufferString(testCase.input)
		iter := newfieldIter(buf)
		for _, result := range testCase.result {
			field, err := (&iter).nextField()
			if err == endOfFields {
				t.Errorf("\npremature endOfFields in case %#v\n", testCase.input)
				break
			}
			switch expectedField := result.(type) {
			case Field:
				if err != nil {
					t.Errorf("\nunexpected error in case %#v:\n%s\n",
						testCase.input, err)
					break
				} else if !fieldEq(field, expectedField) {
					t.Errorf("\nmismatch in case %#v:\nexpected: %#v\ngot:     %#v\n",
						testCase.input, expectedField, field)
					break
				}
			default:
				expectedErr := expectedField.(error)
				if err != expectedErr {
					t.Errorf("\nerror mismatch in case %#v:\nexpected: %s\ngot:     %s\n",
						testCase.input, expectedErr, err)
					break
				}
			}
		}
		if field, err := (&iter).nextField(); err != io.EOF {
			if err != nil {
				t.Errorf("\nin case %#v:\nexpected EOF\ngot: %#v\n",
					testCase.input, field)
			} else {
				t.Errorf("\nin case %#v:\nexpected EOF\ngot error: '%s'\n",
					testCase.input, err)
			}
		}
	}
}
