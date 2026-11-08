package sql

type SQL interface {
	Close() error
	AddTerm(term string) error
	AddUser(tag string) error
	GetTerms() ([]Term, error)
	GetTermCount() (int, error)
}
