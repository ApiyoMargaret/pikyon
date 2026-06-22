package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type UserRepository struct{ db *sql.DB }
type MemoryRepository struct{ db *sql.DB }
type ShareRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) *UserRepository     { return &UserRepository{db: db} }
func NewMemoryRepository(db *sql.DB) *MemoryRepository { return &MemoryRepository{db: db} }
func NewShareRepository(db *sql.DB) *ShareRepository   { return &ShareRepository{db: db} }

// ============================================================
// USER REPOSITORY
// ============================================================

func (r *UserRepository) Create(ctx context.Context, email, name, passwordHash, lang string) (*User, error) {
	var u User
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (email, name, password, preferred_lang)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, email, name, preferred_lang, created_at`,
		email, name, passwordHash, lang,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PreferredLang, &u.CreatedAt)
	return &u, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, name, password, pin_hash, preferred_lang, created_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Password, &u.PINHash, &u.PreferredLang, &u.CreatedAt)
	return &u, err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, name, password, pin_hash, preferred_lang, created_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Password, &u.PINHash, &u.PreferredLang, &u.CreatedAt)
	return &u, err
}

func (r *UserRepository) SetPIN(ctx context.Context, userID, pinHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET pin_hash = $1 WHERE id = $2`, pinHash, userID)
	return err
}

func (r *UserRepository) UpdateLanguage(ctx context.Context, userID, lang string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET preferred_lang = $1 WHERE id = $2`, lang, userID)
	return err
}

// ============================================================
// MEMORY REPOSITORY
// ============================================================

func (r *MemoryRepository) List(ctx context.Context, userID string) ([]*Memory, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, title, story, location, memory_date,
		        visibility, is_locked, media_key, media_type,
		        ai_vibe, ai_vibe_reason, ai_caption, ai_tone,
		        tags, created_at, updated_at
		 FROM memories
		 WHERE user_id = $1
		 ORDER BY COALESCE(memory_date, created_at::date) DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		m := &Memory{}
		err := rows.Scan(
			&m.ID, &m.UserID, &m.Title, &m.Story, &m.Location,
			&m.MemoryDate, &m.Visibility, &m.IsLocked,
			&m.MediaKey, &m.MediaType,
			&m.AIVibe, &m.AIVibeReason, &m.AICaption, &m.AITone,
			&m.Tags, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		m.HasMedia = m.MediaKey.Valid && m.MediaKey.String != ""
		m.MediaKey = sql.NullString{}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func (r *MemoryRepository) GetByID(ctx context.Context, userID, memoryID string) (*Memory, error) {
	m := &Memory{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, title, story, location, memory_date,
		        visibility, is_locked, media_key, media_type,
		        ai_vibe, ai_vibe_reason, ai_caption, ai_tone,
		        tags, created_at, updated_at
		 FROM memories
		 WHERE id = $1 AND user_id = $2`, memoryID, userID,
	).Scan(
		&m.ID, &m.UserID, &m.Title, &m.Story, &m.Location,
		&m.MemoryDate, &m.Visibility, &m.IsLocked,
		&m.MediaKey, &m.MediaType,
		&m.AIVibe, &m.AIVibeReason, &m.AICaption, &m.AITone,
		&m.Tags, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.HasMedia = m.MediaKey.Valid && m.MediaKey.String != ""
	return m, nil
}

