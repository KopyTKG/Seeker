package sql

type Term struct {
	Id  int    `sql:"Id"`
	Key string `sql:"term"`
}

type User struct {
	Id  int    `sql:"Id"`
	Tag string `sql:"tag"`
}

type Notifications struct {
	Id    int    `sql:"Id_User"`
	Tag   string `sql:"tag"`
	Terms []Term `sql:"Term"`
}
