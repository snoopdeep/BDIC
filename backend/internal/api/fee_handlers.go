package api

import (
	"net/http"
	"strings"
	"time"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

// registerFeeRoutes exposes the accounting ledger with role-aware row
// scoping. Families may see only their own invoices and receipts; cash
// collection remains restricted to the accounts office and school leadership.
func (s *Server) registerFeeRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/fees/invoices", s.signedIn(s.handleListInvoices))
	mux.Handle("GET /api/v1/fees/receipts", s.signedIn(s.handleListReceipts))
	mux.Handle("GET /api/v1/fees/collection-report", s.restricted(s.handleCollectionReport,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleAccounts))
	mux.Handle("GET /api/v1/fees/next-receipt", s.restricted(s.handlePeekReceiptNumber,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleAccounts))
	mux.Handle("POST /api/v1/fees/invoices", s.restricted(s.handleCreateInvoice,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleAccounts))
	mux.Handle("POST /api/v1/fees/receipts", s.restricted(s.handleCollectReceipt,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleAccounts))
}

type createInvoiceRequest struct {
	StudentID     string `json:"studentId"`
	InstallmentNo int    `json:"installmentNo"`
	DueDate       string `json:"dueDate"`
	Items         []struct {
		FeeHeadID       string `json:"feeHeadId"`
		AmountPaise     int64  `json:"amountPaise"`
		ConcessionPaise int64  `json:"concessionPaise"`
	} `json:"items"`
}

func (s *Server) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	var input createInvoiceRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if !auth.IsUUID(input.StudentID) || input.InstallmentNo < 1 || input.InstallmentNo > 12 || len(input.Items) == 0 || len(input.Items) > 50 {
		httpx.Fail(w, r, httpx.ErrBadRequest("Choose a student, installment (1–12), and one or more fee items.", "विद्यार्थी, किस्त (1–12) और एक या अधिक शुल्क मद चुनें।"))
		return
	}
	if _, err := time.Parse("2006-01-02", input.DueDate); err != nil {
		httpx.Fail(w, r, httpx.ErrBadRequest("Use YYYY-MM-DD for the due date.", "अंतिम तिथि के लिए YYYY-MM-DD दें।").WithField("dueDate", "Invalid date"))
		return
	}
	items := make([]store.InvoiceItemInput, 0, len(input.Items))
	seen := make(map[string]bool, len(input.Items))
	for _, item := range input.Items {
		if !auth.IsUUID(item.FeeHeadID) || seen[item.FeeHeadID] || item.AmountPaise <= 0 || item.ConcessionPaise < 0 || item.ConcessionPaise > item.AmountPaise {
			httpx.Fail(w, r, httpx.ErrBadRequest("Each fee head must be unique with a valid non-negative amount and concession.", "हर शुल्क मद अलग हो और राशि व रियायत मान्य हो।").WithField("items", "Invalid fee item"))
			return
		}
		seen[item.FeeHeadID] = true
		items = append(items, store.InvoiceItemInput{FeeHeadID: item.FeeHeadID, AmountPaise: item.AmountPaise, ConcessionPaise: item.ConcessionPaise})
	}
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 15*time.Second)
	defer cancel()
	session, err := s.store.GetAcademicSession(ctx, sessionID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	invoice, err := s.store.CreateInvoice(ctx, store.CreateInvoiceInput{StudentID: input.StudentID, SessionID: session.ID, SessionName: session.Name, InstallmentNo: input.InstallmentNo, DueDate: input.DueDate, Items: items})
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{Action: audit.ActionCreate, EntityType: "invoice", EntityID: invoice.ID, Summary: "Issued invoice " + invoice.InvoiceNo, AfterState: invoice})
	httpx.JSON(w, http.StatusCreated, map[string]any{"invoice": invoice})
}

func (s *Server) handleCollectionReport(w http.ResponseWriter, r *http.Request) {
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	items, err := s.store.ListCollectionDays(r.Context(), sessionID, 90)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListInvoices(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	limit, offset := paginate(r)
	studentID, err := queryUUID(r, "studentId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	var items []store.Invoice
	switch {
	case identity.HasRole(auth.RoleStudent):
		if identity.StudentID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListInvoicesForStudent(ctx, sessionID, identity.StudentID, limit, offset)
	case identity.HasRole(auth.RoleParent):
		if identity.GuardianID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListInvoicesForGuardian(ctx, sessionID, identity.GuardianID, limit, offset)
	case identity.CanManageFees():
		items, err = s.store.ListInvoicesAll(ctx, sessionID, studentID, limit, offset)
	default:
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items), "limit": limit, "offset": offset})
}

func (s *Server) handleListReceipts(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	limit, offset := paginate(r)
	studentID, err := queryUUID(r, "studentId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	var items []store.Receipt
	switch {
	case identity.HasRole(auth.RoleStudent):
		if identity.StudentID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListReceiptsForStudent(ctx, sessionID, identity.StudentID, limit, offset)
	case identity.HasRole(auth.RoleParent):
		if identity.GuardianID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListReceiptsForGuardian(ctx, sessionID, identity.GuardianID, limit, offset)
	case identity.CanManageFees():
		items, err = s.store.ListReceiptsAll(ctx, sessionID, studentID, limit, offset)
	default:
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items), "limit": limit, "offset": offset})
}

