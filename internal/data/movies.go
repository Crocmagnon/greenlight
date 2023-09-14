package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Crocmagnon/greenlight/internal/validator"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const timeout = 3 * time.Second

// Movie holds information about a single movie.
// It represents the entity as stored in the DB.
type Movie struct {
	ID        int64          `db:"id"         json:"id"`
	CreatedAt time.Time      `db:"created_at" json:"-"`
	Title     string         `db:"title"      json:"title"`
	Year      int32          `db:"year"       json:"year,omitempty"`
	Runtime   Runtime        `db:"runtime"    json:"runtime,omitempty"`
	Genres    pq.StringArray `db:"genres"     json:"genres,omitempty"`
	Version   int32          `db:"version"    json:"version"`
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
	DB *sqlx.DB
}

// Insert inserts a movie in the database.
// Movie.CreatedAt and Movie.Version are set on the passed movie.
func (m MovieModel) Insert(movie *Movie) error {
	query := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`
	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := m.DB.GetContext(ctx, movie, query, args...)
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := m.DB.GetContext(ctx, &movie, query, id)

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
		WHERE id = $5 AND version = $6
		RETURNING version`
	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := m.DB.GetContext(ctx, &movie.Version, query, args...)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrEditConflict
	case err != nil:
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	res, err := m.DB.ExecContext(ctx, query, id)
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

// GetAll returns a filtered list of movies from the DB.
func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
	query := fmt.Sprintf(`SELECT count(*) OVER() AS total_records, id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (genres @> $2 OR $2 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())
	args := []any{title, pq.Array(genres), filters.limit(), filters.offset()}

	const dbTimeout = 3 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := m.DB.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, fmt.Errorf("listing movies: %w", err)
	}

	defer rows.Close() //nolint:errcheck // we wouldn't do anything with this err

	totalRecords := 0
	movies := []*Movie{}

	for rows.Next() {
		var movie struct {
			TotalRecords int `db:"total_records"`
			Movie
		}

		err = rows.StructScan(&movie)
		if err != nil {
			return nil, Metadata{}, fmt.Errorf("scanning movie: %w", err)
		}

		totalRecords = movie.TotalRecords
		movies = append(movies, &movie.Movie)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, fmt.Errorf("iterating over rows: %w", err)
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return movies, metadata, nil
}
