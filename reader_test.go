package jsonx

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── test helpers ─────────────────────────────────────────────────────────────

type user struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func readerOf(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}

// ── ReadJSONAs ────────────────────────────────────────────────────────────────

func TestReadJSONAs_HappyPath(t *testing.T) {
	got, err := ReadJSONAs[user](readerOf(`{"name":"Alice","age":30}`))
	require.NoError(t, err)
	assert.Equal(t, user{Name: "Alice", Age: 30}, got)
}

func TestReadJSONAs_ErrorCases(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr error
		wantMsg string
	}{
		{
			name:    "empty_body",
			input:   "",
			wantErr: ErrEmptyBody,
		},
		{
			name:    "syntax_error_includes_line_col",
			input:   `{"name":"Alice","age":}`,
			wantErr: ErrBadlyJSON,
			wantMsg: "line 1",
		},
		{
			name:    "syntax_error_multiline_correct_line",
			input:   "{\n  \"name\": \"Alice\",\n  \"age\": }\n",
			wantErr: ErrBadlyJSON,
			wantMsg: "line 3",
		},
		{
			name:    "unexpected_eof_includes_location",
			input:   `{"name":"Alice"`,
			wantErr: ErrBadlyJSON,
			wantMsg: "unexpected end",
		},
		{
			name:    "type_mismatch_includes_field",
			input:   `{"name":"Alice","age":"not-a-number"}`,
			wantErr: ErrBadJSONType,
			wantMsg: `"age"`,
		},
		{
			name:    "type_mismatch_includes_expected_go_type",
			input:   `{"name":"Alice","age":"not-a-number"}`,
			wantErr: ErrBadJSONType,
			wantMsg: "int",
		},
		{
			name:    "unknown_field",
			input:   `{"name":"Alice","age":30,"extra":"oops"}`,
			wantErr: ErrBodyUnknownKey,
			wantMsg: `"extra"`,
		},
		{
			name:    "multiple_values",
			input:   `{"name":"Alice","age":30}{"name":"Bob","age":25}`,
			wantErr: ErrBodyValue,
		},
		{
			name:    "trailing_garbage",
			input:   `{"name":"Alice","age":30} extra`,
			wantErr: ErrBodyValue,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadJSONAs[user](readerOf(tc.input))
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.wantErr),
				"expected errors.Is(err, %v), got: %v", tc.wantErr, err)
			if tc.wantMsg != "" {
				assert.Contains(t, err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestReadJSONAs_SyntaxError_ContainsContextSnippet(t *testing.T) {
	_, err := ReadJSONAs[user](readerOf(`{"name":"Alice","age":}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBadlyJSON))
	assert.Contains(t, err.Error(), "near", "error must include a context snippet")
}

func TestReadJSONAs_TypeMismatch_ContainsExpectedAndGot(t *testing.T) {
	_, err := ReadJSONAs[user](readerOf(`{"name":"Alice","age":"thirty"}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBadJSONType))
	assert.Contains(t, err.Error(), "int", "message must state the expected Go type")
	assert.Contains(t, err.Error(), "string", "message must state the received JSON token type")
}

func TestReadJSONAs_ErrorsIs_WorksThroughWrapper(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"badly_json", `{bad}`, ErrBadlyJSON},
		{"type_error", `{"name":1,"age":30}`, ErrBadJSONType},
		{"unknown_key", `{"name":"x","age":1,"z":2}`, ErrBodyUnknownKey},
		{"empty", ``, ErrEmptyBody},
		{"multiple", `{}{}`, ErrBodyValue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadJSONAs[user](readerOf(tc.input))
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.wantErr),
				"errors.Is must unwrap to %v; got %v", tc.wantErr, err)
		})
	}
}

// ── offsetToLineCol ───────────────────────────────────────────────────────────

func TestOffsetToLineCol(t *testing.T) {
	cases := []struct {
		name     string
		data     string
		offset   int64
		wantLine int
		wantCol  int
	}{
		{"start_of_single_line", `{"a":1}`, 0, 1, 1},
		{"mid_single_line", `{"a":1}`, 4, 1, 5},
		{"first_char_of_second_line", "{\n\"a\":1}", 2, 2, 1},
		{"mid_second_line", "{\n\"a\":1}", 4, 2, 3},
		{"third_line", "{\n\"a\":1,\n\"b\":2}", 10, 3, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			line, col := offsetToLineCol([]byte(tc.data), tc.offset)
			assert.Equal(t, tc.wantLine, line, "line mismatch")
			assert.Equal(t, tc.wantCol, col, "col mismatch")
		})
	}
}

// ── contextSnippet ────────────────────────────────────────────────────────────

func TestContextSnippet_ReturnsQuotedWindow(t *testing.T) {
	data := []byte(`{"name":"Alice","age":30}`)
	snippet := contextSnippet(data, 10)
	assert.True(t, strings.HasPrefix(snippet, `"`), "snippet must be quoted")
	assert.True(t, strings.HasSuffix(snippet, `"`), "snippet must be quoted")
}

func TestContextSnippet_ClampedAtBoundaries(t *testing.T) {
	data := []byte(`{"a":1}`)
	assert.NotEmpty(t, contextSnippet(data, 0))
	assert.NotEmpty(t, contextSnippet(data, int64(len(data)-1)))
}
