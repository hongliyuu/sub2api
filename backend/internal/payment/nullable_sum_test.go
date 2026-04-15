package payment

import (
	"errors"
	"testing"
)

func TestScanNullableSum(t *testing.T) {
	t.Parallel()

	t.Run("returns zero when query returns no rows", func(t *testing.T) {
		t.Parallel()

		sum, err := ScanNullableSum(func(dest any) error {
			rows := dest.(*[]nullableSumRow)
			*rows = nil
			return nil
		})
		if err != nil {
			t.Fatalf("ScanNullableSum returned error: %v", err)
		}
		if sum != 0 {
			t.Fatalf("sum = %v, want 0", sum)
		}
	})

	t.Run("returns zero when aggregate sum is null", func(t *testing.T) {
		t.Parallel()

		sum, err := ScanNullableSum(func(dest any) error {
			rows := dest.(*[]nullableSumRow)
			*rows = []nullableSumRow{{Sum: nil}}
			return nil
		})
		if err != nil {
			t.Fatalf("ScanNullableSum returned error: %v", err)
		}
		if sum != 0 {
			t.Fatalf("sum = %v, want 0", sum)
		}
	})

	t.Run("returns aggregate value when present", func(t *testing.T) {
		t.Parallel()

		expected := 12.34
		sum, err := ScanNullableSum(func(dest any) error {
			rows := dest.(*[]nullableSumRow)
			*rows = []nullableSumRow{{Sum: &expected}}
			return nil
		})
		if err != nil {
			t.Fatalf("ScanNullableSum returned error: %v", err)
		}
		if sum != expected {
			t.Fatalf("sum = %v, want %v", sum, expected)
		}
	})

	t.Run("propagates scan errors", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("scan failed")
		_, err := ScanNullableSum(func(any) error {
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
	})
}
