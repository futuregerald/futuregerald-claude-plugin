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
//
// The type parameter here is load-bearing, unlike encode's: T appears in no
// argument, so it cannot be inferred and the caller must name it
// (decode[request](w, r)). It is what lets decode allocate the value it
// returns. A plain `any` would force every caller to allocate and type-assert.
func decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, err
	}

	// Decoding again into a throwaway value asks "was there anything after the
	// first one?". Its three outcomes are genuinely different and must stay
	// apart:
	//
	//	nil           a second value decoded fine, so the client sent extras.
	//	io.EOF        nothing followed -- the success case, not an error.
	//	anything else the read itself failed: the body overran the cap only
	//	              after the first value, or the connection broke. That
	//	              error has to survive, or decodeStatus can no longer see
	//	              the *http.MaxBytesError and turns a 413 into a 400.
	//
	// The trap is the nil case. fmt.Errorf("...: %w", err) with a nil err does
	// not wrap anything -- it yields an error whose text is the literal
	// %!w(<nil>) -- so it cannot be used as a single catch-all here.
	switch err := dec.Decode(&struct{}{}); {
	case err == nil:
		return v, errors.New("body must contain a single JSON object")
	case errors.Is(err, io.EOF):
		return v, nil
	default:
		return v, fmt.Errorf("read body after first JSON value: %w", err)
	}
}

// encode writes v as JSON. WriteHeader commits the status before the body is
// written, so a failure partway through encoding cannot be turned into a 500 --
// the client has already been told 201. The error is returned rather than
// swallowed so a caller that has a logger in scope can record it; the callers
// in this package have none and discard it deliberately.
//
// v is a plain any: a type parameter would be inferred from the argument and
// used exactly once, so it would compile to the same thing while implying a
// constraint that is not there.
func encode(w http.ResponseWriter, status int, v any) error {
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
