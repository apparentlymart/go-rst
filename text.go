package rst

// Text represents inline markup, which is a mixture of plain text nodes
// and inline markup elements.
type Text []InlineElement

// InlineElement implementation.
//
// A "Text" is not intended to be used directly as an inline element, but this
// type can be embedded into a struct representing an inline markup element to
// easily implement simple markup, like emphasis.
func (t Text) InlineChildNodes() Text {
	return t
}

type InlineElement interface {
	InlineChildNodes() Text
}
