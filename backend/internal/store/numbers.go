package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Number series scopes. These match the CHECK constraint on number_series.
const (
	SeriesAdmissionNo    = "ADMISSION_NO"
	SeriesApplicationNo  = "APPLICATION_NO"
	SeriesReceiptNo      = "RECEIPT_NO"
	SeriesInvoiceNo      = "INVOICE_NO"
	SeriesCertBonafide   = "CERT_BONAFIDE"
	SeriesCertCharacter  = "CERT_CHARACTER"
	SeriesCertTransfer   = "CERT_TRANSFER"
	SeriesCertAttendance = "CERT_ATTENDANCE"
)

// AllocateNumber issues the next number in a series, inside the caller's
// transaction.
//
// This is how an admission number, a receipt number and a certificate serial
// are issued. Three properties matter, and all three come from doing it this
// way rather than with a sequence or a count:
//
//   - Gapless. The UPDATE ... RETURNING takes a row lock, so two clerks
//     collecting a fee at the same moment get consecutive receipt numbers, not
//     the same one. A Postgres sequence would not do this: sequences
//     deliberately leak numbers on rollback, and a fee book with gaps in it is
//     a fee book an auditor will ask about.
//   - Formatted the way the school already writes them, from the prefix and
//     width the office sets, so the new system continues the existing series
//     instead of starting again at 1.
//   - Inside the caller's transaction. If the receipt insert fails, the number
//     is not consumed.
func AllocateNumber(ctx context.Context, tx pgx.Tx, scope, period string) (string, error) {
	const query = `
		UPDATE number_series
		   SET next_value = next_value + 1
		 WHERE scope  = $1
		   AND period = $2
		RETURNING prefix, next_value - 1, pad_width`

	var prefix string
	var value, padWidth int

	err := tx.QueryRow(ctx, query, scope, period).Scan(&prefix, &value, &padWidth)
	if err != nil {
		if noRows(err) == ErrNotFound {
			return "", fmt.Errorf(
				"%w: no number series for %s in %s — add one in school settings",
				ErrNotFound, scope, period)
		}
		return "", fmt.Errorf("allocate %s number: %w", scope, err)
	}

	return prefix + pad(value, padWidth), nil
}

// PeekNumber reports what the next number in a series would be, without
// consuming it. Used to show the office "the next receipt will be RCP/2026/0042"
// before they commit to taking the money.
func (s *Store) PeekNumber(ctx context.Context, scope, period string) (string, error) {
	const query = `
		SELECT prefix, next_value, pad_width
		  FROM number_series
		 WHERE scope = $1 AND period = $2`

	var prefix string
	var value, padWidth int

	if err := s.pool.QueryRow(ctx, query, scope, period).Scan(&prefix, &value, &padWidth); err != nil {
		return "", noRows(err)
	}
	return prefix + pad(value, padWidth), nil
}

// EnsureSeries creates a series for a new academic session if it is missing, so
// rolling over to the next year does not need a migration.
func (s *Store) EnsureSeries(ctx context.Context, scope, period, prefix string, padWidth int) error {
	const query = `
		INSERT INTO number_series (scope, period, prefix, next_value, pad_width)
		VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (scope, period) DO NOTHING`

	_, err := s.pool.Exec(ctx, query, scope, period, prefix, padWidth)
	if err != nil {
		return fmt.Errorf("ensure number series %s/%s: %w", scope, period, err)
	}
	return nil
}

// NextNumber issues the next number in a series in a simple transaction.
// Used for public APIs where the caller is not already in a transaction.
func (s *Store) NextNumber(ctx context.Context, scope, period string) (string, error) {
	var result string
	err := s.InTx(ctx, func(tx pgx.Tx) error {
		var txErr error
		result, txErr = AllocateNumber(ctx, tx, scope, period)
		return txErr
	})
	return result, err
}

func pad(value, width int) string {
	text := fmt.Sprintf("%d", value)
	if len(text) >= width {
		return text
	}
	return strings.Repeat("0", width-len(text)) + text
}
