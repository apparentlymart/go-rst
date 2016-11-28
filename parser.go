package rst

import (
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func ParseFragment(r io.Reader, filename string) *Fragment {
	scanner := NewScanner(r, filename)
	p := &parser{scanner}
	return p.ParseFragment()
}

type parser struct {
	*Scanner
}

func (p *parser) ParseFragment() *Fragment {
	body, structure := p.parseStructureModel(EOF)
	return &Fragment{
		Body:          body,
		ChildElements: structure,
	}
}

// structureModelParser is a temporary helper construct used within the parser
// to parse the "structure model": body elements followed by structure
// elements, possibly with transitions interspersed.
type structureModelParser struct {
	parser          *parser
	appendBody      func(BodyElement, Position)
	appendStructure func(StructureElement, Position)
	appendMixed     func(interface{}, Position)
	blockQuoteBody  func(Position)

	// used when parsing blockquote bodies, to capture the attribution.
	// if nil, attributions are not parsed.
	appendAttribution func(content Text, pos Position)
}

func (m *structureModelParser) parse(endType TokenType) {
	p := m.parser

	for {
		p.SkipBlanks()

		next := p.Peek()

		if next.Type == endType {
			p.Read() // consume terminator
			break
		}

		if next.Type == EOF {
			m.appendMixed(&Error{
				Message: "unexpected EOF",
				Pos:     next.Position,
			}, next.Position)
			break
		}

		if next.Type == INDENT {
			// An indent signals the beginning of a blockquote.
			// The parsing function for blockquotes can potentially return
			// multiple blockquotes if there is a chain of them separated by
			// attribution markers.
			startPos := next.Position
			blockQuoteElems := p.parseBlockQuotes(DEDENT)
			for _, elem := range blockQuoteElems {
				m.appendBody(elem, startPos)
			}
			continue
		}

		if next.Type == LATE_INDENT {
			// This is a signal from the scanner that it has encountered
			// a new indent level between whatever we just dealt with and
			// its preceding indent. This indicates that everything we've
			// seen so far was actually inside a blockquote, so we now
			// need to restructure the DOM to reflect that.
			p.Read() // eat LATE_INDENT token
			m.blockQuoteBody(next.Position)
			continue
		}

		// Only look for attribution syntax if the caller provided an
		// event handler for it.
		if m.appendAttribution != nil && next.Type == LINE {
			if strings.HasPrefix(next.Data, "--") {
				nextChar, ncLen := utf8.DecodeRuneInString(next.Data[2:])
				if unicode.IsSpace(nextChar) {
					firstLine := p.Read()
					startPos := firstLine.Position
					p.PushIndent(ncLen + 2)
					p.PushBackSuffix(firstLine, ncLen+2)
					attribution := p.parseText()

					if p.Peek().Type == DEDENT {
						p.Eat(DEDENT)
					} else {
						m.appendMixed(&Error{
							Message: "missing dedent after attribution",
							Pos:     startPos,
						}, startPos)
					}

					m.appendAttribution(attribution, startPos)
					continue
				}
			}
		}

		if marker, _ := p.detectBulletListItem(next); marker != 0 {
			startPos := next.Position
			listElem := p.parseBulletList(marker)
			m.appendBody(listElem, startPos)
			continue
		}

		if seq, marker, start, _ := p.detectEnumeratedListItem(next); seq != 0 {
			startPos := next.Position
			listElem := p.parseEnumeratedList(seq, marker, start)
			m.appendBody(listElem, startPos)
			continue
		}

		if next.Type == LINE {
			startPos := next.Position
			text := p.parseText()
			m.appendBody(&Paragraph{Text: text}, startPos)
			continue
		}

		// If we manage to get here then we've encountered a parser bug,
		// since by this point we should've dealt with all possible situations.
		p.Read() // Eat whatever is bothering us (TODO: seek forward to recover?)
		m.appendMixed(&Error{
			Message: "unexpected token: " + next.Type.String(),
			Pos:     next.Position,
		}, next.Position)
	}
}

func (p *parser) parseStructureModel(endType TokenType) (Body, Structure) {
	var body Body
	var structure Structure

	var model structureModelParser
	model = structureModelParser{
		parser: p,
		appendBody: func(elem BodyElement, pos Position) {
			body = append(body, elem)
		},
		blockQuoteBody: func(pos Position) {
			body = Body{
				&BlockQuote{
					Quote: body,
				},
			}
		},
		appendStructure: func(elem StructureElement, pos Position) {
			// transition into structure context
			model.appendStructure = func(elem StructureElement, pos Position) {
				structure = append(structure, elem)
			}
			model.appendBody = func(elem BodyElement, pos Position) {
				model.appendStructure(&Error{
					Message: "body elements may not appear after sections",
					Pos:     pos,
				}, pos)
			}
			model.blockQuoteBody = func(pos Position) {
				model.appendStructure(&Error{
					Message: "block quote cannot terminate here",
					Pos:     pos,
				}, pos)
			}
			model.appendMixed = func(elem interface{}, pos Position) {
				model.appendStructure(elem.(StructureElement), pos)
			}

			model.appendStructure(elem, pos)
		},
		appendMixed: func(elem interface{}, pos Position) {
			model.appendBody(elem.(BodyElement), pos)
		},
	}
	model.parse(endType)

	return body, structure
}

func (p *parser) parseBody(endType TokenType) Body {
	var body Body

	var model structureModelParser
	model = structureModelParser{
		parser: p,
		appendBody: func(elem BodyElement, pos Position) {
			body = append(body, elem)
		},
		blockQuoteBody: func(pos Position) {
			body = Body{
				&BlockQuote{
					Quote: body,
				},
			}
		},
		appendStructure: func(elem StructureElement, pos Position) {
			body = append(body, &Error{
				Message: "structure elements may not appear here",
				Pos:     pos,
			})
		},
		appendMixed: func(elem interface{}, pos Position) {
			model.appendBody(elem.(BodyElement), pos)
		},
	}
	model.parse(endType)

	return body
}

func (p *parser) parseBlockQuotes(endType TokenType) Body {
	indent := p.Read()
	if indent.Type != INDENT {
		// should never happen, given a correct caller
		panic("parseBlockQuote called when block quote can't start")
	}

	var current *BlockQuote
	quotes := make(Body, 0, 1)

	ensureCurrent := func() {
		if current == nil {
			current = &BlockQuote{}
			quotes = append(quotes, current)
		}
	}

	var model structureModelParser
	model = structureModelParser{
		parser: p,
		appendBody: func(elem BodyElement, pos Position) {
			ensureCurrent()
			current.Quote = append(current.Quote, elem)
		},
		blockQuoteBody: func(pos Position) {
			ensureCurrent()
			current.Quote = Body{
				&BlockQuote{
					Quote: current.Quote,
				},
			}
		},
		appendStructure: func(elem StructureElement, pos Position) {
			model.appendBody(&Error{
				Message: "structure elements may not appear here",
				Pos:     pos,
			}, pos)
		},
		appendMixed: func(elem interface{}, pos Position) {
			model.appendBody(elem.(BodyElement), pos)
		},
		appendAttribution: func(elem Text, pos Position) {
			current.Attribution = elem

			// an attribution signals the end of the current quote.
			// any further elements will begin another.
			current = nil
		},
	}
	model.parse(endType)

	return quotes
}

// parseText reads zero or more sequential LINE tokens, parses the result
// as inline markup, and returns a Text value representing the inline
// markup structure.
func (p *parser) parseText() Text {
	// This is currently just a placeholder implementation that doesn't
	// do any parsing of inline markup, since we don't yet have an inline
	// markup parser.
	result := make(Text, 0, 1)
	for {
		next := p.Peek()
		if next.Type != LINE {
			break
		}
		token := p.Read()
		result = append(result, CharData(token.Data))
	}
	return result
}

// Attempts to interpret the given token as the beginning of a bullet list
// item.
//
// If it is, returns the bullet list marker and the number of bytes
// of indent to require for subsequent lines. If it is not, returns
// (0, 0).
func (p *parser) detectBulletListItem(next *Token) (marker rune, indent int) {
	if next.Type != LINE {
		return 0, 0
	}

	firstChar, firstCharLen := utf8.DecodeRuneInString(next.Data)
	switch firstChar {
	case '*', '+', '-', '•', '‣', '⁃':
		// possibly a bullet list
		nextChar, nextCharLen := utf8.DecodeRuneInString(next.Data[firstCharLen:])

		switch {
		case nextChar == utf8.RuneError:
			return firstChar, firstCharLen
		case unicode.IsSpace(nextChar):
			return firstChar, firstCharLen + nextCharLen
		default:
			return 0, 0
		}
	default:
		return 0, 0
	}

}

func (p *parser) parseBulletList(marker rune) BodyElement {

	items := make([]*ListItem, 0, 2)
	for {
		p.SkipBlanks()
		next := p.Peek()
		itemMarker, indent := p.detectBulletListItem(next)
		if itemMarker != marker {
			// next is either not a list item or belongs to a different list
			break
		}

		firstLine := p.Read()

		// Let the scanner know that the subsequent lines will be indented
		// to align with the first character of the first line.
		p.PushIndent(indent)

		// Push back our first-line token with the prefix removed
		// so that p.parseBody can re-read it.
		p.PushBackSuffix(firstLine, indent)

		itemContent := p.parseBody(DEDENT)
		items = append(items, &ListItem{itemContent})
	}

	return &BulletList{
		Items: items,
	}
}

type enumSeq rune
type enumMarker rune

const (
	enumSeqInvalid    enumSeq = 0
	enumSeqArabic     enumSeq = '1'
	enumSeqAlphaUpper enumSeq = 'A'
	enumSeqAlphaLower enumSeq = 'a'
	enumSeqRomanUpper enumSeq = 'I'
	enumSeqRomanLower enumSeq = 'i'

	enumMarkerInvalid enumMarker = 0
	enumMarkerPeriod  enumMarker = '.'
	enumMarkerParens  enumMarker = '('
	enumMarkerRParen  enumMarker = ')'
)

// Attempts to interpret the given token as the beginning of an enumerated list
// item.
//
// If it is, returns the sequence, marker type, item ordinal, and indent level.
// If it is not, returns 0, 0, 0, 0.
//
// If it is, returns the bullet list marker and the number of bytes
// of indent to require for subsequent lines. If it is not, returns
// (0, 0).
func (p *parser) detectEnumeratedListItem(next *Token) (enumSeq, enumMarker, int, int) {
	if next.Type != LINE {
		return 0, 0, 0, 0
	}
	if len(next.Data) < 2 {
		return 0, 0, 0, 0
	}

	remain := next.Data
	first := remain[0]
	indent := 0
	marker := enumMarkerInvalid

	if first == '(' {
		marker = enumMarkerParens
		indent++
		remain = remain[1:]
		first = remain[0]
	}

	seq := enumSeqInvalid
	ordinal := 0

	switch {
	case first >= '0' && first <= '9':
		end := 0
		for end = 0; end < len(remain); end++ {
			if remain[end] < '0' || remain[end] > '9' {
				break
			}
			end++
		}
		indent = indent + end - 1
		num := remain[0 : end-1]
		remain = remain[end-1:]

		var err error
		ordinal, err = strconv.Atoi(num)
		if err != nil {
			return 0, 0, 0, 0
		}

		seq = enumSeqArabic

		//case first >= 'A' && first <= 'Z':

		//case first >= 'a' && first <= 'z':

	default:
		return 0, 0, 0, 0
	}

	if len(remain) == 0 {
		// If we have no more characters then there's no room for
		// the closing marker punctuation, so this can't be a list item.
		return 0, 0, 0, 0
	}

	closePunct := remain[0]

	if marker == enumMarkerParens {
		if closePunct != ')' {
			return 0, 0, 0, 0
		}
	} else {
		switch closePunct {
		case ')':
			marker = enumMarkerRParen
		case '.':
			marker = enumMarkerPeriod
		default:
			return 0, 0, 0, 0
		}
	}

	indent++
	remain = remain[1:]

	if len(remain) > 0 {
		if remain[0] != ' ' {
			return 0, 0, 0, 0
		}
		indent++
	}

	return seq, marker, ordinal, indent

}

func (p *parser) parseEnumeratedList(seq enumSeq, marker enumMarker, start int) BodyElement {
	nextOrd := start
	items := make([]*ListItem, 0, 2)
	for {
		p.SkipBlanks()
		next := p.Peek()
		itemSeq, itemMarker, ord, indent := p.detectEnumeratedListItem(next)
		if itemSeq != seq || itemMarker != marker || ord != nextOrd {
			// next is either not a list item or belongs to a different list
			break
		}
		nextOrd++

		firstLine := p.Read()

		// Let the scanner know that the subsequent lines will be indented
		// to align with the first character of the first line.
		p.PushIndent(indent)

		// Push back our first-line token with the prefix removed
		// so that p.parseBody can re-read it.
		p.PushBackSuffix(firstLine, indent)

		itemContent := p.parseBody(DEDENT)
		items = append(items, &ListItem{itemContent})
	}

	list := &EnumeratedList{
		Items:      items,
		FirstIndex: start,
	}

	switch seq {
	case enumSeqArabic:
		list.EnumType = EnumArabic
	case enumSeqAlphaUpper:
		list.EnumType = EnumUpperAlpha
	case enumSeqAlphaLower:
		list.EnumType = EnumLowerAlpha
	case enumSeqRomanUpper:
		list.EnumType = EnumUpperRoman
	case enumSeqRomanLower:
		list.EnumType = EnumLowerRoman
	default:
		panic("invalid enum seq")
	}

	switch marker {
	case enumMarkerPeriod:
		list.EnumSuffix = "."
	case enumMarkerParens:
		list.EnumPrefix = "("
		list.EnumSuffix = ")"
	case enumMarkerRParen:
		list.EnumSuffix = ")"
	default:
		panic("invalid enum marker")
	}

	return list
}
