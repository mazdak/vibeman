// Package types provides common type definitions
package types

import (
	"encoding/json"
	"vibeman/internal/errors"
)

// Result represents a generic result that can contain either data or an error
type Result[T any] struct {
	data T
	err  error
}

// NewResult creates a new Result with data
func NewResult[T any](data T) Result[T] {
	return Result[T]{data: data}
}

// NewErrorResult creates a new Result with an error
func NewErrorResult[T any](err error) Result[T] {
	var zero T
	return Result[T]{data: zero, err: err}
}

// IsError returns true if the result contains an error
func (r Result[T]) IsError() bool {
	return r.err != nil
}

// IsSuccess returns true if the result contains data (no error)
func (r Result[T]) IsSuccess() bool {
	return r.err == nil
}

// Unwrap returns the data and error
func (r Result[T]) Unwrap() (T, error) {
	return r.data, r.err
}

// Data returns the data, panics if there's an error
func (r Result[T]) Data() T {
	if r.err != nil {
		panic("called Data() on error result: " + r.err.Error())
	}
	return r.data
}

// Error returns the error
func (r Result[T]) Error() error {
	return r.err
}

// Map transforms the result data if successful
func (r Result[T]) Map(fn func(T) T) Result[T] {
	if r.err != nil {
		return r
	}
	return NewResult(fn(r.data))
}

// FlatMap transforms the result, allowing error handling
func (r Result[T]) FlatMap(fn func(T) Result[T]) Result[T] {
	if r.err != nil {
		return r
	}
	return fn(r.data)
}

// OrElse returns the result if successful, otherwise returns the alternative
func (r Result[T]) OrElse(alternative T) T {
	if r.err != nil {
		return alternative
	}
	return r.data
}

// OrElseGet returns the result if successful, otherwise calls the function
func (r Result[T]) OrElseGet(fn func() T) T {
	if r.err != nil {
		return fn()
	}
	return r.data
}

// MarshalJSON implements json.Marshaler
func (r Result[T]) MarshalJSON() ([]byte, error) {
	if r.err != nil {
		// Marshal error as JSON
		if ve, ok := r.err.(*errors.VibemanError); ok {
			return json.Marshal(map[string]interface{}{
				"error": ve,
			})
		}
		return json.Marshal(map[string]interface{}{
			"error": map[string]string{
				"message": r.err.Error(),
			},
		})
	}
	return json.Marshal(map[string]interface{}{
		"data": r.data,
	})
}

// Option represents an optional value
type Option[T any] struct {
	value   T
	present bool
}

// Some creates an Option with a value
func Some[T any](value T) Option[T] {
	return Option[T]{value: value, present: true}
}

// None creates an empty Option
func None[T any]() Option[T] {
	var zero T
	return Option[T]{value: zero, present: false}
}

// IsPresent returns true if the option contains a value
func (o Option[T]) IsPresent() bool {
	return o.present
}

// IsEmpty returns true if the option is empty
func (o Option[T]) IsEmpty() bool {
	return !o.present
}

// Get returns the value, panics if empty
func (o Option[T]) Get() T {
	if !o.present {
		panic("called Get() on empty Option")
	}
	return o.value
}

// OrElse returns the value if present, otherwise returns the alternative
func (o Option[T]) OrElse(alternative T) T {
	if o.present {
		return o.value
	}
	return alternative
}

// Map transforms the option value if present
func (o Option[T]) Map(fn func(T) T) Option[T] {
	if !o.present {
		return o
	}
	return Some(fn(o.value))
}
