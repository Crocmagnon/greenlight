package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Crocmagnon/greenlight/internal/validator"
	"github.com/julienschmidt/httprouter"
)

const (
	maxBytes = 1_048_576
)

// Errors used during validation and returned to the consumer.
var (
	ErrInvalidID         = errors.New("invalid id parameter")
	ErrMalformedJSON     = errors.New("body contains malformed JSON")
	ErrIncorrectJSONType = errors.New("body contains incorrect JSON type")
	ErrEmptyBody         = errors.New("body is empty")
	ErrUnknownKey        = errors.New("body contains unknown key")
	ErrBodyTooLarge      = errors.New("body is too large")
	ErrMultipleJSON      = errors.New("body must only contain a single JSON value")
)

func (*application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	const (
		base    = 10
		bitSize = 64
	)

	id, err := strconv.ParseInt(params.ByName("id"), base, bitSize)
	if err != nil || id < 1 {
		return 0, ErrInvalidID
	}

	return id, nil
}

type envelope map[string]any

func (*application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	resp, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encoding to json: %w", err)
	}

	resp = append(resp, '\n')

	for k, v := range headers {
		w.Header()[k] = v
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(resp) //nolint:errcheck

	return nil
}

func (*application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		return wrapError(err)
	}

	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return ErrMultipleJSON
	}

	return nil
}

func wrapError(err error) error {
	const unknownFieldPrefix = "json: unknown field "

	var (
		syntaxError           *json.SyntaxError
		unmarshalTypeError    *json.UnmarshalTypeError
		invalidUnmarshalError *json.InvalidUnmarshalError
		maxBytesError         *http.MaxBytesError
	)

	switch {
	case errors.As(err, &syntaxError):
		return fmt.Errorf("%w (at character %d)", ErrMalformedJSON, syntaxError.Offset)

	case errors.Is(err, io.ErrUnexpectedEOF):
		return ErrMalformedJSON

	case errors.As(err, &unmarshalTypeError):
		if unmarshalTypeError.Field != "" {
			return fmt.Errorf("%w for field %q", ErrIncorrectJSONType, unmarshalTypeError.Field)
		}

		return fmt.Errorf("%w (at character %d)", ErrIncorrectJSONType, unmarshalTypeError.Offset)

	case errors.Is(err, io.EOF):
		return ErrEmptyBody

	case strings.HasPrefix(err.Error(), unknownFieldPrefix):
		fieldName := strings.TrimPrefix(err.Error(), unknownFieldPrefix)
		return fmt.Errorf("%w %s", ErrUnknownKey, fieldName)

	case errors.As(err, &maxBytesError):
		return fmt.Errorf("%w, max size %d bytes", ErrBodyTooLarge, maxBytes)

	case errors.As(err, &invalidUnmarshalError):
		panic(err)

	default:
		return fmt.Errorf("unhandled error: %w", err)
	}
}

func (*application) readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	return s
}

func (*application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	return strings.Split(s, ",")
}

func (*application) readInt(qs url.Values, key string, defaultValue int, validate *validator.Validator) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		validate.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}
