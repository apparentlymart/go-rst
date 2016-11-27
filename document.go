package rst

type Document struct {
	Title    Text
	Subtitle Text

	// TODO: Decoration, Docinfo, Transition

	Body Body

	ChildElements Structure
}
