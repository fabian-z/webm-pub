package edtd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"
)

// TODO tokenTyp
type tokentyp int

const (
	alphaNum tokentyp = iota
	control
	quotedString
	err
	eof
)

// token represents a single set of characters which *could* be a valid token of
// the given type
type token struct {
	typ tokentyp
	val string
}

// Returns the token's value as an error, or nil if the token is not of type
// Err. If the token is nil returns io.EOF, since that is the ostensible meaning
func (t *token) asError() error {
	if t.typ == eof {
		return io.EOF
	} else if t.typ == err {
		return errors.New(t.val)
	}
	return nil
}

func (t *token) String() string {
	if err := t.asError(); err != nil {
		return err.Error()
	} else {
		return fmt.Sprint(t.val)
	}
}

var (
	errInvalidUTF8 = errors.New("invalid utf8 character")
)

type lexerFunc func(*lexer) lexerFunc

// lexer reads through an io.Reader and emits tokens from it.
type lexer struct {
	r      *bufio.Reader
	outbuf *bytes.Buffer
	ch     chan *token
	state  lexerFunc
}

// newLexer constructs a new lexer struct and returns it. r is internally
// wrapped with a bufio.Reader, unless it already is one. This will spawn a
// go-routine which reads from r until it hits an error, at which point it will
// end execution.
func newLexer(r io.Reader) *lexer {
	var br *bufio.Reader
	var ok bool
	if br, ok = r.(*bufio.Reader); !ok {
		br = bufio.NewReader(r)
	}

	l := lexer{
		r:      br,
		ch:     make(chan *token, 1),
		outbuf: bytes.NewBuffer(make([]byte, 0, 1024)),
		state:  lexWhitespace,
	}

	return &l
}

// Returns the next available token. This method should not be called after any
// Err or EOF tokens
func (l *lexer) next() *token {
	for {
		select {
		case t := <-l.ch:
			return t
		default:
			l.state = l.state(l)
		}
	}
}

func (l *lexer) emit(t tokentyp) {
	str := l.outbuf.String()
	l.ch <- &token{
		typ: t,
		val: str,
	}
	l.outbuf.Reset()
}

func (l *lexer) peek() (rune, error) {
	r, err := l.readRune()
	if err != nil {
		return 0, err
	}
	if err = l.r.UnreadRune(); err != nil {
		return 0, err
	}
	return r, nil
}

func (l *lexer) readRune() (rune, error) {
	r, i, err := l.r.ReadRune()
	if err != nil {
		return 0, err
	} else if r == unicode.ReplacementChar && i == 1 {
		return 0, errInvalidUTF8
	}
	return r, nil
}

func (l *lexer) err(errR error) lexerFunc {
	if errR == io.EOF {
		l.ch <- &token{eof, ""}
	} else {
		l.ch <- &token{err, errR.Error()}
	}
	close(l.ch)
	return nil
}

func (l *lexer) errf(format string, args ...interface{}) lexerFunc {
	s := fmt.Sprintf(format, args...)
	l.ch <- &token{err, s}
	close(l.ch)
	return nil
}

func lexWhitespace(l *lexer) lexerFunc {
	r, err := l.readRune()
	if err != nil {
		return l.err(err)
	}

	if unicode.IsSpace(r) {
		return lexWhitespace
	} else if r == '/' {
		return lexComment
	}

	l.outbuf.WriteRune(r)

	if r == ':' {
		return lexColon
	} else if r == '"' {
		return lexQuotedString
	} else if allowedAlphaNum[r] || unicode.IsLetter(r) || unicode.IsNumber(r) {
		return lexAlphaNum
	} else {
		l.emit(control)
		return lexWhitespace
	}
}

func lexComment(l *lexer) lexerFunc {
	r, err := l.peek()
	// There is a / in the buffer. If there's an error or not another / after, I
	// guess it's a control character?
	if err != nil {
		l.emit(control)
		return l.err(err)
	} else if r != '/' {
		l.emit(control)
		return lexWhitespace
	}

	l.readRune()
	return lexCommentRest
}

// Reads until a newline, since everything after a // is tossed away
func lexCommentRest(l *lexer) lexerFunc {
	r, err := l.readRune()
	if err != nil {
		return l.err(err)
	}

	if r == '\n' {
		return lexWhitespace
	}
	return lexCommentRest
}

// Underscore is allowed by the spec, others aren't but it's convenient to have
// here because it's used in places like range and decimal numbers where it acts
// as part of a single thing
var allowedAlphaNum = map[rune]bool{
	'_': true,
	'-': true,
	'.': true,
	'>': true,
	'<': true,
	'=': true,
}

func lexAlphaNum(l *lexer) lexerFunc {
	r, err := l.peek()
	if err != nil {
		return l.err(err)
	}

	if allowedAlphaNum[r] || unicode.IsLetter(r) || unicode.IsNumber(r) {
		l.readRune()
		l.outbuf.WriteRune(r)
		return lexAlphaNum
	}
	l.emit(alphaNum)

	return lexWhitespace
}

func lexColon(l *lexer) lexerFunc {
	r, err := l.peek()
	if err != nil {
		return l.err(err)
	}

	if r == '=' {
		l.readRune()
		l.outbuf.WriteRune(r)
	}

	l.emit(control)

	return lexWhitespace
}

func lexQuotedString(l *lexer) lexerFunc {
	r, err := l.readRune()
	if err != nil {
		l.emit(quotedString)
		return l.err(err)
	}
	l.outbuf.WriteRune(r)

	if r == '\\' {
		r, err := l.readRune()
		if err != nil {
			l.emit(quotedString)
			return l.err(err)
		}
		l.outbuf.WriteRune(r)
	} else if r == '"' {
		l.emit(quotedString)
		return lexWhitespace
	}

	return lexQuotedString
}
