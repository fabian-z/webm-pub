package edtd

import (
	"fmt"
)

func parseHeader(lex *lexer, m elementMap) error {
	for {
		err, done := parseHeaderElement(lex, m)
		if err != nil {
			return err
		} else if done {
			return nil
		}
	}
}

func parseHeaderElement(lex *lexer, m elementMap) (error, bool) {
	nameTok, err := expectType(lex, alphaNum, control)
	if nameTok.val == "}" {
		return nil, true
	} else if nameTok.typ != alphaNum {
		return fmt.Errorf("unexpected '%s' found", nameTok), false
	} else if err != nil {
		return err, false
	}

	if _, err = expect(lex, &assignTok); err != nil {
		return err, false
	}

	var elem *tplElement
	for _, melem := range m {
		if melem.name == nameTok.val {
			elem = melem
			break
		}
	}
	if elem == nil {
		return fmt.Errorf("Unknown element %s in header", nameTok.val), false
	}

	valTok, err := expectType(lex, alphaNum, quotedString)
	if err != nil {
		return err, false
	}

	// default param and the header value take the same form. And we will be
	// handling the header value by setting it to the default value (which
	// parseDefParam fills in) and setting elem.mustMatchDef to true
	if err = parseDefParam(elem, valTok); err != nil {
		return err, false
	}

	elem.mustMatchDef = true

	if _, err = expect(lex, &semiColonTok); err != nil {
		return err, false
	}

	return nil, false
}
