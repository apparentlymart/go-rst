package parser

import (
	"bufio"
	"io"
	"strings"

	"github.com/apparentlymart/go-rst"
)

type Token struct {
	Type     TokenType
	Data     string
	Position rst.Position
}

type TokenType int

//go:generate stringer -type=TokenType
const (
	INVALID TokenType = iota
	LINE
	BLANK
	LITERAL
	INDENT
	DEDENT

	// LATE_INDENT is a special situation where the indent decreases to
	// a place not in the indent stack, thus indicating that there was
	// an implied intermediate level, such as when a block quote itself
	// begins with a nested block quote. In this case the parser must
	// move everything it's parsed so far in the current block context
	// into a new blockquote element before parsing continues.
	LATE_INDENT

	EOF
	ERROR
)

type Scanner struct {
	lineScanner *bufio.Scanner

	filename string
	line     int

	// Keep track of all of the indent levels we've issued INDENT tokens
	// for, so that we can issue symmetrical DEDENT tokens when we
	// see shorter indents.
	indents    []int
	lateIndent bool

	literal    bool
	lazyIndent bool

	peek *Token

	nextIndent int
	nextToken  *Token
}

func NewScanner(r io.Reader, filename string) *Scanner {
	lineScanner := bufio.NewScanner(r)
	lineScanner.Split(splitRSTLines)

	// Our indent stack has one permanent member at column 0, and then
	// grows as necessary. We'll start at capacity 10 so we can parse
	// shallow documents without more allocation.
	indents := make([]int, 1, 10)

	return &Scanner{
		lineScanner: lineScanner,
		filename:    filename,
		line:        1,
		indents:     indents,
		lazyIndent:  false,
		peek:        nil,
	}
}

// Read finds the next token and returns it, advancing the current position
// so that a subsequent Read (or Peek) will return the next token.
//
// Once EOF is reached, the scanner produces an infinite stream of EOF tokens.
//
// If an error occurs, the scanner produces an infinite stream of ERROR tokens.
func (s *Scanner) Read() *Token {
	tok := s.Peek()
	s.peek = nil
	return tok
}

// Peek gets a "sneak preview" of the next token without actually consuming
// it. After peeking a caller must Read() the peeked token before calling
// PushIndent or LazyIndent, or else they will panic.
func (s *Scanner) Peek() *Token {
	if s.peek == nil {
		s.peek = s.next()
	}
	return s.peek
}

// next produces the next token in the token stream, which will either be
// a real token obtained from s.nextToken or it will be a synthetic token
// to adjust the indent level to match s.nextIndent.
func (s *Scanner) next() *Token {
	// Make sure our scanning state is synced and up-to-date
	s.scan()

	if s.lazyIndent {
		s.lazyIndent = false

		// "lazy indent" only applies if the next token is a LINE token
		// which indents more than current.
		if s.nextToken.Type != LINE || s.nextIndent <= s.currentIndent() {
			// Synthetic DEDENT token makes sure we let the parser leave
			// whatever context it was in that was expecting a lazy indent.
			return &Token{
				Type:     DEDENT,
				Data:     "",
				Position: s.nextToken.Position,
			}
		}

		// If we are actually *doing* the lazy indent then no token is
		// emitted for it because the parser is already in whatever
		// context the lazy indent applies to, so we'll just record the
		// new indent to bypass the INDENT token and then emit the
		// LINE token as normal below.
		s.PushIndent(s.nextIndent)
	}

	currentIndent := s.currentIndent()

	switch {
	case s.nextIndent > currentIndent:
		s.indents = append(s.indents, s.nextIndent)

		tokenType := INDENT
		if s.lateIndent {
			tokenType = LATE_INDENT
			s.lateIndent = false
		}

		return &Token{
			Type: tokenType,
			Data: strings.Repeat(" ", s.nextIndent),
			Position: rst.Position{
				Line:     s.nextToken.Position.Line,
				Column:   1,
				Filename: s.nextToken.Position.Filename,
			},
		}
	case s.nextIndent < currentIndent:
		s.indents = s.indents[:len(s.indents)-1]

		// If the *new* current indent is less than what we were shooting
		// for then we've encountered a "late indent" situation which
		// needs special handling so we can let the parser know it needs
		// to adjust what it's been building to account for an extra
		// level of indentation we didn't know about before.
		if s.nextIndent > s.currentIndent() {
			s.lateIndent = true
		}

		return &Token{
			Type:     DEDENT,
			Data:     "",
			Position: s.nextToken.Position,
		}
	default:
		// If we get here then we've already emitted any INDENT and DEDENT
		// tokens we needed to get to the indent level in s.nextIndent and
		// so it's time for us to emit the *real* token.

		token := s.nextToken
		s.nextToken = nil // let scan() know we need another token
		return token
	}
}