func (s *Server) handlePeekReceiptNumber(w http.ResponseWriter, r *http.Request) {
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	session, err := s.store.GetAcademicSession(ctx, sessionID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	number, err := s.store.PeekNumber(ctx, store.SeriesReceiptNo, session.Name)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"receiptNo": number})
}

type collectReceiptRequest struct {
	StudentID   string `json:"studentId"`
	Mode        string `json:"mode"`
	ChequeNo    string `json:"chequeNo"`
	ChequeDate  string `json:"chequeDate"`
	BankName    string `json:"bankName"`
	Narration   string `json:"narration"`
	Allocations []struct {
		InvoiceID   string `json:"invoiceId"`
		AmountPaise int64  `json:"amountPaise"`
	} `json:"allocations"`
}

func (s *Server) handleCollectReceipt(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	var input collectReceiptRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	input.StudentID = strings.TrimSpace(input.StudentID)
	input.Mode = strings.ToUpper(strings.TrimSpace(input.Mode))
	input.ChequeNo = strings.TrimSpace(input.ChequeNo)
	input.ChequeDate = strings.TrimSpace(input.ChequeDate)
	input.BankName = strings.TrimSpace(input.BankName)
	input.Narration = strings.TrimSpace(input.Narration)
	if !auth.IsUUID(input.StudentID) {
		httpx.Fail(w, r, httpx.ErrBadRequest("Choose a valid student.", "एक मान्य छात्र चुनें।").WithField("studentId", "Required"))
		return
	}
	if len(input.Allocations) == 0 || len(input.Allocations) > 50 {
		httpx.Fail(w, r, httpx.ErrBadRequest("Add between one and fifty invoice allocations.", "एक से पचास तक चालान आवंटन जोड़ें।").WithField("allocations", "Required"))
		return
	}
	validModes := map[string]bool{"CASH": true, "CHEQUE": true, "UPI": true, "CARD": true, "NETBANKING": true, "BANK_TRANSFER": true}
	if !validModes[input.Mode] {
		httpx.Fail(w, r, httpx.ErrBadRequest("Choose a valid payment mode.", "एक मान्य भुगतान माध्यम चुनें।").WithField("mode", "Invalid payment mode"))
		return
	}
	if input.Mode == "CHEQUE" {
		if input.ChequeNo == "" || input.ChequeDate == "" {
			httpx.Fail(w, r, httpx.ErrBadRequest("Cheque number and date are required for cheque payments.", "चेक भुगतान के लिए चेक संख्या और दिनांक आवश्यक है।"))
			return
		}
		if _, err := time.Parse("2006-01-02", input.ChequeDate); err != nil {
			httpx.Fail(w, r, httpx.ErrBadRequest("Use YYYY-MM-DD for the cheque date.", "चेक की तिथि YYYY-MM-DD में दें।").WithField("chequeDate", "Invalid date"))
			return
		}
	}

	seen := make(map[string]bool, len(input.Allocations))
	allocations := make([]store.ReceiptAllocationInput, 0, len(input.Allocations))
	for _, allocation := range input.Allocations {
		if !auth.IsUUID(allocation.InvoiceID) || allocation.AmountPaise <= 0 || seen[allocation.InvoiceID] {
			httpx.Fail(w, r, httpx.ErrBadRequest("Each allocation needs a different invoice and a positive paise amount.", "हर आवंटन में अलग चालान और सकारात्मक पैसे की राशि होनी चाहिए।").WithField("allocations", "Invalid allocation"))
			return
		}
		seen[allocation.InvoiceID] = true
		allocations = append(allocations, store.ReceiptAllocationInput{InvoiceID: allocation.InvoiceID, AmountPaise: allocation.AmountPaise})
	}

	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 15*time.Second)
	defer cancel()
	session, err := s.store.GetAcademicSession(ctx, sessionID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	receipt, err := s.store.CollectReceipt(ctx, store.CollectReceiptInput{
		StudentID: input.StudentID, SessionID: session.ID, SessionName: session.Name,
		Mode: input.Mode, ChequeNo: input.ChequeNo, ChequeDate: input.ChequeDate,
		BankName: input.BankName, Narration: input.Narration, ReceivedBy: identity.UserID,
		Allocations: allocations,
	})
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{
		Action: audit.ActionFeeCollect, EntityType: "receipt", EntityID: receipt.ID,
		Summary:    "Collected " + receipt.ReceiptNo + " for " + receipt.StudentID,
		AfterState: map[string]any{"receiptNo": receipt.ReceiptNo, "studentId": receipt.StudentID, "amountPaise": receipt.AmountPaise, "mode": receipt.Mode},
	})
	httpx.JSON(w, http.StatusCreated, map[string]any{"receipt": receipt})
}
