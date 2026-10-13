package reports

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes report HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a reports Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}

func (h *Handler) storeID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.BadRequest(c, "Store not found — please select an active store", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) orgID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) userID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "User not found")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) paramID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid id", nil)
		return uuid.Nil, false
	}
	return id, true
}

// ReportTypes godoc: GET /reports/types
func (h *Handler) ReportTypes(c *gin.Context) {
	response.OK(c, map[string]interface{}{"report_types": h.svc.ReportTypes()})
}

// Generate godoc: POST /reports/generate — CSV download or JSON report data.
func (h *Handler) Generate(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var req struct {
		ReportType string `json:"report_type" binding:"required"`
		Format     string `json:"format"`
		Range      string `json:"range"`
		DateFrom   string `json:"date_from"`
		DateTo     string `json:"date_to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ReportType == "" {
		response.BadRequest(c, "report_type is required", nil)
		return
	}
	format := req.Format
	if format != FormatJSON {
		format = FormatCSV
	}

	query := func(k string) string {
		switch k {
		case "range":
			return req.Range
		case "date_from":
			return req.DateFrom
		case "date_to":
			return req.DateTo
		}
		return ""
	}
	rng := ParseRangeForReport(query)

	data, err := h.svc.GenerateReport(c.Request.Context(), req.ReportType, storeID, userID, rng)
	if err != nil {
		writeErr(c, err)
		return
	}

	if format == FormatCSV {
		csvStr, fname := h.svc.BuildCSV(req.ReportType, data)
		size := len(csvStr)
		_ = h.svc.RecordExport(c.Request.Context(), req.ReportType, storeID, orgID, userID, rng, FormatCSV, size)
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="`+fname+`"`)
		c.String(200, csvStr)
		return
	}

	raw, _ := json.Marshal(data)
	size := len(raw)
	_ = h.svc.RecordExport(c.Request.Context(), req.ReportType, storeID, orgID, userID, rng, FormatJSON, size)

	response.OK(c, map[string]interface{}{
		"report_type":  req.ReportType,
		"period_label": rng.Label,
		"date_from":    rng.From.Format("2006-01-02"),
		"date_to":      rng.To.Format("2006-01-02"),
		"report":       data,
	})
}

// Exports godoc: GET /reports/exports
func (h *Handler) Exports(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	out, err := h.svc.ListExports(c.Request.Context(), storeID, c.Query("report_type"), c.Query("format"), limit)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// DownloadExport godoc: GET /reports/exports/{id}/download
func (h *Handler) DownloadExport(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	id, ok := h.paramID(c)
	if !ok {
		return
	}
	export, err := h.svc.GetExport(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	rng := buildReportRange(export.DateFrom, export.DateTo)
	data, err := h.svc.GenerateReport(c.Request.Context(), export.ReportType, storeID, uuid.Nil, rng)
	if err != nil {
		writeErr(c, err)
		return
	}

	if export.Format == FormatCSV {
		csvStr, fname := h.svc.BuildCSV(export.ReportType, data)
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="`+fname+`"`)
		c.String(200, csvStr)
		return
	}
	response.OK(c, map[string]interface{}{
		"report":       data,
		"period_label": export.PeriodLabel,
	})
}

// ListScheduled godoc: GET /reports/scheduled
func (h *Handler) ListScheduled(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	out, err := h.svc.ListScheduled(c.Request.Context(), storeID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// CreateScheduled godoc: POST /reports/scheduled (owner/admin)
func (h *Handler) CreateScheduled(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var req struct {
		ReportType      string   `json:"report_type" binding:"required"`
		Name            string   `json:"name"`
		Frequency       string   `json:"frequency"`
		DateRangePreset string   `json:"date_range_preset"`
		Recipients      []string `json:"recipients"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ReportType == "" {
		response.BadRequest(c, "report_type is required", nil)
		return
	}
	out, err := h.svc.CreateScheduled(c.Request.Context(), storeID, orgID, userID,
		req.ReportType, req.Name, req.Frequency, req.DateRangePreset, req.Recipients)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, out, "Scheduled report created")
}

// UpdateScheduled godoc: PATCH /reports/scheduled/{id} (owner/admin)
func (h *Handler) UpdateScheduled(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	id, ok := h.paramID(c)
	if !ok {
		return
	}
	var req struct {
		Name            *string  `json:"name"`
		Frequency       *string  `json:"frequency"`
		DateRangePreset *string  `json:"date_range_preset"`
		Recipients      []string `json:"recipients"`
		IsEnabled       *bool    `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	name, frequency, preset := "", "", ""
	recips := []string(nil)
	if req.Name != nil {
		name = *req.Name
	}
	if req.Frequency != nil {
		frequency = *req.Frequency
	}
	if req.DateRangePreset != nil {
		preset = *req.DateRangePreset
	}
	if req.Recipients != nil {
		recips = req.Recipients
	}
	out, err := h.svc.UpdateScheduled(c.Request.Context(), storeID, id, name, frequency, preset, recips, req.IsEnabled)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// DeleteScheduled godoc: DELETE /reports/scheduled/{id} (owner/admin)
func (h *Handler) DeleteScheduled(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	id, ok := h.paramID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteScheduled(c.Request.Context(), storeID, id); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, nil)
}

// SendScheduled godoc: POST /reports/scheduled/{id}/send (owner/admin)
func (h *Handler) SendScheduled(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	id, ok := h.paramID(c)
	if !ok {
		return
	}
	out, err := h.svc.SendScheduled(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}
