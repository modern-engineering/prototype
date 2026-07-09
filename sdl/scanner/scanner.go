// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package scanner

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/modern-engineering/prototype/sdl/token"
)

// An ErrorHandler may be provided to [Scanner.Init]. If a syntax error is
// encountered and a handler was installed, the handler is called with a
// position and an error message. The position points to the beginning of
// the offending token.
type ErrorHandler func(pos token.Position, msg string)

// A Scanner holds the scanner's internal state while processing a given
// source text. It must be initialized via [Scanner.Init] before use.
type Scanner struct {
	// immutable state
	file *token.File
	src  []byte
	err  ErrorHandler

	// scanning state
	ch            rune // current character; eof at end of source
	offset        int  // character offset
	rdOffset      int  // reading offset (position after current character)
	insertNewline bool // emit a NEWLINE token before the next line break

	// ErrorCount is the number of errors encountered so far.
	ErrorCount int
}

const (
	bom = 0xFEFF // byte order mark, permitted only as the first character
	eof = -1     // end of file
)

// next reads the next Unicode character into s.ch. s.ch < 0 means
// end-of-file.
func (s *Scanner) next() {
	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset
		if s.ch == '\n' {
			s.file.AddLine(s.offset)
		}
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= utf8.RuneSelf:
			// not ASCII
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark")
			}
		}
		s.rdOffset += w
		s.ch = r
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.file.AddLine(s.offset)
		}
		s.ch = eof
	}
}

// Init prepares the scanner s to tokenize the source text src from the
// file named filename by setting the scanner at the beginning of src.
// Calls to [Scanner.Scan] invoke the error handler err, if not nil, for
// each syntax error encountered.
//
// Note that Init may call err if there is an error in the first character
// of the file.
func (s *Scanner) Init(filename string, src []byte, err ErrorHandler) {
	s.file = token.NewFile(filename, len(src))
	s.src = src
	s.err = err

	s.ch = ' '
	s.offset = 0
	s.rdOffset = 0
	s.insertNewline = false
	s.ErrorCount = 0

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at file beginning
	}
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil {
		s.err(s.file.Position(offs), msg)
	}
	s.ErrorCount++
}

func (s *Scanner) errorf(offs int, format string, args ...any) {
	s.error(offs, fmt.Sprintf(format, args...))
}

func lower(ch rune) rune     { return ('a' - 'A') | ch } // returns lower-case ch iff ch is ASCII letter
func isDecimal(ch rune) bool { return '0' <= ch && ch <= '9' }

