package rst

import (
	"io"
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

	// used when parsing blockquote bodies, to capture the attribution.
	// if nil, attributions are not parsed.
	// This will be called zero or one times, and if it is called there
	// will be no more appendBody or appendStructure calls afterwards.
	setAttribution func(Text, Position)
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

		if marker, _ := p.detectBulletListItem(next); marker != 0 {
			startPos := next.Position
			listElem := p.parseBulletList(marker)
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
