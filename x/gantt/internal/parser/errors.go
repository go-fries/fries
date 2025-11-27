package parser

import "fmt"

func newParseError(line, column int, msg string) error {
	return ParseError{Line: line, Column: column, Message: msg}
}

func unknownDirectiveError(line int, directive string) error {
	return ParseError{Line: line, Column: 1, Message: fmt.Sprintf("unknown directive: %s", directive)}
}