// scan ensures that our internal state is synchronized by setting nextToken
// and nextIndent. After scan has run once, it will do nothing until method
// "next" sets nextToken back to nil.
func (s *Scanner) scan() {
	if s.nextToken == nil {
		// Initially position is at the start of the current line.
		// We will move "Column" later if we find an indented line toke,
		position := rst.Position{
			Line:     s.line,
			Column:   1,
			Filename: s.filename,
		}
		if s.lineScanner.Scan() {
			s.line++
			whole := s.lineScanner.Text()
			data := whole
			indent := 0
			for {
				if len(data) == 0 {
					break
				}
				if data[0] == 32 {
					indent++
				} else if data[0] == 9 {
					// Advance indent to the next multiple of 8, since RST
					// is defined as using 8-column tab stops
					indent = indent + (8 - (indent % 8))
				} else {
					break
				}
				data = data[1:]
			}

			if s.literal {
				// This is a continuation of a literal block unless it
				// contains non-whitespace characters that are indented
				// less than the line that introduce the literal block,
				// which (whenever s.literal is true) is our current
				// indent level.
				if len(data) > 0 && indent > s.currentIndent() {
					s.nextIndent = s.currentIndent()
					s.nextToken = &Token{
						Type: LITERAL,

						// For literals it is the parser's responsibility
						// to trim off a suitable amount of leading whitespace
						// once it has collected all of the consecutive
						// LITERAL tokens and can see which one has the
						// shortest prefix, so we'll just give it the whole
						// line to work with.
						Data: whole,

						Position: position,
					}
					return
				}
			}

			if len(data) >= 2 && data[len(data)-2:] == "::" {
				// Marker of the beginning of literal lines.
				s.literal = true

				if len(data) >= 3 {
					before := data[len(data)-3]
					if before != 32 && before != 9 {
						// If the character right before the :: marker
						// is not whitespace then we need to retain one
						// of the two colons.
						data = data[:len(data)-1]
					} else {
						// Otherwise, eat both colons
						data = data[:len(data)-2]
					}
				} else {
					// Two colons on a line of their own are just
					// treated as a blank line, except that we do
					// set its indent level here in case it's
					// starting a new indent level.
					s.nextIndent = indent
					s.nextToken = &Token{
						Type:     BLANK,
						Data:     data[:0],
						Position: position,
					}
					return
				}
			}

			data = strings.TrimSpace(data)

			if len(data) == 0 {
				// Blank lines just continue whatever indent level is
				// current, so they will never produce synthetic indentation
				// tokens.
				s.nextIndent = s.currentIndent()
				s.nextToken = &Token{
					Type:     BLANK,
					Data:     data,
					Position: position,
				}
				return
			}

			// If we manage to get down here then we're just holding a
			// regular LINE token.
			position.Column = indent + 1
			s.nextIndent = indent
			s.nextToken = &Token{
				Type:     LINE,
				Data:     data,
				Position: position,
			}
			return

		} else {

			if s.lineScanner.Err() != nil {
				s.nextIndent = s.currentIndent()
				s.nextToken = &Token{
					Type:     ERROR,
					Data:     s.lineScanner.Err().Error(),
					Position: position,
				}
			} else {
				// we need to pop all of the active indents off the stack
				// before we actually emit the EOF token, so that the
				// parser gets a chance to exit any nested context it might
				// be in.
				s.nextIndent = 0

				s.nextToken = &Token{
					Type:     EOF,
					Data:     "",
					Position: position,
				}
			}
			return

		}
	}
}

// PushIndent produces a synthetic indentation level that is n greater than
// the latest.
//
// The parser should use this to give the scanner feedback about constructs
// that have introductory markers that the remaining lines must be indented
// relative to. For example, for bullet list items the remaining lines must be
// indented relative to the text *after* the bullet, not relative to the
// bullet itself.
//
// PushIndent should be used just after reading a line that introduces such
// a construct. An extra DEDENT token will then be emitted at the end of
// the construct.
func (s *Scanner) PushIndent(n int) {
	if s.peek != nil {
		panic("cannot call PushIndent with an active peek")
	}
	s.indents = append(s.indents, s.indents[len(s.indents)-1]+n)
}

// LazyIndent is similar to PushIndent except that the synthetic indentation
// level is not created until the next line token is processed, and the indent
// level of that token becomes the synthetic indent level is long as it is
// greater than the current indent level.
//
// The parser should use LazyIndent to give the scanner feedback about
// constructs that have "hanging" markers, like field and option lists.
func (s *Scanner) LazyIndent() {
	if s.peek != nil {
		panic("cannot call LazyIndent with an active peek")
	}
	s.lazyIndent = true
}

func (s *Scanner) currentIndent() int {
	return s.indents[len(s.indents)-1]
}
