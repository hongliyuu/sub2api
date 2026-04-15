package payment

type nullableSumRow struct {
	Sum *float64 `json:"sum"`
}

// ScanNullableSum normalizes aggregate SUM results so empty datasets behave like 0.
func ScanNullableSum(scan func(dest any) error) (float64, error) {
	var rows []nullableSumRow
	if err := scan(&rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 || rows[0].Sum == nil {
		return 0, nil
	}
	return *rows[0].Sum, nil
}
