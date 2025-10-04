package pkg

import (
	"errors"
	"fmt"
	"testing"
)

func TestNewErrorAttributes(t *testing.T) {
	cause := errors.New("root cause")
	err := NewError("something happened", "E001", 400,
		WithCause(cause),
		WithDetail("user", "123"),
		WithDetails(map[string]string{"trace": "abc"}),
	)

	if err.Message() != "something happened" {
		t.Fatalf("unexpected message: %q", err.Message())
	}
	if err.Code() != "E001" {
		t.Fatalf("unexpected code: %q", err.Code())
	}
	if err.Status() != 400 {
		t.Fatalf("unexpected status: %d", err.Status())
	}
	if err.Kind() != KindError {
		t.Fatalf("unexpected kind: %v", err.Kind())
	}
	if !errors.Is(err, cause) {
		t.Fatalf("expected to unwrap to cause")
	}

	details := err.Details()
	if details["user"] != "123" || details["trace"] != "abc" {
		t.Fatalf("unexpected details: %+v", details)
	}

	details["user"] = "mutated"
	if err.Details()["user"] != "123" {
		t.Fatalf("expected details map to be copied on access")
	}
}

func TestNewFailureAttributes(t *testing.T) {
	failure := NewFailure("boom", "F001", 500, WithDetail("host", "srv-1"))

	if failure.Kind() != KindFailure {
		t.Fatalf("unexpected kind: %v", failure.Kind())
	}
	if failure.Status() != 500 {
		t.Fatalf("unexpected status: %d", failure.Status())
	}
	if failure.Details()["host"] != "srv-1" {
		t.Fatalf("detail not stored")
	}
}

func TestResultThenSuccess(t *testing.T) {
	res := Success(2).
		Then(func(v int) Result[int] {
			return Success(v * 2)
		})

	val, err := res.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 4 {
		t.Fatalf("unexpected value: %d", val)
	}
}

func TestResultThenPropagatesError(t *testing.T) {
	chainErr := NewError("invalid", "E002", 422)
	res := FailureResult[int](chainErr).
		Then(func(v int) Result[int] {
			t.Fatalf("then should not execute on errors")
			return Success(v)
		})

	if res.Error() != chainErr {
		t.Fatalf("expected original error to propagate")
	}
}

func TestResultChainMapsType(t *testing.T) {
	res := Chain(Success(10), func(v int) Result[string] {
		return Success(fmt.Sprintf("%d", v))
	})

	val, err := res.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "10" {
		t.Fatalf("unexpected value: %q", val)
	}
}

func TestOnErrorHandlesAppError(t *testing.T) {
	chainErr := NewError("bad input", "E123", 400)
	called := false
	res := FailureResult[int](chainErr).
		OnError(func(appErr *AppError) Result[int] {
			called = true
			if appErr.Code() != "E123" {
				t.Fatalf("unexpected code: %s", appErr.Code())
			}
			return Success(99)
		})

	val, err := res.Value()
	if !called {
		t.Fatalf("expected OnError callback to be invoked")
	}
	if err != nil {
		t.Fatalf("unexpected error after recovery: %v", err)
	}
	if val != 99 {
		t.Fatalf("unexpected value after recovery: %d", val)
	}
}

func TestOnFailHandlesAppFailure(t *testing.T) {
	failure := NewFailure("boom", "F999", 500)
	called := false
	res := FailureResult[int](failure).
		OnFail(func(appFail *AppFailure) Result[int] {
			called = true
			if appFail.Code() != "F999" {
				t.Fatalf("unexpected code: %s", appFail.Code())
			}
			return Success(7)
		})

	val, err := res.Value()
	if !called {
		t.Fatalf("expected OnFail callback to be invoked")
	}
	if err != nil {
		t.Fatalf("unexpected error after recovery: %v", err)
	}
	if val != 7 {
		t.Fatalf("unexpected value after recovery: %d", val)
	}
}

func TestOnAnyError(t *testing.T) {
	failure := NewFailure("boom", "F000", 500)
	called := false
	res := FailureResult[int](failure).
		OnAnyError(func(err ChainableError) Result[int] {
			called = true
			if err.Kind() != KindFailure {
				t.Fatalf("unexpected kind: %v", err.Kind())
			}
			return Success(1)
		})

	val, err := res.Value()
	if !called {
		t.Fatalf("expected OnAnyError callback to be invoked")
	}
	if err != nil {
		t.Fatalf("unexpected error after recovery: %v", err)
	}
	if val != 1 {
		t.Fatalf("unexpected value after recovery: %d", val)
	}
}

func TestFromWrapsStandardErrorAsFailure(t *testing.T) {
	inputErr := errors.New("plain error")
	res := From(0, inputErr)

	if res.err == nil {
		t.Fatalf("expected failure")
	}
	failure, ok := res.err.(*AppFailure)
	if !ok {
		t.Fatalf("expected failure to be wrapped as AppFailure")
	}
	if failure.Unwrap() != inputErr {
		t.Fatalf("expected original error as cause")
	}
}

func TestFromSuccess(t *testing.T) {
	res := From("value", nil)
	if !res.IsOK() {
		t.Fatalf("expected result to be OK")
	}
	val, err := res.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "value" {
		t.Fatalf("unexpected value: %s", val)
	}
}
