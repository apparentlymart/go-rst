package rst

type BulletList struct {
	bodyElementImpl
	Items []*ListItem
}

type EnumeratedList struct {
	bodyElementImpl
	EnumType   EnumType
	EnumPrefix string
	EnumSuffix string
	FirstIndex int
	Items      []*ListItem
}

type ListItem struct {
	Body
}

type EnumType string

const (
	EnumArabic     EnumType = "arabic"
	EnumLowerAlpha EnumType = "loweralpha"
	EnumUpperAlpha EnumType = "upperalpha"
	EnumLowerRoman EnumType = "lowerroman"
	EnumUpperRoman EnumType = "upperroman"
)
