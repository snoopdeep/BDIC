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

func (s *Server) registerHomeworkRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/homework", s.signedIn(s.handleListHomework))
	mux.Handle("POST /api/v1/homework", s.restricted(s.handleCreateHomework,
		auth.RoleTeacher, auth.RolePrincipal, auth.RoleSuperAdmin))
	mux.Handle("POST /api/v1/homework/{homeworkId}/submissions", s.restricted(s.handleSubmitHomework, auth.RoleStudent))
	mux.Handle("POST /api/v1/homework/submissions/{submissionId}/review", s.restricted(s.handleReviewHomeworkSubmission,
		auth.RoleTeacher, auth.RolePrincipal, auth.RoleSuperAdmin))
}

func (s *Server) handleListHomework(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	limit, offset := paginate(r)
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	var items []store.Homework
	switch {
	case identity.HasRole(auth.RoleStudent):
		if identity.StudentID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListHomeworkForStudent(ctx, sessionID, identity.StudentID, limit, offset)
	case identity.HasRole(auth.RoleParent):
		if identity.GuardianID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListHomeworkForGuardian(ctx, sessionID, identity.GuardianID, limit, offset)
	case identity.HasRole(auth.RoleTeacher):
		if identity.StaffID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListHomeworkForStaff(ctx, sessionID, identity.StaffID, limit, offset)
	case identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal):
		items, err = s.store.ListHomeworkAll(ctx, sessionID, limit, offset)
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

type createHomeworkRequest struct {
	SectionID    string `json:"sectionId"`
	SubjectID    string `json:"subjectId"`
	Title        string `json:"title"`
	Instructions string `json:"instructions"`
	DueDate      string `json:"dueDate"`
	AllowUpload  bool   `json:"allowUpload"`
	Status       string `json:"status"`
	Attachments  []struct {
		FileID  string `json:"fileId"`
		LinkURL string `json:"linkUrl"`
		Label   string `json:"label"`
	} `json:"attachments"`
}

func (s *Server) handleCreateHomework(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	var input createHomeworkRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Instructions = strings.TrimSpace(input.Instructions)
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	if input.Status == "" {
		input.Status = "PUBLISHED"
	}
	if !auth.IsUUID(input.SectionID) || !auth.IsUUID(input.SubjectID) || input.Title == "" || input.Instructions == "" || len(input.Title) > 200 || len(input.Instructions) > 10_000 || (input.Status != "DRAFT" && input.Status != "PUBLISHED") {
		httpx.Fail(w, r, httpx.ErrBadRequest("Provide section, subject, title, instructions, and a DRAFT or PUBLISHED status.", "सेक्शन, विषय, शीर्षक, निर्देश और DRAFT या PUBLISHED स्थिति दें।"))
		return
	}
	if _, err := time.Parse("2006-01-02", input.DueDate); err != nil {
		httpx.Fail(w, r, httpx.ErrBadRequest("Use YYYY-MM-DD for the due date.", "अंतिम तिथि के लिए YYYY-MM-DD दें।").WithField("dueDate", "Invalid date"))
		return
	}
	if len(input.Attachments) > 10 {
		httpx.Fail(w, r, httpx.ErrBadRequest("A homework item can have up to ten attachments.", "एक गृहकार्य में अधिकतम दस संलग्नक हो सकते हैं।").WithField("attachments", "Too many attachments"))
		return
	}
	attachments := make([]store.HomeworkAttachmentInput, 0, len(input.Attachments))
	for _, attachment := range input.Attachments {
		attachment.FileID = strings.TrimSpace(attachment.FileID)
		attachment.LinkURL = strings.TrimSpace(attachment.LinkURL)
		attachment.Label = strings.TrimSpace(attachment.Label)
		if (attachment.FileID == "" && attachment.LinkURL == "") || (attachment.FileID != "" && attachment.LinkURL != "") || (attachment.FileID != "" && !auth.IsUUID(attachment.FileID)) || len(attachment.LinkURL) > 2_000 || len(attachment.Label) > 200 {
			httpx.Fail(w, r, httpx.ErrBadRequest("Each attachment needs exactly one valid uploaded file or link.", "हर संलग्नक में एक मान्य अपलोड की गई फ़ाइल या लिंक होना चाहिए।").WithField("attachments", "Invalid attachment"))
			return
		}
		attachments = append(attachments, store.HomeworkAttachmentInput{FileID: attachment.FileID, LinkURL: attachment.LinkURL, Label: attachment.Label})
	}

	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 15*time.Second)
	defer cancel()
	staffID := identity.StaffID
	if identity.HasRole(auth.RoleTeacher) {
		if staffID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		allowed, err := s.store.TeacherMayPublishHomework(ctx, sessionID, input.SectionID, input.SubjectID, staffID)
		if err != nil {
			httpx.Fail(w, r, storeError(err))
			return
		}
		if !allowed {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
	}
	if staffID == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest("Assign a staff profile to this account before publishing homework.", "गृहकार्य प्रकाशित करने से पहले इस खाते को स्टाफ प्रोफ़ाइल से जोड़ें।"))
		return
	}
	id, err := s.store.CreateHomework(ctx, store.NewHomework{
		SessionID: sessionID, SectionID: input.SectionID, SubjectID: input.SubjectID, StaffID: staffID,
		Title: input.Title, Instructions: input.Instructions, DueDate: input.DueDate, AllowUpload: input.AllowUpload,
		Status: input.Status, Attachments: attachments, CreatorUserID: identity.UserID,
	})
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{Action: audit.ActionPublish, EntityType: "homework", EntityID: id, Summary: "Published homework: " + input.Title, AfterState: input})
	httpx.JSON(w, http.StatusCreated, map[string]string{"id": id})
}

