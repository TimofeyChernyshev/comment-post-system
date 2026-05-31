package cursorpagination

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode_RoundTrip(t *testing.T) {
	id := "550e8400-e29b-41d4-a716-446655440000"
	createdAt := time.Date(2025, 5, 31, 12, 0, 0, 0, time.UTC)

	encoded := Encode(createdAt, id)
	require.NotEmpty(t, encoded)

	cursor, err := Decode(encoded)
	require.NoError(t, err)

	assert.Equal(t, id, cursor.ID)
	assert.True(t, cursor.CreatedAt.Equal(createdAt), "timestamps should match")
}

func TestDecode_InvalidBase64(t *testing.T) {
	_, err := Decode("not-valid-base64!!!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor encoding")
}

func TestDecode_InvalidJSON(t *testing.T) {
	encoded := "bm90IGpzb24="
	_, err := Decode(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor payload")
}

func TestDecode_EmptyID(t *testing.T) {
	encoded := "eyJjcmVhdGVkX2F0IjoiMjAyNS0wNS0zMVQxMjowMDowMFoiLCJpZCI6IiJ9"
	_, err := Decode(encoded)
	require.Error(t, err)
	assert.EqualError(t, err, "cursor contains empty ID")
}

func TestDecode_ValidJSON_AdditionalFields(t *testing.T) {
	encoded := "eyJjcmVhdGVkX2F0IjoiMjAyNS0wNS0zMVQxMjowMDowMFoiLCJpZCI6ImFiYyIsImV4dHJhIjoiZmllbGQifQ=="
	cursor, err := Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, "abc", cursor.ID)
}
