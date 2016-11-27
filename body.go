package rst

// Body represents body markup, which is the set of elements that make up
// the body of a section.
type Body []BodyElement

// BodyElement is an interface that represents the set of types that are
// intended to be used as body elements. Since body elements are diverse
// and have no common interface between them, callers must use a type switch
// (or similar) to do anything useful with a variable of type BodyElement.
type BodyElement interface {

	// Placeholder method just to declare implementation of this interface.
	// Does not return anything dependable, so should not actually be called.
	BodyElement() BodyElement
}

// bodyElementImpl is a struct that implements BodyElement and that can
// be embedded in other structs to mark them as implementations of BodyElement.
type bodyElementImpl struct{}

func (i *bodyElementImpl) BodyElement() BodyElement {
	return i
}

type Paragraph struct {
	bodyElementImpl
	Text
}

type BlockQuote struct {
	bodyElementImpl
	Quote       Body
	Attribution Text
}
