package query

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "simple field:value",
			input:    "status:pending",
			expected: []TokenType{TokenField, TokenColon, TokenValue, TokenEOF},
		},
		{
			name:     "multiple filters",
			input:    "status:pending priority:high",
			expected: []TokenType{TokenField, TokenColon, TokenValue, TokenField, TokenColon, TokenValue, TokenEOF},
		},
		{
			name:     "@mention",
			input:    "@backend",
			expected: []TokenType{TokenAt, TokenValue, TokenEOF},
		},
		{
			name:     "@~fuzzy mention",
			input:    "@~backend",
			expected: []TokenType{TokenAt, TokenTilde, TokenValue, TokenEOF},
		},
		{
			name:     "negated filter",
			input:    "-tag:wontfix",
			expected: []TokenType{TokenMinus, TokenField, TokenColon, TokenValue, TokenEOF},
		},
		{
			name:     "comparison operators",
			input:    "due:<2025-01-15",
			expected: []TokenType{TokenField, TokenColon, TokenLT, TokenValue, TokenEOF},
		},
		{
			name:     "quoted value",
			input:    `status:"in progress"`,
			expected: []TokenType{TokenField, TokenColon, TokenValue, TokenEOF},
		},
		{
			name:     "complex query",
			input:    "status:pending priority:high @backend tag:bug -tag:wontfix",
			expected: []TokenType{
				TokenField, TokenColon, TokenValue,
				TokenField, TokenColon, TokenValue,
				TokenAt, TokenValue,
				TokenField, TokenColon, TokenValue,
				TokenMinus, TokenField, TokenColon, TokenValue,
				TokenEOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("Tokenize() got %d tokens, expected %d", len(tokens), len(tt.expected))
			}

			for i, token := range tokens {
				if token.Type != tt.expected[i] {
					t.Errorf("Token[%d] type = %v, expected %v (value: %s)", i, token.Type, tt.expected[i], token.Value)
				}
			}
		})
	}
}

func TestTokenizeValues(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		tokenIndex    int
	}{
		{
			name:          "simple value",
			input:         "status:pending",
			expectedValue: "pending",
			tokenIndex:    2,
		},
		{
			name:          "quoted value with spaces",
			input:         `status:"in progress"`,
			expectedValue: "in progress",
			tokenIndex:    2,
		},
		{
			name:          "value with hyphen",
			input:         "priority:high-priority",
			expectedValue: "high-priority",
			tokenIndex:    2,
		},
		{
			name:          "@mention value",
			input:         "@backend-api",
			expectedValue: "backend-api",
			tokenIndex:    1,
		},
		{
			name:          "date value",
			input:         "due:2025-01-15",
			expectedValue: "2025-01-15",
			tokenIndex:    2,
		},
		{
			name:          "relative date",
			input:         "due:+7d",
			expectedValue: "+7d",
			tokenIndex:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}

			if tt.tokenIndex >= len(tokens) {
				t.Fatalf("Token index %d out of range (got %d tokens)", tt.tokenIndex, len(tokens))
			}

			if tokens[tt.tokenIndex].Value != tt.expectedValue {
				t.Errorf("Token[%d] value = %q, expected %q", tt.tokenIndex, tokens[tt.tokenIndex].Value, tt.expectedValue)
			}
		})
	}
}

func TestTokenizeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unterminated quote",
			input: `status:"pending`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if err == nil {
				hasError := false
				for _, token := range tokens {
					if token.Type == TokenError {
						hasError = true
						break
					}
				}
				if !hasError {
					t.Errorf("Tokenize() expected error or error token, got nil")
				}
			}
		})
	}
}

func TestIsQueryLanguage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "field:value syntax",
			input:    "status:pending",
			expected: true,
		},
		{
			name:     "@mention syntax",
			input:    "@backend",
			expected: true,
		},
		{
			name:     "multiple filters",
			input:    "status:pending priority:high",
			expected: true,
		},
		{
			name:     "plain text",
			input:    "find tasks",
			expected: false,
		},
		{
			name:     "text with colon not field",
			input:    "meeting at 3:00pm",
			expected: false,
		},
		{
			name:     "priority field",
			input:    "priority:high",
			expected: true,
		},
		{
			name:     "due field",
			input:    "due:today",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "@~ fuzzy mention",
			input:    "@~back",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsQueryLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("IsQueryLanguage(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLexerFieldRecognition(t *testing.T) {
	knownFields := []string{"status", "priority", "project", "tag", "due", "created", "updated"}

	for _, field := range knownFields {
		t.Run(field, func(t *testing.T) {
			input := field + ":value"
			tokens, err := Tokenize(input)
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}

			if len(tokens) < 1 || tokens[0].Type != TokenField {
				t.Errorf("Expected first token to be TokenField, got %v", tokens[0].Type)
			}

			if tokens[0].Value != field {
				t.Errorf("Expected field value %q, got %q", field, tokens[0].Value)
			}
		})
	}
}
