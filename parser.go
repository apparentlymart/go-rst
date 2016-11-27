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

func (p *parser) parseStructureModel(endType TokenType) (Body, Structure) {
	var body Body
	var structure Structure

	// The following functions handle the two states that the following
	// loop can find itself in.
	var appendBody func(BodyElement, Position)
	var appendStructure func(StructureElement, Position)
	var appendMixed func(interface{}, Position)

	// We start off in body context, and then the first time we encounter
	// structural markup we transition in structural context and these
	// functions change to respect that state.
	appendBody = func(elem BodyElement, pos Position) {
		body = append(body, elem)
	}
	appendStructure = func(elem StructureElement, pos Position) {
		// transition into structure context
		appendStructure = func(elem StructureElement, pos Position) {
			structure = append(structure, elem)
		}
		appendBody = func(elem BodyElement, pos Position) {
			appendStructure(&Error{
				Message: "body elements may not appear after sections",
				Pos:     pos,
			}, pos)
		}
		appendMixed = func(elem interface{}, pos Position) {
			appendStructure(elem.(StructureElement), pos)
		}

		appendStructure(elem, pos)
	}
	appendMixed = func(elem interface{}, pos Position) {
		appendBody(elem.(BodyElement), pos)
	}

	for {
		p.SkipBlanks()

		next := p.Peek()

		if next.Type == endType {
			p.Read() // consume terminator
			break
		}

		if next.Type == EOF {
			appendMixed(&Error{
				Message: "unexpected EOF",
				Pos:     next.Position,
			}, next.Position)
			break
		}

		if marker, _ := p.detectBulletListItem(next); marker != 0 {
			startPos := next.Position
			listElem := p.parseBulletList(marker)
			appendBody(listElem, startPos)
			continue
		}

		// If we get down here and still have a LINE token waiting then
		// we'll interpret what follows as a plain paragraph.
		if next.Type == LINE {
			startPos := next.Position
			text := p.parseText()
			appendBody(&Paragraph{Text: text}, startPos)
		}
	}

	return body, structure
}

func (p *parser) parseBody(endType TokenType) Body {
	body, structure := p.parseStructureModel(endType)
	if structure != nil && len(structure) > 0 {
		body = append(body, &Error{
			Message: "structural element not permitted here",
			Pos:     structure[0].Position(),
		})
		// TODO: append an error element to the body to report that there
		// were structure elements that are not valid in this context.
	}
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
