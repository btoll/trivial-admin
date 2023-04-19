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

func (p *Postgres) Create() error {
	return nil
}

func (p *Postgres) Read() error {
	return nil
}

func (p *Postgres) Update(data string) error {
	//trivial=> INSERT INTO games (owner_id, name, filename) VALUES
	//trivial-> ((SELECT user_id FROM users WHERE username = 'ben'), 'foo', 'foo.csv');
	return nil
}

func (p *Postgres) Delete() error {
	return nil
}
