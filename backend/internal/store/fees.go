package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Invoice is the safe ledger view returned to a family or the accounts office.
// Amounts are always paise so that neither the API nor the browser introduces
// a rounding error into the school ledger.
type Invoice struct {
	ID               string `json:"id"`
	InvoiceNo        string `json:"invoiceNo"`
	StudentID        string `json:"studentId"`
	AdmissionNo      string `json:"admissionNo"`
	StudentName      string `json:"studentName"`
	InstallmentNo    int    `json:"installmentNo"`
	DueDate          string `json:"dueDate"`
	GrossPaise       int64  `json:"grossPaise"`
	ConcessionPaise  int64  `json:"concessionPaise"`
	LateFeePaise     int64  `json:"lateFeePaise"`
	PaidPaise        int64  `json:"paidPaise"`
	OutstandingPaise int64  `json:"outstandingPaise"`
	Status           string `json:"status"`
}

// Receipt is the safe receipt-book view. Cancelled receipts intentionally stay
// visible: accounting records are cancelled, never silently erased.
type Receipt struct {
	ID          string     `json:"id"`
	ReceiptNo   string     `json:"receiptNo"`
	StudentID   string     `json:"studentId"`
	AdmissionNo string     `json:"admissionNo"`
	StudentName string     `json:"studentName"`
	AmountPaise int64      `json:"amountPaise"`
	Mode        string     `json:"mode"`
	Narration   string     `json:"narration"`
	ReceivedAt  time.Time  `json:"receivedAt"`
	CancelledAt *time.Time `json:"cancelledAt,omitempty"`
}

// ReceiptAllocationInput assigns part of one receipt to one outstanding
// invoice. The store locks each invoice before it checks this value.
type ReceiptAllocationInput struct {
	InvoiceID   string
	AmountPaise int64
}

// CollectReceiptInput is the complete office-counter transaction. A receipt
// cannot exist without allocations, which prevents unallocated cash from
// entering the ledger.
type CollectReceiptInput struct {
	StudentID   string
	SessionID   string
	SessionName string
	Mode        string
	ChequeNo    string
	ChequeDate  string
	BankName    string
	Narration   string
	ReceivedBy  string
	Allocations []ReceiptAllocationInput
}

type InvoiceItemInput struct {
	FeeHeadID       string
	AmountPaise     int64
	ConcessionPaise int64
}

type CreateInvoiceInput struct {
	StudentID     string
	SessionID     string
	SessionName   string
	InstallmentNo int
	DueDate       string
	Items         []InvoiceItemInput
}

