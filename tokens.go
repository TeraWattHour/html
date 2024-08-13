package html

type Token interface {
	Kind() string
}

type Location struct {
	Line   int
	Column int
	Cursor int
}

type StartTag struct {
	// Name must contain only letters, digits, hyphens, and colons, although it must start with a letter.
	Name          string
	Attributes    map[string]Attribute
	IsSelfClosing bool
	Location
}

func (t *StartTag) Kind() string {
	return "START_TAG"
}

type EndTag struct {
	Name string
	Location
}

func (t *EndTag) Kind() string {
	return "END_TAG"
}

type Text struct {
	Value string
	Location
}

func (t *Text) Kind() string {
	return "TEXT"
}

type Attribute struct {
	Name          string
	Value         string
	NameLocation  Location
	ValueLocation Location
}

type Illegal struct {
	Reason string
	Location
}

func (t *Illegal) Kind() string {
	return "ILLEGAL"
}

func (t *Illegal) Error() string {
	return t.Reason
}

type Eof struct {
	Location
}

func (t *Eof) Kind() string {
	return "EOF"
}
