package pkg

import "fmt"

// ErrorKind categorizes managed errors so callers can react differently to errors vs failures.
type ErrorKind string

const (
	// KindError represents an expected domain/business error condition.
	KindError ErrorKind = "error"
	// KindFailure represents an unexpected failure that usually needs special handling.
	KindFailure ErrorKind = "failure"
)

// ChainableError captures rich error information that can be chained and recovered from.
type ChainableError interface {
	error
	Message() string
	Code() string
	Status() int
	Details() map[string]string
	Kind() ErrorKind
	Unwrap() error
}

// ErrorOption customises an error during construction.
type ErrorOption func(*baseError)

// WithCause attaches an underlying cause that can be unwrapped later.
func WithCause(err error) ErrorOption {
	return func(b *baseError) {
		b.cause = err
	}
}

// WithDetails sets multiple detail entries at once. Keys are merged with existing ones.
func WithDetails(details map[string]string) ErrorOption {
	return func(b *baseError) {
		if len(details) == 0 {
			return
		}
		if b.details == nil {
			b.details = make(map[string]string, len(details))
		}
		for k, v := range details {
			b.details[k] = v
		}
	}
}

// WithDetail sets a single detail key/value pair.
func WithDetail(key, value string) ErrorOption {
	return func(b *baseError) {
		if b.details == nil {
			b.details = make(map[string]string, 1)
		}
		b.details[key] = value
	}
}

type baseError struct {
	message string
	code    string
	status  int
	details map[string]string
	cause   error
	kind    ErrorKind
}

func newBaseError(kind ErrorKind, message, code string, status int, opts ...ErrorOption) *baseError {
	b := &baseError{
		message: message,
		code:    code,
		status:  status,
		details: make(map[string]string),
		kind:    kind,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

func (b *baseError) Message() string {
	return b.message
}

func (b *baseError) Code() string {
	return b.code
}

func (b *baseError) Status() int {
	return b.status
}

func (b *baseError) Kind() ErrorKind {
	return b.kind
}

func (b *baseError) Details() map[string]string {
	if len(b.details) == 0 {
		return map[string]string{}
	}
	clone := make(map[string]string, len(b.details))
	for k, v := range b.details {
		clone[k] = v
	}
	return clone
}

func (b *baseError) Unwrap() error {
	return b.cause
}

func (b *baseError) Error() string {
	switch {
	case b.code != "" && b.status != 0:
		return fmt.Sprintf("%s (code=%s status=%d)", b.message, b.code, b.status)
	case b.code != "":
		return fmt.Sprintf("%s (code=%s)", b.message, b.code)
	case b.status != 0:
		return fmt.Sprintf("%s (status=%d)", b.message, b.status)
	default:
		return b.message
	}
}

func (b *baseError) addDetail(key, value string) {
	if b.details == nil {
		b.details = make(map[string]string)
	}
	b.details[key] = value
}

func (b *baseError) mergeDetails(details map[string]string) {
	if len(details) == 0 {
		return
	}
	if b.details == nil {
		b.details = make(map[string]string, len(details))
	}
	for k, v := range details {
		b.details[k] = v
	}
}

// AppError represents an expected error scenario.
type AppError struct {
	*baseError
}

// NewError builds a new AppError with optional configuration.
func NewError(message, code string, status int, opts ...ErrorOption) *AppError {
	return &AppError{newBaseError(KindError, message, code, status, opts...)}
}

// AddDetail augments the error with additional contextual data.
func (e *AppError) AddDetail(key, value string) *AppError {
	if e == nil {
		return nil
	}
	e.addDetail(key, value)
	return e
}

// AddDetails augments the error with multiple contextual entries.
func (e *AppError) AddDetails(details map[string]string) *AppError {
	if e == nil {
		return nil
	}
	e.mergeDetails(details)
	return e
}

// AppFailure represents an unexpected failure scenario.
type AppFailure struct {
	*baseError
}

// NewFailure builds a new AppFailure with optional configuration.
func NewFailure(message, code string, status int, opts ...ErrorOption) *AppFailure {
	return &AppFailure{newBaseError(KindFailure, message, code, status, opts...)}
}

// AddDetail augments the failure with additional contextual data.
func (f *AppFailure) AddDetail(key, value string) *AppFailure {
	if f == nil {
		return nil
	}
	f.addDetail(key, value)
	return f
}

// AddDetails augments the failure with multiple contextual entries.
func (f *AppFailure) AddDetails(details map[string]string) *AppFailure {
	if f == nil {
		return nil
	}
	f.mergeDetails(details)
	return f
}

// Result wraps a value together with a ChainableError to enable promise-like chaining.
type Result[T any] struct {
	value T
	err   ChainableError
}

// Success creates a successful Result.
func Success[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// FailureResult creates an errored Result from a ChainableError.
func FailureResult[T any](err ChainableError) Result[T] {
	return Result[T]{err: err}
}

// From converts a standard Go error into a Result, wrapping non-chainable errors as failures.
func From[T any](value T, err error) Result[T] {
	if err == nil {
		return Success(value)
	}
	if chain, ok := err.(ChainableError); ok {
		return FailureResult[T](chain)
	}
	failure := NewFailure(err.Error(), "unexpected_failure", 500, WithCause(err))
	return FailureResult[T](failure)
}

// Then executes the callback when the current Result has no error.
func (r Result[T]) Then(fn func(T) Result[T]) Result[T] {
	if r.err != nil {
		return FailureResult[T](r.err)
	}
	return fn(r.value)
}

// Chain allows mapping a Result into another Result with a different value type.
func Chain[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.err != nil {
		return FailureResult[U](r.err)
	}
	return fn(r.value)
}

// OnError executes the callback when the Result carries an AppError.
func (r Result[T]) OnError(fn func(*AppError) Result[T]) Result[T] {
	if r.err == nil {
		return r
	}
	if err, ok := r.err.(*AppError); ok {
		return fn(err)
	}
	return r
}

// OnFail executes the callback when the Result carries an AppFailure.
func (r Result[T]) OnFail(fn func(*AppFailure) Result[T]) Result[T] {
	if r.err == nil {
		return r
	}
	if failure, ok := r.err.(*AppFailure); ok {
		return fn(failure)
	}
	return r
}

// OnAnyError executes regardless of the error kind.
func (r Result[T]) OnAnyError(fn func(ChainableError) Result[T]) Result[T] {
	if r.err == nil {
		return r
	}
	return fn(r.err)
}

// Value returns the inner value and any ChainableError encountered.
func (r Result[T]) Value() (T, ChainableError) {
	return r.value, r.err
}

// Error exposes the stored ChainableError for inspection.
func (r Result[T]) Error() ChainableError {
	return r.err
}

// IsOK reports whether the Result is free of errors.
func (r Result[T]) IsOK() bool {
	return r.err == nil
}
