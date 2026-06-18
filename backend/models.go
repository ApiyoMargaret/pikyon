package main

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type User struct {
	ID            string         `json:"id"`
	Email         string         `json:"email"`
	Name          string         `json:"name"`
	Password      string         `json:"-"`
	PINHash       sql.NullString `json:"-"`
	AvatarURL     sql.NullString `json:"avatar_url"`
	PreferredLang string         `json:"preferred_lang"`
	CreatedAt     time.Time      `json:"created_at"`
}

type Memory struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Title         string         `json:"title"`
	Story         sql.NullString `json:"story"`
	Location      sql.NullString `json:"location"`
	MemoryDate    *time.Time     `json:"memory_date"`
	Visibility    string         `json:"visibility"`
	IsLocked      bool           `json:"is_locked"`
	MediaKey      sql.NullString `json:"-"`
	MediaType     string         `json:"media_type"`
	MediaThumbnail sql.NullString `json:"-"`
	AIVibe        sql.NullString `json:"ai_vibe"`
	AIVibeReason  sql.NullString `json:"ai_vibe_reason"`
	AICaption     sql.NullString `json:"ai_caption"`
	AITone        sql.NullString `json:"ai_tone"`
	Tags          pq.StringArray `json:"tags"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	HasMedia      bool           `json:"has_media"`
}

type SharedAccess struct {
	ID         string         `json:"id"`
	MemoryID   string         `json:"memory_id"`
	OwnerID    string         `json:"owner_id"`
	ShareToken string         `json:"share_token"`
	ExpiresAt  *time.Time     `json:"expires_at"`
	ViewCount  int            `json:"view_count"`
	MaxViews   sql.NullInt64  `json:"max_views"`
	CreatedAt  time.Time      `json:"created_at"`
}

type AIAnalysis struct {
	Vibe          string            `json:"vibe"`
	VibeReason    string            `json:"vibe_reason"`
	Tone          string            `json:"tone"`
	Captions      map[string]string `json:"captions"`
	SuggestedTags []string          `json:"suggested_tags"`
}

type CreateMemoryInput struct {
	UserID     string
	Title      string
	Story      *string
	Location   *string
	MemoryDate *time.Time
	Visibility string
	IsLocked   bool
	MediaKey   *string
	MediaType  string
	Tags       []string
}