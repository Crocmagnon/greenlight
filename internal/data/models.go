package data

import (
	"database/sql"
	"errors"
)

var (
	// ErrRecordNotFound is returned when a record couldn't be found in the DB.
	ErrRecordNotFound = errors.New("record not found")
	// ErrEditConflict is returned when updating a record with an incorrect version.
	// This kind of update is very likely due to a data race in the update endpoint.
	ErrEditConflict = errors.New("edit conflict")
)

// Models holds all model interfaces.
type Models struct {
	Movies interface { // use an interface to ease testing
		Insert(movie *Movie) error
		Get(id int64) (*Movie, error)
		Update(movie *Movie) error
		Delete(id int64) error
		GetAll(title string, genres []string, filters Filters) ([]*Movie, error)
	}
}

// NewModels initializes Models with the proper implementations
// for production use.
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
	}
}
