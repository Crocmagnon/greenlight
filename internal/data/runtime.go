package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidRuntimeFormat is returned when unmarshaling JSON.
// The expected format is "\d+ mins".
var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

// Runtime represents the duration of a movie.
type Runtime int32

// MarshalJSON implements json.Marshaler.
// intentionally using a value receiver.
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)

	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	parts := strings.Split(unquotedJSONValue, " ")

	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	const (
		base    = 10
		bitSize = 32
	)

	i, err := strconv.ParseInt(parts[0], base, bitSize)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	*r = Runtime(i)

	return nil
}
