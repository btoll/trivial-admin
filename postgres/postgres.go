package postgres

import (
	"database/sql"
)

type Postgres struct {
	db *sql.DB
}

func New(db *sql.DB) *Postgres {
	return &Postgres{
		db: db,
	}
}

// TODO
func (p *Postgres) Get(token string) error {
	return nil
}

// TODO
func (p *Postgres) Write(token string, data string) error {
	return nil
}
