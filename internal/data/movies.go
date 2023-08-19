package data

import (
	"time"

	"github.com/Crocmagnon/greenlight/internal/validator"
)

// Movie holds information about a single movie.
// It represents the entity as stored in the DB.
type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

// ValidateMovie validates a movie.
// The passed validator will contain all detected errors.
// The caller is expected to call [validator.Validator.Valid]
// after this method.
func ValidateMovie(validate *validator.Validator, movie *Movie) {
	const (
		titleMaxLength    = 500
		minYear           = 1888
		minGenres         = 1
		maxGenres         = 5
		fieldTitle        = "title"
		fieldYear         = "year"
		fieldRuntime      = "runtime"
		fieldGenres       = "genres"
		errMustBeProvided = "must be provided"
	)

	validate.Check(movie.Title != "", fieldTitle, errMustBeProvided)
	validate.Check(len(movie.Title) <= titleMaxLength, fieldTitle, "must not be more than 500 bytes long")

	validate.Check(movie.Year != 0, fieldYear, errMustBeProvided)
	validate.Check(movie.Year >= minYear, fieldYear, "must be greater than 1888")
	validate.Check(movie.Year <= int32(time.Now().Year()), fieldYear, "must not be in the future")

	validate.Check(movie.Runtime != 0, fieldRuntime, errMustBeProvided)
	validate.Check(movie.Runtime > 0, fieldRuntime, "must be a positive integer")

	validate.Check(movie.Genres != nil, fieldGenres, errMustBeProvided)
	validate.Check(len(movie.Genres) >= minGenres, fieldGenres, "must contain at least 1 genre")
	validate.Check(len(movie.Genres) <= maxGenres, fieldGenres, "must not contain more than 5 genres")
	validate.Check(validator.Unique(movie.Genres), fieldGenres, "must not contain duplicate values")
}
