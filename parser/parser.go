package parser

import (
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/apparentlymart/go-rst"
)

func ParseFragment(r io.Reader, filename string) *rst.Fragment {
	scanner := NewScanner(r, filename)
	p := &parser{scanner}
	return p.ParseFragment()
}

type parser struct {
	*Scanner
}

func (p *parser) ParseFragment() *rst.Fragment {
	body, structure := p.parseStructureModel(EOF)
	return &rst.Fragment{
		Body:          body,
		ChildElements: structure,
	}
}

func (p *parser) parseStructureModel(endType TokenType) (rst.Body, rst.Structure) {
	var body rst.Body
	var structure rst.Structure
	for {
		p.SkipBlanks()

		next := p.Peek()

		if next.Type == endType {
			p.Read() // consume terminator
			break
		}

		if next.Type == EOF {
			err := &rst.Error{
				Message: "unexpected EOF",
				Pos:     next.Position,
			}
			if structure != nil {
				structure = append(structure, err)
			} else {
				body = append(body, err)
			}
			break
		}

		if marker, _ := p.detectBulletListItem(next); marker != 0 {
			if structure != nil {
				structure = append(structure, &rst.Error{
					Message: "can't start bullet list after structural",
					Pos:     next.Position,
				})
				break
			}
			listElem := p.parseBulletList(marker)
			body = append(body, listElem)
			continue
		}

		// If we manage to get down here then we have something that
		// isn't valid in structural model context, so we'll produce
		// an error and then try to recover.
		// TODO: actually do that, once we have a recovery mechanism
		panic("structure model can't start here")
	}

	return body, structure
}

func (p *parser) parseBody(endType TokenType) rst.Body {
	body, structure := p.parseStructureModel(endType)
	if structure != nil && len(structure) > 0 {
		body = append(body, &rst.Error{
			Message: "structural element not permitted here",
			Pos:     structure[0].Position(),
		})
		// TODO: append an error element to the body to report that there
		// were structure elements that are not valid in this context.
	}
	return body
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

func (p *parser) parseBulletList(marker rune) rst.BodyElement {

	items := make([]*rst.ListItem, 0, 2)
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
		items = append(items, &rst.ListItem{itemContent})
	}

	return &rst.BulletList{
		Items: items,
	}
}