func isLetter(ch rune) bool {
	return 'a' <= lower(ch) && lower(ch) <= 'z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return isDecimal(ch) || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

// scanIdentifier reads the string of valid identifier characters at
// s.offset. It must only be called when s.ch is a valid letter.
func (s *Scanner) scanIdentifier() string {
	offs := s.offset
	for isLetter(s.ch) || isDigit(s.ch) {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

// scanNumber scans an integer or duration lexeme whose digits start at
// the current character; offs is the token start (it precedes s.offset
// when a sign was consumed). A number followed immediately by a letter or
// '.' continues greedily as a DURATION lexeme; the parser validates it
// with time.ParseDuration.
func (s *Scanner) scanNumber(offs int) (token.Token, string) {
	tok := token.INT
	for isDecimal(s.ch) {
		s.next()
	}
	if s.ch == '.' || isLetter(s.ch) || isDigit(s.ch) {
		tok = token.DURATION
		for s.ch == '.' || isLetter(s.ch) || isDigit(s.ch) {
			s.next()
		}
	}
	return tok, string(s.src[offs:s.offset])
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= lower(ch) && lower(ch) <= 'f':
		return int(lower(ch) - 'a' + 10)
	}
	return 16 // larger than any legal digit val
}

// scanEscape parses an escape sequence where rune is the accepted escaped
// quote. In case of a syntax error, it stops at the offending character
// (without consuming it) and returns false. Otherwise it returns true.
func (s *Scanner) scanEscape(quote rune) bool {
	offs := s.offset

	var n int
	var base, max uint32
	switch s.ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', quote:
		s.next()
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.next()
		n, base, max = 2, 16, 255
	case 'u':
		s.next()
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.next()
		n, base, max = 8, 16, unicode.MaxRune
	default:
		msg := "unknown escape sequence"
		if s.ch < 0 {
			msg = "escape sequence not terminated"
		}
		s.error(offs, msg)
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(digitVal(s.ch))
		if d >= base {
			msg := fmt.Sprintf("illegal character %#U in escape sequence", s.ch)
			if s.ch < 0 {
				msg = "escape sequence not terminated"
			}
			s.error(s.offset, msg)
			return false
		}
		x = x*base + d
		s.next()
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		s.error(offs, "escape sequence is invalid Unicode code point")
		return false
	}
	return true
}

// scanString scans an interpreted string literal; offs is the offset of
// the opening quote, which has already been consumed. The returned lexeme
// includes the quotes.
func (s *Scanner) scanString(offs int) string {
	for {
		ch := s.ch
		if ch == '\n' || ch < 0 {
			s.error(offs, "string literal not terminated")
			break
		}
		s.next()
		if ch == '"' {
			break
		}
		if ch == '\\' {
			s.scanEscape('"')
		}
	}
	return string(s.src[offs:s.offset])
}

// scanComment scans a // line comment; offs is the offset of the first
// '/', which has already been consumed. The returned text includes the
// leading "//" and excludes the terminating line break.
func (s *Scanner) scanComment(offs int) string {
	s.next() // consume the second '/'
	for s.ch != '\n' && s.ch >= 0 {
		s.next()
	}
	lit := s.src[offs:s.offset]
	if len(lit) > 0 && lit[len(lit)-1] == '\r' {
		lit = lit[:len(lit)-1]
	}
	return string(lit)
}

// Scan scans the next token and returns the token position, the token,
// and its literal string if applicable. The source end is indicated by
// [token.EOF].
//
// If the returned token is a literal (an identifier, a string, integer,
// or duration), a keyword, or a [token.COMMENT], the literal string has
// the corresponding value. If the returned token is [token.NEWLINE], the
// literal string is "\n" (also for the NEWLINE synthesized at EOF).
// Otherwise the literal string is empty.
//
// If a syntax error was encountered, Scan reports it through the error
// handler installed with Init, returns a [token.ILLEGAL] token where no
// better classification exists, and continues, so a full (degraded) token
// stream is always available.
func (s *Scanner) Scan() (pos token.Position, tok token.Token, lit string) {
	// Skip white space, deciding statement termination at line breaks.
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\r' || s.ch == '\n' {
		if s.ch == '\n' && s.insertNewline {
			s.insertNewline = false
			pos = s.file.Position(s.offset)
			s.next()
			return pos, token.NEWLINE, "\n"
		}
		s.next()
	}

	pos = s.file.Position(s.offset)

	// insertNewline records whether a line break after this token
	// terminates a statement (see the package documentation).
	insertNewline := false

	switch ch := s.ch; {
	case isLetter(ch):
		lit = s.scanIdentifier()
		tok = token.Lookup(lit)
		insertNewline = tok == token.IDENT || tok == token.TRUE || tok == token.FALSE
	case isDecimal(ch):
		tok, lit = s.scanNumber(s.offset)
		insertNewline = true
	default:
		s.next() // always make progress
		switch ch {
		case eof:
			if s.insertNewline {
				s.insertNewline = false
				return pos, token.NEWLINE, "\n"
			}
			return pos, token.EOF, ""
		case '"':
			tok = token.STRING
			lit = s.scanString(pos.Offset)
			insertNewline = true
		case '/':
			if s.ch == '/' {
				tok = token.COMMENT
				lit = s.scanComment(pos.Offset)
				insertNewline = s.insertNewline // comments are transparent
				break
			}
			s.errorf(pos.Offset, "illegal character %#U", ch)
			insertNewline = s.insertNewline // preserve pending NEWLINE
			tok, lit = token.ILLEGAL, string(ch)
		case '+', '-':
			if isDecimal(s.ch) {
				tok, lit = s.scanNumber(pos.Offset)
				insertNewline = true
				break
			}
			s.errorf(pos.Offset, "illegal character %#U", ch)
			insertNewline = s.insertNewline // preserve pending NEWLINE
			tok, lit = token.ILLEGAL, string(ch)
		case '{':
			tok = token.LBRACE
		case '}':
			tok = token.RBRACE
			insertNewline = true
		case '(':
			tok = token.LPAREN
		case ')':
			tok = token.RPAREN
			insertNewline = true
		case ':':
			tok = token.COLON
		case '.':
			tok = token.PERIOD
		default:
			s.errorf(pos.Offset, "illegal character %#U", ch)
			insertNewline = s.insertNewline // preserve pending NEWLINE
			tok, lit = token.ILLEGAL, string(ch)
		}
	}

	s.insertNewline = insertNewline
	return pos, tok, lit
}
