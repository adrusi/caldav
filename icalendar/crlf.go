package icalendar

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

type fieldIter struct {
	src *bufio.Scanner
	eof bool
}

func newfieldIter(src io.Reader) (iter fieldIter) {
	iter.src = bufio.NewScanner(src)
	iter.src.Split(bufio.ScanLines)
	if !iter.src.Scan() && iter.src.Err() == nil && len(iter.src.Bytes()) == 0 {
		// iter.src.Err() will be handled later.
		// If there are some bytes remaining we don't want to call it EOF just
		// yet, as those need to be processed. When Scan() is called again it
		// will return empty data and EOF again.
		iter.eof = true
	}
	return
}

var (
	endOfFields = errors.New("No more fields to report")
	crlfError   = errors.New("CR not followed by LF")
)

func (iter *fieldIter) nextField() (field Field, err error) {
	if err = iter.src.Err(); err != nil {
		return
	}
	if iter.eof {
		err = io.EOF
		return
	}
	var line []byte
	line, err = iter.nextLine()
	switch err {
	case io.EOF:
		err = endOfFields
		return
	case nil:
		break
	default:
		return
	}
	line, err = verifyAndUnfold(line)
	if err != nil {
		return
	}
	field, err = readField(line)
	// implicitly report error
	return
}

func (iter *fieldIter) nextLine() (line []byte, err error) {
	// Inconveniently for this use-case, bufio.ScanLines omits the linebreaks
	// from the returned data.
	// TODO is there a potential vulnerability in treating a bare LF the same
	// as a CRLF? This is how bufio.ScanLines works, but it shouldn't be too
	// hard to fix it using a custom ScanFunc.
	line = append(line, iter.src.Bytes()...)
	line = append(line, '\r', '\n')
	for {
		if !iter.src.Scan() {
			if err = iter.src.Err(); err != nil {
				return
			}
			if len(iter.src.Bytes()) == 0 {
				iter.eof = true
				return
			}
		}
		// len(src.Bytes()) will always be > 0 here because it will always end
		// with '\n' unless EOF is reached, which has already been handled.
		if iter.src.Bytes()[0] != ' ' {
			return
		}
		line = append(line, iter.src.Bytes()...)
		line = append(line, '\r', '\n')
	}
}

func verifyAndUnfold(line []byte) ([]byte, error) {
	line = bytes.Replace(line, []byte{'\r', '\n', ' '}, []byte{}, -1)
	line = bytes.TrimSuffix(line, []byte{'\r', '\n'})
	for _, c := range []byte{'\r', '\n'} {
		if bytes.IndexByte(line, c) != -1 {
			return nil, crlfError
		}
	}
	return line, nil
}
