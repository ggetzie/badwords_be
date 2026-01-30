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
	"time"

	"github.com/ggetzie/badwords_be/internal/validator"
	"github.com/julienschmidt/httprouter"
)

type envelope map[string]any

func (app *application) writeJSON(
	w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	js = append(js, '\n')
	for key, value := range headers {
		w.Header()[key] = value
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}

func (app *application) readJSON(
	w http.ResponseWriter, r *http.Request, dst any) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type at position %d", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown field %s", fieldName)
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}
	return nil
}

func (app *application) readString(qs url.Values, key, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	return s
}

func (app *application) readDateTime(qs url.Values, key string, defaultValue time.Time, v *validator.Validator) time.Time {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		v.AddError(key, "must be a valid datetime")
		return time.Time{}
	}
	return t
}

func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}
	return i
}

func (app *application) readIntList(qs url.Values, key string, defaultValue []int, v *validator.Validator, minValue, maxValue int) []int {
	var nums []int
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	parts := strings.Split(s, ",")
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			v.AddError(key, "must be a list of comma-separated integers")
			return defaultValue
		}
		if num < minValue || num > maxValue {
			v.AddError(key, fmt.Sprintf("must be between %d and %d", minValue, maxValue))
			return defaultValue
		}
		nums = append(nums, num)
	}
	return nums
}

func (app *application) readStringParam(r *http.Request, key string) (string, error) {
	params := httprouter.ParamsFromContext(r.Context())
	value := params.ByName(key)
	if value == "" {
		return "", fmt.Errorf("missing %s parameter", key)
	}
	return value, nil
}

func (app *application) background(fn func()) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Sprintf("panic: %v", err))
			}
		}()
		fn()
	}()
}
