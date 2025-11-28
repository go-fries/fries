package parser

func newParseError(line, column int, msg string) error {
	return ParseError{Line: line, Column: column, Message: msg}
}
