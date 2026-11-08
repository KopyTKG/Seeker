package sql

type SQL interface {
	Close() error
	// CRD Term (Update is not needed)
	CreateTerm(term string) error
	ReadBulkTerm() ([]Term, error)
	ReadTerm(term string) (*Term, error)
	DeleteTerm(term string) error
	// CR User
	AddUser(id string) error
	ReadUser(id string) (*User, error)
	// CRRD UserTerm
	CreateUserTerm(tId int, id string) error
	ReadUsersByTerm(key string) (*Notification, error)
	ReadTermsByUser(id string) (*UserInfo, error)
	DeleteUserTerm(key, id string) error
}
