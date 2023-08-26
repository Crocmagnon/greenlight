package data

import "github.com/Crocmagnon/greenlight/internal/validator"

// Filters are used to filter and sort pages of data.
// They should be validated using ValidateFilters.
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string
}

// ValidateFilters validates filters.
// The passed validator will contain all detected errors.
// The caller is expected to call [validator.Validator.Valid]
// after this method.
func ValidateFilters(validate *validator.Validator, filters Filters) {
	const (
		maxPage     = 10_000_000
		maxPageSize = 100
	)

	validate.Check(filters.Page > 0, "page", "must be greater than zero")
	validate.Check(filters.Page <= maxPage, "page", "must be a maximum of 10 million")
	validate.Check(filters.PageSize > 0, "page_size", "must be greater than zero")
	validate.Check(filters.PageSize <= maxPageSize, "page_size", "must be a maximum of 100")
	validate.Check(validator.PermittedValue(filters.Sort, filters.SortSafelist...), "sort", "invalid sort value")
}
