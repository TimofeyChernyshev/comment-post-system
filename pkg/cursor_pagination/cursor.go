package cursorpagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Cursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func Encode(createdAt time.Time, id string) string {
	payload, _ := json.Marshal(Cursor{CreatedAt: createdAt.UTC(), ID: id})

	return base64.StdEncoding.EncodeToString(payload)
}

func Decode(cursor string) (*Cursor, error) {
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	var result Cursor
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("invalid cursor payload: %w", err)
	}

	if result.ID == "" {
		return nil, errors.New("cursor contains empty ID")
	}

	return &result, nil
}
