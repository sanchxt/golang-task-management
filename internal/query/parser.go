package query

import (
	"fmt"
	"strings"
)

type QueryFilter struct {
	Field    string
	Operator string // ":", "=", "<", ">", "!=", "<=" ">=" (MVP: just ":")
	Value    string
	IsNot    bool   // true if prefixed with - (exclusion)
	IsFuzzy  bool   // true if @~ or fuzzy matching requested
}

func (qf QueryFilter) String() string {
	prefix := ""
	if qf.IsNot {
		prefix = "-"
	}
	if qf.IsFuzzy {
		prefix += "~"
	}
	return fmt.Sprintf("%s%s%s%s", prefix, qf.Field, qf.Operator, qf.Value)
}

type ParsedQuery struct {
	Filters []QueryFilter
	Errors  []ParseError
}

type ParseError struct {
	Message string
	Pos     int
}

func (e ParseError) String() string {
	return fmt.Sprintf("parse error at position %d: %s", e.Pos, e.Message)
}

type Parser struct {
	tokens []Token
	pos    int
	errors []ParseError
}

func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
		errors: []ParseError{},
	}
}

func ParseQuery(input string) (*ParsedQuery, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return &ParsedQuery{Filters: []QueryFilter{}, Errors: []ParseError{}}, nil
	}

	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}

	parser := NewParser(tokens)
	return parser.parse()
}

func (p *Parser) parse() (*ParsedQuery, error) {
	filters := []QueryFilter{}

	for !p.isAtEnd() {
		if p.current().Type == TokenEOF {
			break
		}

		filter, err := p.parseFilter()
		if err != nil {
			p.errors = append(p.errors, ParseError{
				Message: err.Error(),
				Pos:     p.current().Pos,
			})
			p.skipToNextFilter()
			continue
		}

		if filter != nil {
			filters = append(filters, *filter)
		}
	}

	query := &ParsedQuery{
		Filters: filters,
		Errors:  p.errors,
	}

	if len(p.errors) > 0 {
		return query, fmt.Errorf("%s", p.errors[0].String())
	}

	return query, nil
}

func (p *Parser) parseFilter() (*QueryFilter, error) {
	token := p.current()

	if token.Type == TokenAt {
		return p.parseAtMention()
	}

	if token.Type == TokenMinus {
		return p.parseNegatedFilter()
	}

	if token.Type == TokenField {
		return p.parseFieldFilter()
	}

	if token.Type != TokenEOF {
		p.advance()
	}

	return nil, nil
}

func (p *Parser) parseAtMention() (*QueryFilter, error) {
	p.advance() // @

	isFuzzy := false
	if p.current().Type == TokenTilde {
		isFuzzy = true
		p.advance() // ~
	}

	if p.current().Type != TokenValue && p.current().Type != TokenField && p.current().Type != TokenNumber {
		return nil, fmt.Errorf("expected project name after @")
	}

	value := p.current().Value
	p.advance()

	return &QueryFilter{
		Field:    "project",
		Operator: ":",
		Value:    value,
		IsNot:    false,
		IsFuzzy:  isFuzzy,
	}, nil
}

func (p *Parser) parseNegatedFilter() (*QueryFilter, error) {
	p.advance() // -

	if p.current().Type != TokenField {
		return nil, fmt.Errorf("expected field name after -")
	}

	field := p.current().Value
	p.advance()

	if p.current().Type != TokenColon {
		return nil, fmt.Errorf("expected : after field name")
	}
	p.advance()

	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	return &QueryFilter{
		Field:    field,
		Operator: ":",
		Value:    value,
		IsNot:    true,
		IsFuzzy:  false,
	}, nil
}

func (p *Parser) parseFieldFilter() (*QueryFilter, error) {
	field := p.current().Value
	p.advance()

	hasColon := false
	if p.current().Type == TokenColon {
		hasColon = true
		p.advance()
	}

	operator := ":"
	switch p.current().Type {
	case TokenLT:
		operator = "<"
		p.advance()
	case TokenGT:
		operator = ">"
		p.advance()
	case TokenEQ:
		operator = "="
		p.advance()
	case TokenNE:
		operator = "!="
		p.advance()
	default:
		if !hasColon {
			return nil, fmt.Errorf("expected operator after field name '%s'", field)
		}
		operator = ":"
	}

	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	return &QueryFilter{
		Field:    field,
		Operator: operator,
		Value:    value,
		IsNot:    false,
		IsFuzzy:  false,
	}, nil
}

func (p *Parser) parseValue() (string, error) {
	token := p.current()

	if token.Type == TokenValue || token.Type == TokenNumber {
		value := token.Value
		p.advance()
		return value, nil
	}

	if token.Type == TokenField {
		value := token.Value
		p.advance()
		return value, nil
	}

	return "", fmt.Errorf("expected value, got %s", token.String())
}

func (p *Parser) skipToNextFilter() {
	for !p.isAtEnd() {
		token := p.current()
		if token.Type == TokenAt || token.Type == TokenMinus || token.Type == TokenField || token.Type == TokenEOF {
			return
		}
		p.advance()
	}
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.current().Type == TokenEOF
}

func (q *ParsedQuery) HasField(field string) bool {
	for _, filter := range q.Filters {
		if filter.Field == field && !filter.IsNot {
			return true
		}
	}
	return false
}

func (q *ParsedQuery) GetField(field string) *QueryFilter {
	for _, filter := range q.Filters {
		if filter.Field == field && !filter.IsNot {
			return &filter
		}
	}
	return nil
}

func (q *ParsedQuery) GetAllFields(field string) []QueryFilter {
	var filters []QueryFilter
	for _, filter := range q.Filters {
		if filter.Field == field {
			filters = append(filters, filter)
		}
	}
	return filters
}

func (q *ParsedQuery) HasErrors() bool {
	return len(q.Errors) > 0
}