type submitHomeworkRequest struct {
	FileID string `json:"fileId"`
	Note   string `json:"note"`
}

func (s *Server) handleSubmitHomework(w http.ResponseWriter, r *http.Request) {
	homeworkID, err := pathUUID(r, "homeworkId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	identity := auth.MustFromContext(r.Context())
	if identity.StudentID == "" {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}
	var input submitHomeworkRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	input.FileID = strings.TrimSpace(input.FileID)
	input.Note = strings.TrimSpace(input.Note)
	if input.FileID != "" && !auth.IsUUID(input.FileID) || len(input.Note) > 4_000 {
		httpx.Fail(w, r, httpx.ErrBadRequest("Provide an uploaded submission file and/or a short note.", "अपलोड की गई सबमिशन फ़ाइल और/या संक्षिप्त नोट दें।"))
		return
	}
	if input.FileID == "" && input.Note == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest("Add a file or a note before submitting.", "जमा करने से पहले फ़ाइल या नोट जोड़ें।"))
		return
	}
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	id, err := s.store.SubmitHomework(ctx, homeworkID, identity.StudentID, identity.UserID, input.FileID, input.Note)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{Action: audit.ActionCreate, EntityType: "homework_submission", EntityID: id, Summary: "Submitted homework"})
	httpx.JSON(w, http.StatusCreated, map[string]string{"id": id})
}

type reviewHomeworkSubmissionRequest struct {
	Status string `json:"status"`
	Remark string `json:"remark"`
}

func (s *Server) handleReviewHomeworkSubmission(w http.ResponseWriter, r *http.Request) {
	submissionID, err := pathUUID(r, "submissionId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	identity := auth.MustFromContext(r.Context())
	var input reviewHomeworkSubmissionRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	input.Remark = strings.TrimSpace(input.Remark)
	if (input.Status != "SEEN" && input.Status != "ACCEPTED" && input.Status != "RETURNED") || len(input.Remark) > 4_000 {
		httpx.Fail(w, r, httpx.ErrBadRequest("Choose SEEN, ACCEPTED, or RETURNED and keep feedback concise.", "SEEN, ACCEPTED या RETURNED चुनें और प्रतिक्रिया संक्षिप्त रखें।"))
		return
	}
	if identity.HasRole(auth.RoleTeacher) && identity.StaffID == "" {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	if err := s.store.ReviewHomeworkSubmission(ctx, submissionID, identity.UserID, identity.StaffID, input.Status, input.Remark, identity.CanManageSchool()); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{Action: audit.ActionUpdate, EntityType: "homework_submission", EntityID: submissionID, Summary: "Reviewed homework submission", AfterState: input})
	httpx.JSON(w, http.StatusOK, map[string]string{"id": submissionID, "status": input.Status})
}
