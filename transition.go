package rst

// A Transition marks a change in topic without creating a new section.
// It represents the idea normally communicated by a horizontal rule, or
// three spaced asterisks in printed works of fiction.
//
// Transition is both a structural element and a body element, since it can
// both separate body elements within a section and separate subsections
// of a section.
type Transition struct {
	bodyElementImpl
}

func (t *Transition) StructureChildElements() Structure {
	return nil
}
