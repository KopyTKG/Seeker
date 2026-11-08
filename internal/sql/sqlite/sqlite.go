package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
	isql "thekrew.app/seeker/internal/sql"
)

type SQLite struct {
	c *sql.DB
}

func defaultDB(conn *sql.DB) error {
	stmt, err := conn.Prepare(`
PRAGMA foreign_keys = ON; -- per connection, off by default in SQLite

CREATE TABLE IF NOT EXISTS Terms (
    Id   INTEGER PRIMARY KEY AUTOINCREMENT,
    term TEXT NOT NULL UNIQUE COLLATE NOCASE
);

CREATE TABLE IF NOT EXISTS Users (
    Id  INTEGER PRIMARY KEY AUTOINCREMENT,
    tag TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS UserTerm (
    Id_term INTEGER NOT NULL REFERENCES Terms(Id) ON DELETE CASCADE,
    Id_User INTEGER NOT NULL REFERENCES Users(Id) ON DELETE CASCADE,
    PRIMARY KEY (Id_term, Id_User)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_userterm_user ON UserTerm(Id_User);
	`)
	if err != nil {
		return fmt.Errorf("Error while preparing statement: %s", err)
	}
	stmt.Exec()
	return nil
}

func NewSQL(source string) (*SQLite, error) {

	db, err := sql.Open("sqlite", source)
	if err != nil {
		return nil, fmt.Errorf("Cannot open database file at %s. Error at: %s", source, err)
	}

	defaultDB(db)

	sql := SQLite{c: db}

	return &sql, nil
}

func (s *SQLite) Close() error {
	if s.c == nil {
		return fmt.Errorf("Missing connection to close")
	}
	s.c.Close()
	return nil
}

func (s *SQLite) AddTerm(term string) error {
	stmt, err := s.c.Prepare(`
INSERT INTO Terms (term) VALUES (?)
		`)
	if err != nil {
		return err
	}

	stmt.Exec(term)
	return nil
}

func (s *SQLite) GetTerms() ([]isql.Term, error) {
	rows, err := s.c.Query("SELECT Id, term FROM Terms")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terms []isql.Term
	for rows.Next() {
		var t isql.Term
		if err := rows.Scan(&t.Id, &t.Key); err != nil {
			return nil, err
		}
		terms = append(terms, t)
	}
	return terms, rows.Err()
}

func (s *SQLite) GetTermCount() (int, error) {
	var n int
	if err := s.c.QueryRow("SELECT COUNT(*) FROM Terms").Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// ------------------------------------
// User

func (s *SQLite) AddUser(tag string) error {
	var n int
	if err := s.c.QueryRow("SELECT COUNT(tag) FROM Users WHERE tag = ?", tag).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	} else {
		stmt, err := s.c.Prepare(`INSERT INTO Users (tag) VALUES (?)`)
		if err != nil {
			return err
		}
		stmt.Exec(tag)
		return nil
	}
}
