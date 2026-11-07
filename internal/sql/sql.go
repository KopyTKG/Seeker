package sql

type SQL interface {
	Close() error
	AddTerm(term string) error
	GetTerms() ([]Term, error)
}