// CreateInvoice issues one auditable student installment. Fee-plan generation
// can call this same primitive later; manual issuance lets accounts bill a
// newly admitted learner before every class plan has been finalised.
func (s *Store) CreateInvoice(ctx context.Context, input CreateInvoiceInput) (Invoice, error) {
	if len(input.Items) == 0 || len(input.Items) > 50 || input.InstallmentNo < 1 || input.InstallmentNo > 12 {
		return Invoice{}, fmt.Errorf("create invoice: %w", ErrConflict)
	}
	var result Invoice
	err := s.InTx(ctx, func(tx pgx.Tx) error {
		const enrolled = `
			SELECT EXISTS (SELECT 1 FROM enrollments
			 WHERE student_id = $1::uuid AND session_id = $2::uuid AND status = 'ENROLLED')`
		var isEnrolled bool
		if err := tx.QueryRow(ctx, enrolled, input.StudentID, input.SessionID).Scan(&isEnrolled); err != nil {
			return fmt.Errorf("check invoice enrollment: %w", err)
		}
		if !isEnrolled {
			return fmt.Errorf("create invoice: student is not enrolled for this session: %w", ErrConflict)
		}

		seenHeads := make(map[string]bool, len(input.Items))
		var gross, concession int64
		for _, item := range input.Items {
			if item.AmountPaise <= 0 || item.ConcessionPaise < 0 || item.ConcessionPaise > item.AmountPaise || seenHeads[item.FeeHeadID] || gross > (1<<62)-item.AmountPaise {
				return fmt.Errorf("create invoice: invalid items: %w", ErrConflict)
			}
			seenHeads[item.FeeHeadID] = true
			const validHead = `SELECT EXISTS (SELECT 1 FROM fee_heads WHERE id = $1::uuid)`
			var exists bool
			if err := tx.QueryRow(ctx, validHead, item.FeeHeadID).Scan(&exists); err != nil {
				return fmt.Errorf("check fee head: %w", err)
			}
			if !exists || concession > (1<<62)-item.ConcessionPaise {
				return fmt.Errorf("create invoice: invalid fee head: %w", ErrConflict)
			}
			gross += item.AmountPaise
			concession += item.ConcessionPaise
		}
		invoiceNo, err := AllocateNumber(ctx, tx, SeriesInvoiceNo, input.SessionName)
		if err != nil {
			return err
		}
		const insertInvoice = `
			INSERT INTO invoices (invoice_no, student_id, session_id, installment_no, due_date, gross_paise, concession_paise)
			VALUES ($1, $2::uuid, $3::uuid, $4, $5::date, $6, $7)
			RETURNING id::text`
		if err := tx.QueryRow(ctx, insertInvoice, invoiceNo, input.StudentID, input.SessionID, input.InstallmentNo, input.DueDate, gross, concession).Scan(&result.ID); err != nil {
			if _, unique := isUniqueViolation(err); unique {
				return fmt.Errorf("create invoice: %w", ErrConflict)
			}
			return fmt.Errorf("insert invoice: %w", err)
		}
		for _, item := range input.Items {
			const insertItem = `
				INSERT INTO invoice_items (invoice_id, fee_head_id, amount_paise, concession_paise)
				VALUES ($1::uuid, $2::uuid, $3, $4)`
			if _, err := tx.Exec(ctx, insertItem, result.ID, item.FeeHeadID, item.AmountPaise, item.ConcessionPaise); err != nil {
				return fmt.Errorf("insert invoice item: %w", err)
			}
		}
		const student = `SELECT admission_no, full_name_en FROM students WHERE id = $1::uuid`
		if err := tx.QueryRow(ctx, student, input.StudentID).Scan(&result.AdmissionNo, &result.StudentName); err != nil {
			return fmt.Errorf("read invoice student: %w", err)
		}
		result.InvoiceNo, result.StudentID, result.InstallmentNo, result.DueDate = invoiceNo, input.StudentID, input.InstallmentNo, input.DueDate
		result.GrossPaise, result.ConcessionPaise, result.OutstandingPaise, result.Status = gross, concession, gross-concession, "DUE"
		return nil
	})
	if err != nil {
		return Invoice{}, err
	}
	return result, nil
}

type CollectionDay struct {
	Date         string `json:"date"`
	ReceiptCount int    `json:"receiptCount"`
	AmountPaise  int64  `json:"amountPaise"`
}

