package data

import (
	"database/sql"
	"errors"
)

// ErrRecordNotFound is returned when a record couldn't be found in the DB.
var ErrRecordNotFound = errors.New("record not found")

// Models holds all model interfaces.
type Models struct {
	Movies interface { // use an interface to ease testing
		Insert(movie *Movie) error
		Get(id int64) (*Movie, error)
		Update(movie *Movie) error
		Delete(id int64) error
	}
}

// NewModels initializes Models with the proper implementations
// for production use.
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
	}
}
