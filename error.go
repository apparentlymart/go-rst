package rst

// Error is an element that can appear in structural, body and inline context
// which replaces an element that failed to parse correctly for some reason,
// giving some context about what failed.
type Error struct {
	Message string
	Pos     Position
	bodyElementImpl
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Position() Position {
	return e.Pos
}

func (e *Error) StructureChildElements() Structure {
	return nil
}

func (e *Error) InlineChildNodes() Text {
	return nil
}
