package sql

type SQL interface {
	Close() error
	// CRD Term (Update is not needed)
	CreateTerm(term string) error
	ReadBulkTerm() ([]Term, error)
	ReadTerm(term string) (*Term, error)
	DeleteTerm(term string) error
	// CRD User (No update since input isn't givven by user but by session)
	AddUser(id string) error
	ReadUser(id string) (*User, error)
}
