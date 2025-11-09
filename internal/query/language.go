package query

import (
	"fmt"
	"strings"
	"unicode"
)

type TokenType int

const (
	// specials
	TokenEOF TokenType = iota
	TokenError

	// literals
	TokenField
	TokenValue
	TokenNumber

	// ops
	TokenColon  // :
	TokenAt     // @
	TokenTilde  // ~
	TokenMinus  // -
	TokenLT     // < (future: less than)
	TokenGT     // > (future: greater than)
	TokenEQ     // = (future: equals)
	TokenNE     // != (future: not equals)
	TokenPipe   // | (future: OR separator)
	TokenLParen // ( (future: grouping)
	TokenRParen // ) (future: grouping)

	// (future: boolean operators)
	TokenAND // AND
	TokenOR  // OR
	TokenNOT // NOT
)

type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

func (t Token) String() string {
	switch t.Type {
	case TokenEOF:
		return "EOF"
	case TokenError:
		return fmt.Sprintf("ERROR(%s)", t.Value)
	case TokenField:
		return fmt.Sprintf("FIELD(%s)", t.Value)
	case TokenValue:
		return fmt.Sprintf("VALUE(%s)", t.Value)
	case TokenNumber:
		return fmt.Sprintf("NUMBER(%s)", t.Value)
	case TokenColon:
		return "COLON"
	case TokenAt:
		return "AT"
	case TokenTilde:
		return "TILDE"
	case TokenMinus:
		return "MINUS"
	case TokenLT:
		return "LT"
	case TokenGT:
		return "GT"
	case TokenEQ:
		return "EQ"
	case TokenNE:
		return "NE"
	case TokenPipe:
		return "PIPE"
	case TokenLParen:
		return "LPAREN"
	case TokenRParen:
		return "RPAREN"
	case TokenAND:
		return "AND"
	case TokenOR:
		return "OR"
	case TokenNOT:
		return "NOT"
	default:
		return fmt.Sprintf("UNKNOWN(%s)", t.Value)
	}
}

type Lexer struct {
	input  string
	pos    int
	ch     rune
	tokens []Token
}

func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		pos:    0,
		tokens: []Token{},
	}
	if len(input) > 0 {
		l.ch = rune(input[0])
	} else {
		l.ch = 0
	}
	return l
}

func Tokenize(input string) ([]Token, error) {
	lexer := NewLexer(input)
	return lexer.tokenize()
}

func (l *Lexer) tokenize() ([]Token, error) {
	for {
		token := l.nextToken()
		l.tokens = append(l.tokens, token)

		if token.Type == TokenEOF {
			break
		}
		if token.Type == TokenError {
			return l.tokens, fmt.Errorf("lexer error at position %d: %s", token.Pos, token.Value)
		}
	}
	return l.tokens, nil
}

func (l *Lexer) nextToken() Token {
	l.skipWhitespace()

	if l.ch == 0 {
		return Token{Type: TokenEOF, Pos: l.pos}
	}

	pos := l.pos

	switch l.ch {
	case ':':
		l.advance()
		return Token{Type: TokenColon, Value: ":", Pos: pos}
	case '@':
		l.advance()
		return Token{Type: TokenAt, Value: "@", Pos: pos}
	case '~':
		l.advance()
		return Token{Type: TokenTilde, Value: "~", Pos: pos}
	case '-':
		next := l.peek()
		if unicode.IsLetter(next) {
			l.advance()
			return Token{Type: TokenMinus, Value: "-", Pos: pos}
		}
		return l.readValue()
	case '<':
		l.advance()
		return Token{Type: TokenLT, Value: "<", Pos: pos}
	case '>':
		l.advance()
		return Token{Type: TokenGT, Value: ">", Pos: pos}
	case '=':
		l.advance()
		return Token{Type: TokenEQ, Value: "=", Pos: pos}
	case '!':
		if l.peek() == '=' {
			l.advance()
			l.advance()
			return Token{Type: TokenNE, Value: "!=", Pos: pos}
		}
		return Token{Type: TokenError, Value: "unexpected character: !", Pos: pos}
	case '|':
		l.advance()
		return Token{Type: TokenPipe, Value: "|", Pos: pos}
	case '(':
		l.advance()
		return Token{Type: TokenLParen, Value: "(", Pos: pos}
	case ')':
		l.advance()
		return Token{Type: TokenRParen, Value: ")", Pos: pos}
	case '"', '\'':
		return l.readQuotedValue()
	}

	if unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '+' {
		return l.readIdentifier()
	}

	ch := l.ch
	l.advance()
	return Token{
		Type:  TokenError,
		Value: fmt.Sprintf("unexpected character: %c", ch),
		Pos:   pos,
	}
}

func (l *Lexer) readIdentifier() Token {
	pos := l.pos
	var sb strings.Builder

	for l.ch != 0 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_' || l.ch == '-' || l.ch == '.' || l.ch == '+') {
		sb.WriteRune(l.ch)
		l.advance()
	}

	value := sb.String()

	switch strings.ToUpper(value) {
	case "AND":
		return Token{Type: TokenAND, Value: value, Pos: pos}
	case "OR":
		return Token{Type: TokenOR, Value: value, Pos: pos}
	case "NOT":
		return Token{Type: TokenNOT, Value: value, Pos: pos}
	}

	switch strings.ToLower(value) {
	case "status", "priority", "project", "tag", "due", "created", "updated":
		return Token{Type: TokenField, Value: strings.ToLower(value), Pos: pos}
	}

	return Token{Type: TokenValue, Value: value, Pos: pos}
}

func (l *Lexer) readValue() Token {
	pos := l.pos
	var sb strings.Builder

	for l.ch != 0 && !unicode.IsSpace(l.ch) && l.ch != ':' && l.ch != '@' && l.ch != '(' && l.ch != ')' {
		sb.WriteRune(l.ch)
		l.advance()
	}

	value := sb.String()
	return Token{Type: TokenValue, Value: value, Pos: pos}
}

func (l *Lexer) readQuotedValue() Token {
	pos := l.pos
	quote := l.ch
	l.advance()

	var sb strings.Builder

	for l.ch != 0 && l.ch != quote {
		if l.ch == '\\' && l.peek() == quote {
			l.advance()
			sb.WriteRune(l.ch)
			l.advance()
		} else {
			sb.WriteRune(l.ch)
			l.advance()
		}
	}

	if l.ch != quote {
		return Token{
			Type:  TokenError,
			Value: "unterminated quoted string",
			Pos:   pos,
		}
	}


	return Token{Type: TokenValue, Value: sb.String(), Pos: pos}
}

func (l *Lexer) skipWhitespace() {
	for l.ch != 0 && unicode.IsSpace(l.ch) {
		l.advance()
	}
}

func (l *Lexer) advance() {
	l.pos++
	if l.pos < len(l.input) {
		l.ch = rune(l.input[l.pos])
	} else {
		l.ch = 0
	}
}

func (l *Lexer) peek() rune {
	if l.pos+1 < len(l.input) {
		return rune(l.input[l.pos+1])
	}
	return 0
}

func IsQueryLanguage(input string) bool {
	input = strings.TrimSpace(input)

	if strings.HasPrefix(input, "@") {
		return true
	}

	if strings.Contains(input, ":") {
		knownFields := []string{"status:", "priority:", "project:", "tag:", "due:", "created:", "updated:"}
		for _, field := range knownFields {
			if strings.Contains(strings.ToLower(input), field) {
				return true
			}
		}
	}

	return false
}
