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
	// PRAGMA is per connection; keep a single connection so it always applies
	conn.SetMaxOpenConns(1)

	stmts := []string{
		`PRAGMA foreign_keys = ON`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			%s INTEGER PRIMARY KEY AUTOINCREMENT,
			%s TEXT NOT NULL UNIQUE COLLATE NOCASE
		)`, TERM_TABLE, TERM_ID, TERM_KEY),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			%s INTEGER PRIMARY KEY AUTOINCREMENT,
			%s TEXT NOT NULL UNIQUE
		)`, USER_TABLE, USER_ID, USER_TAG),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			%s INTEGER NOT NULL REFERENCES %s(%s) ON DELETE CASCADE,
			%s INTEGER NOT NULL REFERENCES %s(%s) ON DELETE CASCADE,
			PRIMARY KEY (%s, %s)
		) WITHOUT ROWID`, UT_TABLE, UT_TERM, TERM_TABLE, TERM_ID, UT_USER, USER_TABLE, USER_ID, UT_TERM, UT_USER),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_userterm_user ON %s(%s)`, UT_TABLE, UT_USER),
	}
	for _, q := range stmts {
		if _, err := conn.Exec(q); err != nil {
			return fmt.Errorf("Error while initializing database: %s", err)
		}
	}
	return nil
}

func NewSQL(source string) (*SQLite, error) {
	db, err := sql.Open("sqlite", source)
	if err != nil {
		return nil, fmt.Errorf("Cannot open database file at %s. Error at: %s", source, err)
	}

	if err := defaultDB(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLite{c: db}, nil
}

func (s *SQLite) Close() error {
	if s.c == nil {
		return fmt.Errorf("Missing connection to close")
	}
	return s.c.Close()
}

func tag(id string) string { return "<@" + id + ">" }

// selectAllWhere: table/column are trusted constants, only the value is a placeholder
func selectAllWhere(table, col string) string {
	return fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", table, col)
}

/*
=========================================
* Term CRD no U cuz F*ck update
=========================================
*/

// Create
func (s *SQLite) CreateTerm(term string) error {
	_, err := s.c.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (?)", TERM_TABLE, TERM_KEY), strings.ToLower(term))
	return err
}

// Bulk Read
func (s *SQLite) ReadBulkTerm() ([]isql.Term, error) {
	rows, err := s.c.Query(fmt.Sprintf("SELECT %s, %s FROM %s", TERM_ID, TERM_KEY, TERM_TABLE))
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
	if err := s.c.QueryRow(selectAllWhere(TERM_TABLE, TERM_KEY), term).Scan(&t.Id, &t.Key); err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete
func (s *SQLite) DeleteTerm(term string) error {
	_, err := s.c.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", TERM_TABLE, TERM_KEY), term)
	return err
}

/*
=========================================
* User CR
=========================================
*/

// Create (no-op if already exists)
func (s *SQLite) CreateUser(id string) error {
	_, err := s.c.Exec(fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (?)", USER_TABLE, USER_TAG), tag(id))
	return err
}

// Read
func (s *SQLite) ReadUser(id string) (*isql.User, error) {
	var u isql.User
	if err := s.c.QueryRow(selectAllWhere(USER_TABLE, USER_TAG), tag(id)).Scan(&u.Id, &u.Tag); err != nil {
		return nil, err
	}
	return &u, nil
}

/*
=========================================
* UserTerm CRRD
=========================================
*/

// Create
func (s *SQLite) CreateUserTerm(tId int, id string) error {
	u, err := s.ReadUser(id)
	if err != nil {
		return err
	}
	_, err = s.c.Exec(fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (?, ?)", UT_TABLE, UT_TERM, UT_USER), tId, u.Id)
	return err
}

// Read
func (s *SQLite) ReadUsersByTerm(key string) (*isql.Notification, error) {
	term, err := s.ReadTerm(key)
	if err != nil {
		return nil, err
	}
	notif := isql.Notification{Id: term.Id, Key: term.Key}

	q := fmt.Sprintf("SELECT u.%s, u.%s FROM %s u JOIN %s ut ON ut.%s = u.%s WHERE ut.%s = ?",
		USER_ID, USER_TAG, USER_TABLE, UT_TABLE, UT_USER, USER_ID, UT_TERM)
	rows, err := s.c.Query(q, term.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u isql.User
		if err := rows.Scan(&u.Id, &u.Tag); err != nil {
			return nil, err
		}
		notif.Users = append(notif.Users, u)
	}
	return &notif, rows.Err()
}

// Read
func (s *SQLite) ReadTermsByUser(id string) (*isql.UserInfo, error) {
	user, err := s.ReadUser(id)
	if err != nil {
		return nil, err
	}
	info := isql.UserInfo{Id: user.Id, Tag: user.Tag}

	q := fmt.Sprintf("SELECT t.%s, t.%s FROM %s t JOIN %s ut ON ut.%s = t.%s WHERE ut.%s = ?",
		TERM_ID, TERM_KEY, TERM_TABLE, UT_TABLE, UT_TERM, TERM_ID, UT_USER)
	rows, err := s.c.Query(q, user.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t isql.Term
		if err := rows.Scan(&t.Id, &t.Key); err != nil {
			return nil, err
		}
		info.Terms = append(info.Terms, t)
	}
	return &info, rows.Err()
}

// Delete
func (s *SQLite) DeleteUserTerm(key, id string) error {
	user, err := s.ReadUser(id)
	if err != nil {
		return err
	}
	term, err := s.ReadTerm(key)
	if err != nil {
		return err
	}
	_, err = s.c.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ? AND %s = ?", UT_TABLE, UT_TERM, UT_USER), term.Id, user.Id)
	return err
}
