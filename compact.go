package edn

import (
	"bytes"
	"io"
)

func tokNeedsDelim(t tokenType) bool {
	switch t {
	case tokenString, tokenListStart, tokenListEnd, tokenVectorStart,
		tokenVectorEnd, tokenMapEnd, tokenMapStart, tokenSetStart, tokenDiscard, tokenError:
		return false
	}
	return true
}

func delimits(r rune) bool {
	switch r {
	case '{', '}', '[', ']', '(', ')', '\\', '"':
		return true
	}
	return isWhitespace(r)
}

func Compact(dst *bytes.Buffer, src []byte) error {
	origLen := dst.Len()
	var lex lexer
	lex.reset()
	buf := bytes.NewBuffer(src)
	start, pos := 0, 0
	needsDelim := false
	prevIgnore := '\uFFFD'
	r, size, err := buf.ReadRune()
	for ; err == nil; r, size, err = buf.ReadRune() {
		ls := lex.state(r)
		ppos := pos
		pos += size
		switch ls {
		case lexCont:
			if ppos == start && needsDelim && !delimits(r) {
				dst.WriteRune(prevIgnore)
			}
			continue
		case lexIgnore:
			prevIgnore = r
			start = pos
		case lexError:
			dst.Truncate(origLen)
			return lex.err
		case lexEnd:
			// here we might want to discard #_ and the like. Currently we don't.
			dst.Write(src[start:pos])
			needsDelim = tokNeedsDelim(lex.token)
			lex.reset()
			start = pos
		case lexEndPrev:
			dst.Write(src[start:ppos])
			needsDelim = tokNeedsDelim(lex.token)
			lex.reset()
			lss := lex.state(r)
			switch lss {
			case lexIgnore:
				prevIgnore = r
				start = pos
			case lexCont:
				start = ppos
			case lexEnd:
				dst.WriteRune(r)
				lex.reset()
				start = pos
			case lexEndPrev:
				dst.Truncate(origLen)
				return errInternal
			case lexError:
				dst.Truncate(origLen)
				return lex.err
			}
		}
	}
	if err != io.EOF {
		return err
	}
	ls := lex.eof()
	switch ls {
	case lexEnd:
		dst.Write(src[start:pos])
	case lexError:
		dst.Truncate(origLen)
		return lex.err
	}
	return nil
}