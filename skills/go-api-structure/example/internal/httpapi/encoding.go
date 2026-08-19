package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// maxBodyBytes caps every decoded request body. Without it a single client can
// make the server allocate until it dies -- the decoder will happily read
// gigabytes.
const maxBodyBytes = 1 << 20 // 1 MiB

// decode reads exactly one JSON value of type T from the request.
//
// The three guards are each a real failure seen in production: an unbounded
// body (resource exhaustion), an unknown field (a client sending `admin:true`
// and believing it worked), and trailing data (two objects concatenated, of
// which only the first is honoured).
//
// Validation is deliberately NOT done here -- see the note below on why the
// Validator interface is not used with handler-local request types.
func decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, err
	}
	// A second successful decode means the client sent more than one value.
	// io.EOF here is the success case, not an error.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return v, errors.New("body must contain a single JSON object")
	}
	return v, nil
}

// encode writes v as JSON. Status must be written before the body, and after
// WriteHeader nothing can change the status -- so an encoding failure here can
// only be logged, never converted into a 500.
func encode[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	return nil
}

// decodeStatus maps a decode failure to the status the client deserves.
// An oversized body is 413, not 400: the request was well-formed, just too big.
func decodeStatus(err error) (int, string) {
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return http.StatusRequestEntityTooLarge, "request body too large"
	}
	return http.StatusBadRequest, "malformed body"
}
