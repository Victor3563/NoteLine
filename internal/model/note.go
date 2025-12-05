package model

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"
)

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Text      string    `json:"text"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Deleted   bool      `json:"deleted"`
}

func NewNote(title, text string, tags []string) *Note {
	now := time.Now().UTC()
	var rnd [8]byte
	_, _ = rand.Read(rnd[:])

	raw := fmt.Sprintf("%d|%x|%d|%d", now.UnixNano(), rnd, len(title), len(text))
	h := sha1.Sum([]byte(raw))
	id := hex.EncodeToString(h[:])

	return &Note{
		ID:        id,
		Title:     title,
		Text:      text,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
		Deleted:   false,
	}
}