// ListCollectionDays is the cashier-safe financial report. Cancelled receipts
// are excluded from cash totals but remain visible in the receipt register.
func (s *Store) ListCollectionDays(ctx context.Context, sessionID string, limit int) ([]CollectionDay, error) {
	const query = `
		SELECT to_char(received_at::date, 'YYYY-MM-DD'), count(*)::int, sum(amount_paise)::bigint
		  FROM receipts
		 WHERE session_id = $1::uuid AND cancelled_at IS NULL
		 GROUP BY received_at::date
		 ORDER BY received_at::date DESC
		 LIMIT $2`
	rows, err := s.pool.Query(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list fee collection: %w", err)
	}
	defer rows.Close()
	items := []CollectionDay{}
	for rows.Next() {
		var item CollectionDay
		if err := rows.Scan(&item.Date, &item.ReceiptCount, &item.AmountPaise); err != nil {
			return nil, fmt.Errorf("scan fee collection: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fee collection: %w", err)
	}
	return items, nil
}

// ListInvoicesAll returns all invoices in a session, optionally narrowed to a
// particular learner. Only the handler calls this for accounts staff.
func (s *Store) ListInvoicesAll(ctx context.Context, sessionID, studentID string, limit, offset int) ([]Invoice, error) {
	const query = `
		SELECT i.id::text, i.invoice_no, i.student_id::text, st.admission_no,
		       st.full_name_en, i.installment_no, to_char(i.due_date, 'YYYY-MM-DD'),
		       i.gross_paise, i.concession_paise, i.late_fee_paise, i.paid_paise,
		       (i.gross_paise - i.concession_paise + i.late_fee_paise - i.paid_paise)::bigint,
		       i.status
		  FROM invoices i
		  JOIN students st ON st.id = i.student_id
		 WHERE i.session_id = $1::uuid
		   AND (NULLIF($2, '') IS NULL OR i.student_id = NULLIF($2, '')::uuid)
		 ORDER BY i.due_date DESC, i.invoice_no DESC
		 LIMIT $3 OFFSET $4`
	return s.listInvoices(ctx, query, sessionID, studentID, limit, offset)
}

// ListInvoicesForStudent scopes the ledger to one student, regardless of what
// identifiers an authenticated student tries to place in a query string.
func (s *Store) ListInvoicesForStudent(ctx context.Context, sessionID, studentID string, limit, offset int) ([]Invoice, error) {
	const query = `
		SELECT i.id::text, i.invoice_no, i.student_id::text, st.admission_no,
		       st.full_name_en, i.installment_no, to_char(i.due_date, 'YYYY-MM-DD'),
		       i.gross_paise, i.concession_paise, i.late_fee_paise, i.paid_paise,
		       (i.gross_paise - i.concession_paise + i.late_fee_paise - i.paid_paise)::bigint,
		       i.status
		  FROM invoices i
		  JOIN students st ON st.id = i.student_id
		 WHERE i.session_id = $1::uuid AND i.student_id = $2::uuid
		 ORDER BY i.due_date DESC, i.invoice_no DESC
		 LIMIT $3 OFFSET $4`
	return s.listInvoices(ctx, query, sessionID, studentID, limit, offset)
}

// ListInvoicesForGuardian scopes records through student_guardians. A parent
// account has no route to unrelated learners' fees.
func (s *Store) ListInvoicesForGuardian(ctx context.Context, sessionID, guardianID string, limit, offset int) ([]Invoice, error) {
	const query = `
		SELECT i.id::text, i.invoice_no, i.student_id::text, st.admission_no,
		       st.full_name_en, i.installment_no, to_char(i.due_date, 'YYYY-MM-DD'),
		       i.gross_paise, i.concession_paise, i.late_fee_paise, i.paid_paise,
		       (i.gross_paise - i.concession_paise + i.late_fee_paise - i.paid_paise)::bigint,
		       i.status
		  FROM invoices i
		  JOIN students st ON st.id = i.student_id
		  JOIN student_guardians sg ON sg.student_id = st.id
		 WHERE i.session_id = $1::uuid AND sg.guardian_id = $2::uuid
		 ORDER BY i.due_date DESC, i.invoice_no DESC
		 LIMIT $3 OFFSET $4`
	return s.listInvoices(ctx, query, sessionID, guardianID, limit, offset)
}

func (s *Store) listInvoices(ctx context.Context, query string, firstID, secondID string, limit, offset int) ([]Invoice, error) {
	rows, err := s.pool.Query(ctx, query, firstID, secondID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	items := []Invoice{}
	for rows.Next() {
		var item Invoice
		if err := rows.Scan(
			&item.ID, &item.InvoiceNo, &item.StudentID, &item.AdmissionNo,
			&item.StudentName, &item.InstallmentNo, &item.DueDate,
			&item.GrossPaise, &item.ConcessionPaise, &item.LateFeePaise, &item.PaidPaise,
			&item.OutstandingPaise, &item.Status,
		); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoices: %w", err)
	}
	return items, nil
}

// ListReceiptsAll returns every receipt in a session for the accounts office.
func (s *Store) ListReceiptsAll(ctx context.Context, sessionID, studentID string, limit, offset int) ([]Receipt, error) {
	const query = `
		SELECT r.id::text, r.receipt_no, r.student_id::text, st.admission_no,
		       st.full_name_en, r.amount_paise, r.mode, COALESCE(r.narration, ''),
		       r.received_at, r.cancelled_at
		  FROM receipts r
		  JOIN students st ON st.id = r.student_id
		 WHERE r.session_id = $1::uuid
		   AND (NULLIF($2, '') IS NULL OR r.student_id = NULLIF($2, '')::uuid)
		 ORDER BY r.received_at DESC, r.receipt_no DESC
		 LIMIT $3 OFFSET $4`
	return s.listReceipts(ctx, query, sessionID, studentID, limit, offset)
}

func (s *Store) ListReceiptsForStudent(ctx context.Context, sessionID, studentID string, limit, offset int) ([]Receipt, error) {
	const query = `
		SELECT r.id::text, r.receipt_no, r.student_id::text, st.admission_no,
		       st.full_name_en, r.amount_paise, r.mode, COALESCE(r.narration, ''),
		       r.received_at, r.cancelled_at
		  FROM receipts r
		  JOIN students st ON st.id = r.student_id
		 WHERE r.session_id = $1::uuid AND r.student_id = $2::uuid
		 ORDER BY r.received_at DESC, r.receipt_no DESC
		 LIMIT $3 OFFSET $4`
	return s.listReceipts(ctx, query, sessionID, studentID, limit, offset)
}

func (s *Store) ListReceiptsForGuardian(ctx context.Context, sessionID, guardianID string, limit, offset int) ([]Receipt, error) {
	const query = `
		SELECT r.id::text, r.receipt_no, r.student_id::text, st.admission_no,
		       st.full_name_en, r.amount_paise, r.mode, COALESCE(r.narration, ''),
		       r.received_at, r.cancelled_at
		  FROM receipts r
		  JOIN students st ON st.id = r.student_id
		  JOIN student_guardians sg ON sg.student_id = st.id
		 WHERE r.session_id = $1::uuid AND sg.guardian_id = $2::uuid
		 ORDER BY r.received_at DESC, r.receipt_no DESC
		 LIMIT $3 OFFSET $4`
	return s.listReceipts(ctx, query, sessionID, guardianID, limit, offset)
}

func (s *Store) listReceipts(ctx context.Context, query string, firstID, secondID string, limit, offset int) ([]Receipt, error) {
	rows, err := s.pool.Query(ctx, query, firstID, secondID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list receipts: %w", err)
	}
	defer rows.Close()

	items := []Receipt{}
	for rows.Next() {
		var item Receipt
		if err := rows.Scan(
			&item.ID, &item.ReceiptNo, &item.StudentID, &item.AdmissionNo,
			&item.StudentName, &item.AmountPaise, &item.Mode, &item.Narration,
			&item.ReceivedAt, &item.CancelledAt,
		); err != nil {
			return nil, fmt.Errorf("scan receipt: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate receipts: %w", err)
	}
	return items, nil
}

// CollectReceipt creates the receipt, locks every affected invoice, records
// allocations, and advances invoice balances as one atomic transaction. If a
// second cashier pays the same invoice first, this transaction observes the
// new balance under the row lock and fails safely instead of over-collecting.
func (s *Store) CollectReceipt(ctx context.Context, input CollectReceiptInput) (Receipt, error) {
	if len(input.Allocations) == 0 {
		return Receipt{}, fmt.Errorf("collect receipt: %w", ErrConflict)
	}

	var result Receipt
	err := s.InTx(ctx, func(tx pgx.Tx) error {
		var total int64
		for _, allocation := range input.Allocations {
			if allocation.AmountPaise <= 0 || allocation.AmountPaise > (1<<62) {
				return fmt.Errorf("collect receipt: invalid allocation: %w", ErrConflict)
			}

			const lockInvoice = `
				SELECT gross_paise, concession_paise, late_fee_paise, paid_paise
				  FROM invoices
				 WHERE id = $1::uuid
				   AND student_id = $2::uuid
				   AND session_id = $3::uuid
				   AND status IN ('DUE', 'PARTIAL')
				 FOR UPDATE`
			var gross, concession, lateFee, paid int64
			if err := tx.QueryRow(ctx, lockInvoice, allocation.InvoiceID, input.StudentID, input.SessionID).
				Scan(&gross, &concession, &lateFee, &paid); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf("collect receipt: invoice is not payable: %w", ErrConflict)
				}
				return fmt.Errorf("lock invoice: %w", err)
			}

			outstanding := gross - concession + lateFee - paid
			if allocation.AmountPaise > outstanding || total > (1<<62)-allocation.AmountPaise {
				return fmt.Errorf("collect receipt: allocation exceeds outstanding amount: %w", ErrConflict)
			}
			total += allocation.AmountPaise
		}

		receiptNo, err := AllocateNumber(ctx, tx, SeriesReceiptNo, input.SessionName)
		if err != nil {
			return err
		}

		const insertReceipt = `
			INSERT INTO receipts (
				receipt_no, student_id, session_id, amount_paise, mode,
				cheque_no, cheque_date, bank_name, narration, received_by
			) VALUES (
				$1, $2::uuid, $3::uuid, $4, $5,
				NULLIF($6, ''), NULLIF($7, '')::date, NULLIF($8, ''), NULLIF($9, ''), $10::uuid
			)
			RETURNING id::text, received_at`
		var receiptID string
		if err := tx.QueryRow(ctx, insertReceipt,
			receiptNo, input.StudentID, input.SessionID, total, input.Mode,
			input.ChequeNo, input.ChequeDate, input.BankName, input.Narration, input.ReceivedBy,
		).Scan(&receiptID, &result.ReceivedAt); err != nil {
			return fmt.Errorf("insert receipt: %w", err)
		}

		for _, allocation := range input.Allocations {
			const insertAllocation = `
				INSERT INTO receipt_allocations (receipt_id, invoice_id, amount_paise)
				VALUES ($1::uuid, $2::uuid, $3)`
			if _, err := tx.Exec(ctx, insertAllocation, receiptID, allocation.InvoiceID, allocation.AmountPaise); err != nil {
				return fmt.Errorf("insert receipt allocation: %w", err)
			}

			const updateInvoice = `
				UPDATE invoices
				   SET paid_paise = paid_paise + $2,
				       status = CASE
				           WHEN paid_paise + $2 >= gross_paise - concession_paise + late_fee_paise THEN 'PAID'
				           ELSE 'PARTIAL'
				       END,
				       updated_at = now()
				 WHERE id = $1::uuid`
			if _, err := tx.Exec(ctx, updateInvoice, allocation.InvoiceID, allocation.AmountPaise); err != nil {
				return fmt.Errorf("update invoice: %w", err)
			}
		}

		result = Receipt{
			ID:          receiptID,
			ReceiptNo:   receiptNo,
			StudentID:   input.StudentID,
			AmountPaise: total,
			Mode:        input.Mode,
			Narration:   input.Narration,
			ReceivedAt:  result.ReceivedAt,
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return Receipt{}, fmt.Errorf("collect receipt: %w", ErrConflict)
		}
		if _, unique := isUniqueViolation(err); unique {
			return Receipt{}, fmt.Errorf("collect receipt: %w", ErrConflict)
		}
		return Receipt{}, err
	}
	return result, nil
}
