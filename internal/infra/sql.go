package infra

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"thekrew.app/seeker/internal/infra/model"
)

type Database struct {
	ORM *gorm.DB
}

func OpenDatabase(source string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(source), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	d := Database{ORM: db}

	err = d.ORM.AutoMigrate(&model.Term{}, &model.User{}, &model.CVE{})
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.ORM.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
