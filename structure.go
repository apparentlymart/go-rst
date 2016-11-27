package rst

// Structure represents structural markup, which is a limited set of types
// used to define the overall structure of a document.
//
// For a valid RST document, a Structure sequence will be a list of sections,
// with each sequential pair of sections optionally separated by one transition.
type Structure []StructureElement

// StructureElement implementers can participate in the structural model of
// a document.
type StructureElement interface {

	// StructureChildElements can be used to traverse down the structural
	// tree towards the leaf nodes.
	//
	// Returns a Structure sequence that might be empty or nil.
	StructureChildElements() Structure
}

type Section struct {
	Title         Text
	Body          Body
	ChildElements Structure
}

func (s *Section) StructureChildElements() Structure {
	return s.ChildElements
}
