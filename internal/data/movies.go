package data

import (
	"time"

	"github.com/Crocmagnon/greenlight/internal/validator"
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

func ValidateMovie(validate *validator.Validator, movie *Movie) {
	const (
		titleMaxLength = 500
		minYear        = 1888
		minGenres      = 1
		maxGenres      = 5
	)

	validate.Check(movie.Title != "", "title", "must be provided")
	validate.Check(len(movie.Title) <= titleMaxLength, "title", "must not be more than 500 bytes long")

	validate.Check(movie.Year != 0, "year", "must be provided")
	validate.Check(movie.Year >= minYear, "year", "must be greater than 1888")
	validate.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	validate.Check(movie.Runtime != 0, "runtime", "must be provided")
	validate.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	validate.Check(movie.Genres != nil, "genres", "must be provided")
	validate.Check(len(movie.Genres) >= minGenres, "genres", "must contain at least 1 genre")
	validate.Check(len(movie.Genres) <= maxGenres, "genres", "must not contain more than 5 genres")
	validate.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}
