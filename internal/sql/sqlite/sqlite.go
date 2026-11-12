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

const (
	USER_TABLE = "Users"
	TERM_TABLE = "Terms"
	UT_TABLE   = "UserTerm"

	// User table fields
	USER_ID  = "Id"
	USER_TAG = "tag"

	// Term table fields
	TERM_ID  = "Id"
	TERM_KEY = "key"

	// Link
	UT_TERM = "Id_Term"
	UT_USER = "Id_User"
)

func defaultDB(conn *sql.DB) error {
	stmt, err := conn.Prepare(`PRAGMA foreign_keys = ON; -- per connection, off by default in SQLite

CREATE TABLE IF NOT EXISTS ? (
    ?   INTEGER PRIMARY KEY AUTOINCREMENT,
    ?   TEXT NOT NULL UNIQUE COLLATE NOCASE
);

CREATE TABLE IF NOT EXISTS ? (
    ?  INTEGER PRIMARY KEY AUTOINCREMENT,
    ?  TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS ? (
    ? INTEGER NOT NULL REFERENCES Terms(?) ON DELETE CASCADE,
    ? INTEGER NOT NULL REFERENCES Users(?) ON DELETE CASCADE,
    PRIMARY KEY (?, ?)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_userterm_user ON UserTerm(?);
	`)
	if err != nil {
		return fmt.Errorf("Error while preparing statement: %s", err)
	}
	stmt.Exec(TERM_TABLE, TERM_ID, TERM_KEY, USER_TABLE, USER_ID, USER_TAG, UT_TABLE, UT_TERM, UT_USER, UT_TERM, UT_USER, UT_USER)
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

func (s *SQLite) selectAllWhere() (*sql.Stmt, error) {
	return s.c.Prepare("SELECT * FROM ? WHERE ? = ?")
}

/*
=========================================
* Term CRD no U cuz F*ck update
=========================================
*/

// Create
func (s *SQLite) CreateTerm(term string) error {
	stmt, err := s.c.Prepare(`INSERT INTO ? (?) VALUES (?)`)
	if err != nil {
		return err
	}

	stmt.Exec(TERM_TABLE, TERM_KEY, strings.ToLower(term))
	return nil
}

// Bulk Read
func (s *SQLite) ReadBulkTerm() ([]isql.Term, error) {
	stmt, err := s.c.Prepare("SELECT * FROM ?")
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(TERM_TABLE)
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
	stmt, err := s.selectAllWhere()
	if err != nil {
		return nil, err
	}

	var t isql.Term
	if err := stmt.QueryRow(TERM_TABLE, TERM_KEY, term).Scan(&t.Id, &t.Key); err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete
func (s *SQLite) DeleteTerm(term string) error {
	stmt, err := s.c.Prepare(`DELETE FROM ? WHERE ? = ?`)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(TERM_TABLE, TERM_KEY, term); err != nil {
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
	stmt, err := s.c.Prepare("SELECT COUNT(?) FROM ? WHERE ? = ?")
	if err != nil {
		return err
	}

	tag := "<@" + id + ">"
	var n int
	if err := stmt.QueryRow(USER_TAG, USER_TABLE, USER_TAG, tag).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	} else {
		stmt, err := s.c.Prepare(`INSERT INTO ? (?) VALUES (?)`)
		if err != nil {
			return err
		}
		stmt.Exec(USER_TABLE, USER_TAG, tag)
		return nil
	}
}

// Read
func (s *SQLite) ReadUser(id string) (*isql.User, error) {
	stmt, err := s.selectAllWhere()
	if err != nil {
		return nil, err
	}

	tag := "<@" + id + ">"
	var t isql.User
	if err := stmt.QueryRow(USER_TABLE, USER_TAG, tag).Scan(&t.Id, &t.Tag); err != nil {
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
	stmt, err := s.selectAllWhere()
	if err != nil {
		return err
	}

	tag := "<@" + id + ">"
	var t isql.User
	if err := stmt.QueryRow(USER_TABLE, USER_TAG, tag).Scan(&t.Id, &t.Tag); err != nil {
		return err
	}

	stmt, err = s.c.Prepare("INSERT INTO ? (?, ?) VALUES (?, ?)")
	if err != nil {
		return err
	}
	stmt.Exec(UT_TABLE, UT_TERM, UT_USER, tId, t.Id)
	return nil
}

// Read
func (s *SQLite) ReadUsersByTerm(key string) (*isql.Notification, error) {
	stmt, err := s.selectAllWhere()
	if err != nil {
		return nil, err
	}

	stmt1, err := s.selectAllWhere()
	if err != nil {
		return nil, err
	}

	var notif isql.Notification

	var term isql.Term
	if err := stmt.QueryRow(TERM_TABLE, TERM_KEY, key).Scan(&term.Id, &term.Key); err != nil {
		return nil, err
	}
	notif.Id = term.Id
	notif.Key = term.Key

	rows, err := stmt1.Query(UT_TABLE, UT_TERM, term.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		stmt2, err := s.selectAllWhere()
		if err != nil {
			return nil, err
		}

		var temp isql.UserTerm
		if err := rows.Scan(&temp.Id_Term, &temp.Id_User); err != nil {
			return nil, err
		}
		var user isql.User
		if err := stmt2.QueryRow(USER_TABLE, USER_ID, temp.Id_User).Scan(&user.Id, &user.Tag); err != nil {
			return nil, err
		}
		notif.Users = append(notif.Users, user)
	}

	return &notif, nil
}

// Read
func (s *SQLite) ReadTermsByUser(id string) (*isql.UserInfo, error) {
	stmt, err := s.selectAllWhere()
	if err != nil {
		return nil, err
	}

	stmt1, err := s.selectAllWhere()
	if err != nil {
		return nil, err
	}

	var info isql.UserInfo
	tag := "<@" + id + ">"

	var user isql.User
	if err := stmt.QueryRow(USER_TABLE, USER_TAG, tag).Scan(&user.Id, &user.Tag); err != nil {
		return nil, err
	}
	info.Id = user.Id
	info.Tag = user.Tag

	rows, err := stmt1.Query(UT_TABLE, UT_USER, user.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		stmt, err := s.selectAllWhere()
		if err != nil {
			return nil, err
		}

		var temp isql.UserTerm
		if err := rows.Scan(&temp.Id_Term, &temp.Id_User); err != nil {
			return nil, err
		}
		var term isql.Term
		if err := stmt.QueryRow(TERM_TABLE, TERM_ID, temp.Id_Term).Scan(&term.Id, &term.Key); err != nil {
			return nil, err
		}
		info.Terms = append(info.Terms, term)
	}

	return &info, nil
}

// Delete
func (s *SQLite) DeleteUserTerm(key, id string) error {
	tag := "<@" + id + ">"

	stmt, err := s.selectAllWhere()
	if err != nil {
		return err
	}
	var user isql.User
	if err := s.c.QueryRow(USER_TABLE, USER_TAG, tag).Scan(&user.Id, &user.Tag); err != nil {
		return err
	}

	stmt, err = s.selectAllWhere()
	if err != nil {
		return err
	}
	var term isql.Term
	if err := s.c.QueryRow(TERM_TABLE, TERM_KEY, key).Scan(&term.Id, &term.Key); err != nil {
		return err
	}
	stmt, err = s.c.Prepare("SELECT * FROM ? WHERE ? = ? and ? = ?")
	if err != nil {
		return err
	}
	var userTerm isql.UserTerm
	if err := s.c.QueryRow(UT_TABLE, UT_TERM, UT_USER, term.Id, user.Id).Scan(&userTerm.Id_Term, &userTerm.Id_User); err != nil {
		return err
	}

	stmt, err = s.c.Prepare("DELETE FROM ? WHERE ? = ? and ? = ?")
	if err != nil {
		return err
	}

	stmt.Exec(UT_TERM, UT_TERM, UT_USER, term.Id, user.Id)
	return nil
}
