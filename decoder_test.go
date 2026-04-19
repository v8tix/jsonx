package jsonx

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// partial only declares a subset of the fields in the JSON object.
// Used to verify Lenient() allows unknown fields.
type partial struct {
	Name string `json:"name"`
}

// ── Decoder — From (strict, default) ─────────────────────────────────────────

func TestDecoder_From_StrictHappyPath(t *testing.T) {
	got, err := Decoder[user]().From(readerOf(`{"name":"Alice","age":30}`))
	require.NoError(t, err)
	assert.Equal(t, user{Name: "Alice", Age: 30}, got)
}

func TestDecoder_From_StrictRejectsUnknownField(t *testing.T) {
	_, err := Decoder[user]().From(readerOf(`{"name":"Alice","age":30,"extra":"oops"}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBodyUnknownKey))
}

func TestDecoder_From_StrictRejectsEmptyBody(t *testing.T) {
	_, err := Decoder[user]().From(readerOf(``))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyBody))
}

func TestDecoder_From_StrictRejectsMultipleValues(t *testing.T) {
	_, err := Decoder[user]().From(readerOf(`{"name":"Alice","age":30}{"name":"Bob","age":25}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBodyValue))
}

// ── Decoder — FromBytes (strict) ─────────────────────────────────────────────

func TestDecoder_FromBytes_HappyPath(t *testing.T) {
	got, err := Decoder[user]().FromBytes([]byte(`{"name":"Bob","age":25}`))
	require.NoError(t, err)
	assert.Equal(t, user{Name: "Bob", Age: 25}, got)
}

func TestDecoder_FromBytes_StrictRejectsUnknownField(t *testing.T) {
	_, err := Decoder[user]().FromBytes([]byte(`{"name":"Bob","age":25,"x":1}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBodyUnknownKey))
}

// ── Decoder — Lenient().From ──────────────────────────────────────────────────

func TestDecoder_Lenient_AllowsUnknownFields(t *testing.T) {
	// JSON has "age" which is not declared in partial.
	got, err := Decoder[partial]().Lenient().From(readerOf(`{"name":"Alice","age":30}`))
	require.NoError(t, err)
	assert.Equal(t, partial{Name: "Alice"}, got)
}

func TestDecoder_Lenient_StillRejectsEmptyBody(t *testing.T) {
	_, err := Decoder[partial]().Lenient().From(readerOf(``))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyBody))
}

func TestDecoder_Lenient_StillRejectsMultipleValues(t *testing.T) {
	_, err := Decoder[partial]().Lenient().From(readerOf(`{"name":"Alice"}{"name":"Bob"}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBodyValue))
}

func TestDecoder_Lenient_StillRejectsSyntaxError(t *testing.T) {
	_, err := Decoder[partial]().Lenient().From(readerOf(`{"name":}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBadlyJSON))
}

// ── Decoder — Lenient().FromBytes ─────────────────────────────────────────────

func TestDecoder_Lenient_FromBytes_AllowsUnknownFields(t *testing.T) {
	got, err := Decoder[partial]().Lenient().FromBytes([]byte(`{"name":"Carol","age":40,"extra":true}`))
	require.NoError(t, err)
	assert.Equal(t, partial{Name: "Carol"}, got)
}

// ── Decoder — immutability ────────────────────────────────────────────────────

func TestDecoder_Lenient_DoesNotMutateOriginal(t *testing.T) {
	base := Decoder[user]()
	lenient := base.Lenient()

	// base must still be strict
	_, err := base.From(readerOf(`{"name":"Alice","age":30,"extra":"oops"}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBodyUnknownKey), "base decoder must remain strict")

	// lenient must allow unknown fields
	_, err = lenient.From(readerOf(`{"name":"Alice","age":30,"extra":"oops"}`))
	require.NoError(t, err)
}

// ── Decoder — scalar / collection types ──────────────────────────────────────

func TestDecoder_FromBytes_StringScalar(t *testing.T) {
	got, err := Decoder[string]().FromBytes([]byte(`"hello"`))
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestDecoder_FromBytes_NumberFallback(t *testing.T) {
	// Simulates gaia_models.go: final_answer may be a bare number.
	got, err := Decoder[any]().FromBytes([]byte(`140`))
	require.NoError(t, err)
	assert.InDelta(t, float64(140), got, 0.001)
}

func TestDecoder_FromBytes_SliceOfStructs(t *testing.T) {
	type item struct {
		ID int `json:"id"`
	}
	got, err := Decoder[[]item]().FromBytes([]byte(`[{"id":1},{"id":2}]`))
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, 1, got[0].ID)
	assert.Equal(t, 2, got[1].ID)
}

func TestDecoder_FromBytes_MapAny(t *testing.T) {
	// Simulates test helpers that capture full request bodies.
	got, err := Decoder[map[string]any]().FromBytes([]byte(`{"model":"gpt-4","temperature":0.7}`))
	require.NoError(t, err)
	assert.Equal(t, "gpt-4", got["model"])
}
