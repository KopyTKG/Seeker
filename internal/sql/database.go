package sql

type Term struct {
	Id  int    `sql:"Id"`
	Key string `sql:"term"`
}

type User struct {
	Id  int    `sql:"Id"`
	Tag string `sql:"tag"`
}

type UserTerm struct {
	Id_Term int `sql:"Id_term"`
	Id_User int `sql:"Id_User"`
}

type Notification struct {
	Id    int    `sql:"Id_term"`
	Key   string `sql:"term"`
	Users []User
}

type UserInfo struct {
	Id    int
	Tag   string
	Terms []Term
}
