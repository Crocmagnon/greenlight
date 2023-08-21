package data

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Crocmagnon/greenlight/internal/validator"
	"github.com/lib/pq"
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

// MovieModel implements methods to query the database.
type MovieModel struct {
	DB *sql.DB
}

// Insert inserts a movie in the database.
// Movie.CreatedAt and Movie.Version are set on the passed movie.
func (m MovieModel) Insert(movie *Movie) error {
	query := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`
	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	//nolint:execinquery // False positive
	err := m.DB.QueryRow(query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
	if err != nil {
		return fmt.Errorf("inserting movie in DB: %w", err)
	}

	return nil
}

// Get returns the Movie with the given id from the DB,
// or an error if it couldn't be found.
func (m MovieModel) Get(id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE id=$1`

	var movie Movie

	err := m.DB.QueryRow(query, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
	)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrRecordNotFound
	case err != nil:
		return nil, fmt.Errorf("querying movie: %w", err)
	}

	return &movie, nil
}

// Update updates a movie in the DB.
// Movie.Version is set on the passed movie.
func (m MovieModel) Update(movie *Movie) error {
	query := `
		UPDATE movies
		SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
		WHERE id = $5
		RETURNING version`
	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
	}

	//nolint:execinquery // False positive
	err := m.DB.QueryRow(query, args...).Scan(&movie.Version)
	if err != nil {
		return fmt.Errorf("inserting movie in DB: %w", err)
	}

	return nil
}

// Delete deletes a movie from the DB.
func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `DELETE FROM movies WHERE id=$1`

	res, err := m.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("deleting movie from db: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("counting affected rows: %w", err)
	}

	if rows == 0 {
		return ErrRecordNotFound
	}

	return nil
}
