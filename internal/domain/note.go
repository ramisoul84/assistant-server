package domain

import (
	"database/sql"
	"time"
)

type Note struct {
	ID        int64        `db:"id"         json:"id"`
	UserID    int64        `db:"user_id"    json:"user_id"`
	Content   string       `db:"content"    json:"content"`
	Datetime  sql.NullTime `db:"datetime"   json:"datetime"`
	Tags      []string     `db:"tags"       json:"tags"`
	CreatedAt time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt time.Time    `db:"updated_at" json:"updated_at"`
}

func (n *Note) IsAppointment() bool { return n.Datetime.Valid }