func (r *MemoryRepository) GetByIDWithKey(ctx context.Context, userID, memoryID string) (*Memory, error) {
	m := &Memory{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, title, story, location, memory_date,
		        visibility, is_locked, media_key, media_type,
		        ai_vibe, ai_vibe_reason, ai_caption, ai_tone,
		        tags, created_at, updated_at
		 FROM memories
		 WHERE id = $1 AND user_id = $2`, memoryID, userID,
	).Scan(
		&m.ID, &m.UserID, &m.Title, &m.Story, &m.Location,
		&m.MemoryDate, &m.Visibility, &m.IsLocked,
		&m.MediaKey, &m.MediaType,
		&m.AIVibe, &m.AIVibeReason, &m.AICaption, &m.AITone,
		&m.Tags, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.HasMedia = m.MediaKey.Valid && m.MediaKey.String != ""
	return m, nil
}

func (r *MemoryRepository) Create(ctx context.Context, input CreateMemoryInput) (*Memory, error) {
	m := &Memory{}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO memories
		 (user_id, title, story, location, memory_date, visibility,
		  is_locked, media_key, media_type, tags)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, user_id, title, story, location, memory_date,
		           visibility, is_locked, media_key, media_type,
		           ai_vibe, ai_vibe_reason, ai_caption, ai_tone,
		           tags, created_at, updated_at`,
		input.UserID, input.Title, input.Story, input.Location,
		input.MemoryDate, input.Visibility, input.IsLocked,
		input.MediaKey, input.MediaType, pq.Array(input.Tags),
	).Scan(
		&m.ID, &m.UserID, &m.Title, &m.Story, &m.Location,
		&m.MemoryDate, &m.Visibility, &m.IsLocked,
		&m.MediaKey, &m.MediaType,
		&m.AIVibe, &m.AIVibeReason, &m.AICaption, &m.AITone,
		&m.Tags, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.HasMedia = m.MediaKey.Valid
	m.MediaKey = sql.NullString{}
	return m, nil
}

func (r *MemoryRepository) Update(ctx context.Context, userID, memoryID string, updates map[string]interface{}) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE memories SET
		 title = COALESCE($1, title),
		 story = COALESCE($2, story),
		 location = COALESCE($3, location),
		 visibility = COALESCE($4, visibility),
		 is_locked = COALESCE($5, is_locked)
		 WHERE id = $6 AND user_id = $7`,
		updates["title"], updates["story"], updates["location"],
		updates["visibility"], updates["is_locked"],
		memoryID, userID,
	)
	return err
}

func (r *MemoryRepository) UpdateAI(ctx context.Context, userID, memoryID, vibe, vibeReason, caption, tone string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE memories
		 SET ai_vibe=$1, ai_vibe_reason=$2, ai_caption=$3, ai_tone=$4
		 WHERE id=$5 AND user_id=$6`,
		vibe, vibeReason, caption, tone, memoryID, userID)
	return err
}

func (r *MemoryRepository) UpdateMediaKey(ctx context.Context, userID, memoryID, mediaKey, mediaType string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE memories SET media_key=$1, media_type=$2 WHERE id=$3 AND user_id=$4`,
		mediaKey, mediaType, memoryID, userID)
	return err
}

func (r *MemoryRepository) Delete(ctx context.Context, userID, memoryID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM memories WHERE id=$1 AND user_id=$2`, memoryID, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ============================================================
// SHARE REPOSITORY
// ============================================================

func (r *ShareRepository) Create(ctx context.Context, memoryID, ownerID string) (*SharedAccess, error) {
	var sa SharedAccess
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO shared_access (memory_id, owner_id)
		 VALUES ($1,$2)
		 RETURNING id, memory_id, owner_id, share_token, expires_at, view_count, created_at`,
		memoryID, ownerID,
	).Scan(&sa.ID, &sa.MemoryID, &sa.OwnerID, &sa.ShareToken,
		&sa.ExpiresAt, &sa.ViewCount, &sa.CreatedAt)
	return &sa, err
}

func (r *ShareRepository) GetByToken(ctx context.Context, token string) (*SharedAccess, error) {
	var sa SharedAccess
	err := r.db.QueryRowContext(ctx,
		`SELECT id, memory_id, owner_id, share_token, expires_at, view_count, created_at
		 FROM shared_access WHERE share_token=$1`, token,
	).Scan(&sa.ID, &sa.MemoryID, &sa.OwnerID, &sa.ShareToken,
		&sa.ExpiresAt, &sa.ViewCount, &sa.CreatedAt)
	return &sa, err
}

func (r *ShareRepository) IncrementViewCount(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE shared_access SET view_count=view_count+1 WHERE share_token=$1`, token)
	return err
}

func (r *ShareRepository) Delete(ctx context.Context, memoryID, ownerID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM shared_access WHERE memory_id=$1 AND owner_id=$2`, memoryID, ownerID)
	return err
}

var _ = fmt.Sprintf