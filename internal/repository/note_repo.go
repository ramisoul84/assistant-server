package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type NoteRepository interface {
	Create(ctx context.Context, n *domain.Note) (*domain.Note, error)
	Update(ctx context.Context, id, userID int64, content string, datetime *time.Time, tags []string) (*domain.Note, error)
	Delete(ctx context.Context, id, userID int64) error
	GetAll(ctx context.Context, userID int64, from, to *time.Time) ([]domain.Note, error)
	GetUpcoming(ctx context.Context, from, to time.Time) ([]domain.Note, error)
}

type noteRepo struct{ db *sqlx.DB }

func NewNoteRepository(db *sqlx.DB) NoteRepository { return &noteRepo{db: db} }

// scanNote manually scans a row into a Note, handling the pq.StringArray for tags.
// This is required because sqlx cannot automatically convert Postgres text[] into []string.
func scanNote(row interface {
	Scan(dest ...any) error
}) (domain.Note, error) {
	var n domain.Note
	var tags pq.StringArray
	if err := row.Scan(
		&n.ID, &n.UserID, &n.Content, &n.Datetime, &tags, &n.CreatedAt, &n.UpdatedAt,
	); err != nil {
		return domain.Note{}, err
	}
	n.Tags = []string(tags)
	if n.Tags == nil {
		n.Tags = []string{}
	}
	return n, nil
}

func (r *noteRepo) Create(ctx context.Context, n *domain.Note) (*domain.Note, error) {
	const q = `
		INSERT INTO notes (user_id, content, datetime, tags)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, content, datetime, tags, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, q, n.UserID, n.Content, n.Datetime, pq.Array(n.Tags))
	result, err := scanNote(row)
	if err != nil {
		return nil, fmt.Errorf("noteRepo.Create: %w", err)
	}
	return &result, nil
}

func (r *noteRepo) Update(ctx context.Context, id, userID int64, content string, datetime *time.Time, tags []string) (*domain.Note, error) {
	const q = `
		UPDATE notes
		SET content = $1, datetime = $2, tags = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
		RETURNING id, user_id, content, datetime, tags, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, q, content, datetime, pq.Array(tags), id, userID)
	result, err := scanNote(row)
	if err != nil {
		return nil, fmt.Errorf("noteRepo.Update: %w", err)
	}
	return &result, nil
}

func (r *noteRepo) Delete(ctx context.Context, id, userID int64) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM notes WHERE id = $1 AND user_id = $2`, id, userID); err != nil {
		return fmt.Errorf("noteRepo.Delete: %w", err)
	}
	return nil
}

func (r *noteRepo) GetAll(ctx context.Context, userID int64, from, to *time.Time) ([]domain.Note, error) {
	q := `SELECT id, user_id, content, datetime, tags, created_at, updated_at
		  FROM notes WHERE user_id = $1`
	args := []any{userID}
	i := 2
	if from != nil {
		q += fmt.Sprintf(" AND (datetime IS NULL OR datetime >= $%d)", i)
		args = append(args, *from)
		i++
	}
	if to != nil {
		q += fmt.Sprintf(" AND (datetime IS NULL OR datetime <= $%d)", i)
		args = append(args, *to)
	}
	q += " ORDER BY COALESCE(datetime, created_at) DESC"

	return r.queryNotes(ctx, q, args...)
}

func (r *noteRepo) GetUpcoming(ctx context.Context, from, to time.Time) ([]domain.Note, error) {
	const q = `
		SELECT id, user_id, content, datetime, tags, created_at, updated_at
		FROM notes
		WHERE datetime IS NOT NULL AND datetime >= $1 AND datetime <= $2
		ORDER BY datetime ASC`

	return r.queryNotes(ctx, q, from, to)
}

// queryNotes runs a query and scans all rows into []domain.Note.
func (r *noteRepo) queryNotes(ctx context.Context, q string, args ...any) ([]domain.Note, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("noteRepo.queryNotes: %w", err)
	}
	defer rows.Close()

	var list []domain.Note
	for rows.Next() {
		var n domain.Note
		var tags pq.StringArray
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Content, &n.Datetime, &tags, &n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("noteRepo.queryNotes scan: %w", err)
		}
		n.Tags = []string(tags)
		if n.Tags == nil {
			n.Tags = []string{}
		}
		list = append(list, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("noteRepo.queryNotes rows: %w", err)
	}
	return list, nil
}
