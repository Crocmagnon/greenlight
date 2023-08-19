package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

const (
	maxBytes = 1_048_576
)

var (
	ErrInvalidID         = errors.New("invalid id parameter")
	ErrMalformedJSON     = errors.New("body contains malformed JSON")
	ErrIncorrectJSONType = errors.New("body contains incorrect JSON type")
	ErrEmptyBody         = errors.New("body is empty")
	ErrUnknownKey        = errors.New("body contains unknown key")
	ErrBodyTooLarge      = errors.New("body is too large")
	ErrMultipleJSON      = errors.New("body must only contain a single JSON value")
)

func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, ErrInvalidID
	}

	return id, nil
}

type envelope map[string]any

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
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

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
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
