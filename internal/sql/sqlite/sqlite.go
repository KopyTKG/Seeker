package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
	isql "thekrew.app/seeker/internal/sql"
)

/*
=========================================
* Common
=========================================
*/

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

/*
=========================================
* Term CRD no U cuz F*ck update
=========================================
*/

// Create
func (s *SQLite) CreateTerm(term string) error {
	stmt, err := s.c.Prepare(`INSERT INTO Terms (term) VALUES (?)`)
	if err != nil {
		return err
	}

	stmt.Exec(strings.ToLower(term))
	return nil
}

// Bulk Read
func (s *SQLite) ReadBulkTerm() ([]isql.Term, error) {
	rows, err := s.c.Query("SELECT * FROM Terms")
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

// Read
func (s *SQLite) ReadTerm(term string) (*isql.Term, error) {
	var t isql.Term
	if err := s.c.QueryRow("SELECT * FROM Terms WHERE term = ?", term).Scan(&t.Id, &t.Key); err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete
func (s *SQLite) DeleteTerm(term string) error {
	stmt, err := s.c.Prepare(`DELETE FROM Terms WHERE term = ?`)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(term); err != nil {
		return err
	}
	return nil
}

/*
=========================================
* User CR
=========================================
*/

// Create
func (s *SQLite) CreateUser(id string) error {
	tag := "<@" + id + ">"
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

// Read
func (s *SQLite) ReadUser(id string) (*isql.User, error) {
	tag := "<@" + id + ">"
	var t isql.User
	if err := s.c.QueryRow("SELECT * FROM Users WHERE tag = ?", tag).Scan(&t.Id, &t.Tag); err != nil {
		return nil, err
	}
	return &t, nil
}

/*
=========================================
* UserTerm CRRD
=========================================
*/

// Create
func (s *SQLite) CreateUserTerm(tId int, id string) error {
	tag := "<@" + id + ">"
	var t isql.User
	if err := s.c.QueryRow("SELECT * FROM Users WHERE tag = ?", tag).Scan(&t.Id, &t.Tag); err != nil {
		return err
	}

	stmt, err := s.c.Prepare("INSERT INTO UserTerm (Id_term, Id_User) VALUES (?, ?)")
	if err != nil {
		return err
	}
	stmt.Exec(tId, t.Id)
	return nil
}

// Read
func (s *SQLite) ReadUsersByTerm(key string) (*isql.Notification, error) {
	var notif isql.Notification

	var term isql.Term
	if err := s.c.QueryRow("SELECT * FROM Terms WHERE key = ?", key).Scan(&term.Id, &term.Key); err != nil {
		return nil, err
	}
	notif.Id = term.Id
	notif.Key = term.Key

	rows, err := s.c.Query("SELECT * FROM UserTerm WHERE Id_term = ?", term.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var temp isql.UserTerm
		if err := rows.Scan(&temp.Id_Term, &temp.Id_User); err != nil {
			return nil, err
		}
		var user isql.User
		if err := s.c.QueryRow("SELECT * FROM Users WHERE Id = ?", temp.Id_User).Scan(&user.Id, &user.Tag); err != nil {
			return nil, err
		}
		notif.Users = append(notif.Users, user)
	}

	return &notif, nil
}

// Read
func (s *SQLite) ReadTermsByUser(id string) (*isql.UserInfo, error) {
	var info isql.UserInfo
	tag := "<@" + id + ">"

	var user isql.User
	if err := s.c.QueryRow("SELECT * FROM Users WHERE tag = ?", tag).Scan(&user.Id, &user.Tag); err != nil {
		return nil, err
	}
	info.Id = user.Id
	info.Tag = user.Tag

	rows, err := s.c.Query("SELECT * FROM UserTerm WHERE Id_User = ?", user.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var temp isql.UserTerm
		if err := rows.Scan(&temp.Id_Term, &temp.Id_User); err != nil {
			return nil, err
		}
		var term isql.Term
		if err := s.c.QueryRow("SELECT * FROM Terms WHERE Id = ?", temp.Id_Term).Scan(&term.Id, &term.Key); err != nil {
			return nil, err
		}
		info.Terms = append(info.Terms, term)
	}

	return &info, nil
}

// Delete
func (s *SQLite) DeleteUserTerm(key, id string) error {
	tag := "<@" + id + ">"

	var user isql.User
	if err := s.c.QueryRow("SELECT * FROM Users WHERE tag = ?", tag).Scan(&user.Id, &user.Tag); err != nil {
		return err
	}

	var term isql.Term
	if err := s.c.QueryRow("SELECT * FROM Terms WHERE term = ?", key).Scan(&term.Id, &term.Key); err != nil {
		return err
	}

	var userTerm isql.UserTerm
	if err := s.c.QueryRow("SELECT * FROM UserTerm WHERE Id_term = ? and Id_User = ?", term.Id, user.Id).Scan(&userTerm.Id_Term, &userTerm.Id_User); err != nil {
		return err
	}

	stmt, err := s.c.Prepare("DELETE FROM UserTerm WHERE Id_term = ? and Id_User = ?")
	if err != nil {
		return err
	}

	stmt.Exec(term.Id, user.Id)
	return nil
}
