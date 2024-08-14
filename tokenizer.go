package html

import (
	"errors"
	"iter"
	"regexp"
	"slices"
	"unicode"
)

func NewTokenizer(template string) Tokenizer {
	return Tokenizer{template: []rune(template), line: 1, column: 1}
}

func Tokenize(template string) iter.Seq[Token] {
	t := NewTokenizer(template)
	return func(yield func(Token) bool) {
		for token := t.next(); token.Kind() != "EOF" && yield(token); token = t.next() {
		}
	}
}

type Tokenizer struct {
	template []rune
	i        int
	line     int
	column   int
}

func (t *Tokenizer) next() Token {
	if t.match(regexp.MustCompile(`^(?i)<!DOCTYPE\s+`)) {
		return t.doctype()
	} else if t.is('<') && t.peek() == '/' {
		return t.endTag()
	} else if t.is('<') && isLetter(t.peek()) {
		return t.startTag()
	} else if t.is(0) {
		return &Eof{t.location()}
	}

	textLocation := t.location()
	for !t.is(0) && (!t.is('<') || (t.is('<') && !isLetter(t.peek()) && t.peek() != '/' && t.peek() != '!')) {
		t.advance()
	}

	return &Text{
		string(t.template[textLocation.Cursor:t.i]),
		textLocation,
	}
}

// https://html.spec.whatwg.org/multipage/syntax.html#the-doctype
func (t *Tokenizer) doctype() Token {
	location := t.location()

	for range len("<!DOCTYPE ") {
		t.advance()
	}

	t.skipWhitespace()
	if !t.match(regexp.MustCompile(`^(?i)html`)) {
		return &Illegal{"expected `html` after `<!DOCTYPE `", t.location()}
	}

	for range len("html") {
		t.advance()
	}

	t.skipWhitespace()
	if t.match(regexp.MustCompile(`^SYSTEM\s+("about:legacy-compat"|'about:legacy-compat')\s*>`)) {
		t.until('>')
		t.advance()
		return &Doctype{true, location}
	}

	if !t.consume('>') {
		return &Illegal{"malformed DOCTYPE, expected closing angle bracket", t.location()}
	}

	return &Doctype{Location: location}
}

func (t *Tokenizer) startTag() Token {
	var err error

	location := t.location()
	t.advance()

	if !isLetter(t.current()) {
		return &Illegal{Reason: "expected tag name", Location: t.location()}
	}

	tag := StartTag{
		Location:   location,
		Attributes: make(map[string]Attribute),
	}

	if tag.Name, err = t.tagName(); err != nil {
		return &Illegal{Reason: err.Error(), Location: t.location()}
	}

	t.skipWhitespace()

	for !t.is('>', '/') {
		attribute := Attribute{
			NameLocation: t.location(),
		}

		if attribute.Name, err = t.attributeName(); err != nil {
			return &Illegal{Reason: err.Error(), Location: t.location()}
		}

		t.skipWhitespace()
		if t.consume('=') {
			t.skipWhitespace()
			attribute.ValueLocation = t.location()

			// NOTE: contrary to 13.1.2.3, unquoted attribute values are disallowed
			if !t.is('"', '\'') {
				return &Illegal{Reason: "expected quotes in attribute definition", Location: t.location()}
			}

			if attribute.Value, err = t.string(); err != nil {
				return &Illegal{Reason: err.Error(), Location: t.location()}
			}
		}

		tag.Attributes[attribute.Name] = attribute

		t.skipWhitespace()
	}

	tag.IsSelfClosing = t.consume('/')

	if !t.consume('>') {
		return &Illegal{Reason: "expected closing angle bracket", Location: t.location()}
	}

	return &tag
}

func (t *Tokenizer) endTag() Token {
	var err error
	tag := EndTag{Location: t.location()}
	t.advance()
	t.advance()

	if !isLetter(t.current()) {
		return &Illegal{Reason: "expected tag name", Location: t.location()}
	}

	if tag.Name, err = t.tagName(); err != nil {
		return &Illegal{Reason: err.Error(), Location: t.location()}
	}

	t.skipWhitespace()

	if !t.consume('>') {
		return &Illegal{Reason: "expected closing angle bracket", Location: t.location()}
	}

	return &tag
}

func (t *Tokenizer) tagName() (string, error) {
	validate := func(c rune) bool {
		return isLetter(c) || c == '-' || c == ':'
	}

	start := t.i

	if !isLetter(t.advance()) {
		return "", errors.New("tag name must start with a letter")
	}

	for c := t.current(); !isWhitespace(c) && c != 0 && c != '>'; c = t.current() {
		if !validate(c) {
			return "", errors.New("unexpected character in tag name")
		}
		t.advance()
	}
	return string(t.template[start:t.i]), nil
}

func (t *Tokenizer) attributeName() (string, error) {
	validate := func(c rune) bool {
		return isDigit(c) || isLetter(c) || c == '-' || c == '_' || c == ':'
	}

	if !validate(t.current()) {
		return "", errors.New("attribute name must not start with a digit")
	}

	start := t.i
	for c := t.current(); !isWhitespace(c) && c != 0 && c != '>' && c != '='; c = t.current() {
		if !validate(c) {
			return "", errors.New("unexpected character in attribute name")
		}
		t.advance()
	}

	if t.is(0) {
		return "", errors.New("unexpected end of input")
	}

	return string(t.template[start:t.i]), nil
}

func (t *Tokenizer) string() (string, error) {
	literal := t.until(t.advance(), '\\')
	c := t.advance()
	if c != '"' && c != '\'' {
		return "", errors.New("expected closing quote")
	}
	return literal, nil
}

func (t *Tokenizer) skipWhitespace() {
	for isWhitespace(t.current()) {
		t.advance()
	}
}

func (t *Tokenizer) until(what rune, notAfter ...rune) string {
	start := t.i
	var previous rune

	for c := t.current(); c != 0; previous, c = t.advance(), t.current() {
		if c != what {
			continue
		}
		if !slices.Contains(notAfter, previous) {
			break
		}
	}
	return string(t.template[start:t.i])
}

func (t *Tokenizer) match(pattern *regexp.Regexp) bool {
	return pattern.MatchString(string(t.template[t.i:]))
}

func (t *Tokenizer) is(what ...rune) bool {
	return slices.Contains(what, t.current())
}

func (t *Tokenizer) consume(what rune) bool {
	if t.current() == what {
		t.advance()
		return true
	}
	return false
}

func (t *Tokenizer) current() rune {
	if t.i >= len(t.template) {
		return 0
	}
	return t.template[t.i]
}

func (t *Tokenizer) peek() rune {
	if t.i+1 >= len(t.template) {
		return 0
	}
	return t.template[t.i+1]
}

func (t *Tokenizer) advance() rune {
	previous := t.current()
	if previous == 0 {
		return 0
	}
	t.i++
	if previous == '\n' {
		t.line++
		t.column = 0
	}
	t.column++
	return previous
}

func (t *Tokenizer) location() Location {
	return Location{Line: t.line, Column: t.column, Cursor: t.i}
}

func isDigit(r rune) bool {
	return unicode.IsDigit(r) && r < 128
}

func isLetter(r rune) bool {
	return unicode.IsLetter(r) && r < 128
}

// Whitespace is defined to be U+0009 TAB, U+000A LF, U+000C FF, U+000D CR, or U+0020 SPACE
func isWhitespace(r rune) bool {
	return r == '\u0009' || r == '\u000A' || r == '\u000C' || r == '\u000D' || r == '\u0020'
}
