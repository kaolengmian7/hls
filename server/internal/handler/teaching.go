package handler

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"golang.org/x/crypto/bcrypt"

	"hls.dev/server/internal/config"
	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/svc"
)

type createTeachingRequest struct {
	Title              string  `json:"title"`
	StudentID          int64   `json:"studentId"`
	NewStudentNickname string  `json:"newStudentNickname"`
	TotalHours         float64 `json:"totalHours"`
	Price              float64 `json:"price"`
	CommissionRate     float64 `json:"commissionRate"`
	BonusRate          float64 `json:"bonusRate"`
}

type topUpRequest struct {
	AddHours float64 `json:"addHours"`
	AddPrice float64 `json:"addPrice"`
}

type scorecardRequest struct {
	DurationHours float64        `json:"durationHours"`
	Scores        map[string]int `json:"scores"`
	Comment       string         `json:"comment"`
}

type teachingReviewRequest struct {
	AttitudeScore  int    `json:"attitudeScore"`
	TechnicalScore int    `json:"technicalScore"`
	Comment        string `json:"comment"`
}

func parseTeachingPath(path string) (string, []string) {
	rest := strings.TrimPrefix(path, "/api/teaching-orders/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil
	}
	return parts[0], parts[1:]
}

func validateCreateTeaching(req createTeachingRequest) string {
	if strings.TrimSpace(req.Title) == "" {
		return "invalid_title"
	}
	if req.TotalHours <= 0 {
		return "invalid_hours"
	}
	if req.CommissionRate < 0 || req.CommissionRate > 100 {
		return "invalid_commission"
	}
	if req.StudentID <= 0 && strings.TrimSpace(req.NewStudentNickname) == "" {
		return "missing_student"
	}
	return ""
}

func validateScores(scores map[string]int, validKeys map[string]bool) string {
	for k, v := range scores {
		if !validKeys[k] {
			return "invalid_score_key"
		}
		if v < 0 || v > 10 {
			return "invalid_score_value"
		}
	}
	return ""
}

func hoursToMinutes(h float64) int { return int(math.Round(h * 60)) }

func minutesToHours(m int) float64 { return math.Round(float64(m)/60*100) / 100 }

func teacherView(o *repo.TeachingOrder) any {
	if !o.TeacherID.Valid {
		return nil
	}
	return map[string]any{"id": o.TeacherID.Int64, "name": o.TeacherName.String}
}

// toTeachingOrderView projects a repo.TeachingOrder into the API response shape.
func toTeachingOrderView(o *repo.TeachingOrder) map[string]any {
	return map[string]any{
		"code":             o.Code,
		"title":            o.Title,
		"student":          map[string]any{"id": o.StudentID, "name": o.StudentName},
		"dispatcherName":   o.DispatcherName,
		"teacher":          teacherView(o),
		"price":            o.Price,
		"commissionRate":   o.CommissionRate,
		"bonusRate":        o.BonusRate,
		"bonusQualified":   o.BonusQualified,
		"totalHours":       minutesToHours(o.TotalMinutes),
		"remainingHours":   minutesToHours(o.RemainingMinutes),
		"totalMinutes":     o.TotalMinutes,
		"remainingMinutes": o.RemainingMinutes,
		"status":           o.Status,
		"earnings":         repo.ComputeEarnings(o.Price, o.CommissionRate, o.BonusRate, o.BonusQualified, 1),
		"createdAt":        o.CreatedAt,
	}
}

// genStudentPassword returns an 8-char random lowercase-alnum password for newly created customer accounts.
func genStudentPassword() string {
	const chars = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, 8)
	_, _ = crand.Read(b)
	out := make([]byte, 8)
	for i, c := range b {
		out[i] = chars[int(c)%len(chars)]
	}
	return string(out)
}

// CreateTeachingOrderHandler POST /api/teaching-orders (owner, service).
func CreateTeachingOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		uid, _ := extractUserID(r, ctx)
		me, err := ctx.UserRepo.FindByID(r.Context(), uid)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		var req createTeachingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		if errKey := validateCreateTeaching(req); errKey != "" {
			writeError(w, r, http.StatusBadRequest, errKey, nil)
			return
		}

		// resolve student: existing id, or create a new customer
		studentID, studentName := req.StudentID, ""
		var newCreds map[string]any
		if studentID > 0 {
			su, err := ctx.UserRepo.FindByID(r.Context(), studentID)
			if err != nil || su.Role != "customer" {
				writeError(w, r, http.StatusBadRequest, "invalid_student", nil)
				return
			}
			studentName = su.Nickname
		} else {
			// create a customer with a random password; return creds so owner can hand them over
			pw := genStudentPassword()
			hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "hash_error", err)
				return
			}
			su, err := ctx.UserRepo.CreateWithPassword(r.Context(), strings.TrimSpace(req.NewStudentNickname), string(hash), "customer")
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			studentID, studentName = su.ID, su.Nickname
			newCreds = map[string]any{"username": su.Username.String, "password": pw}
		}

		o := &repo.TeachingOrder{
			Title: strings.TrimSpace(req.Title), StudentID: studentID, StudentName: studentName,
			DispatcherID: uid, DispatcherName: me.Nickname,
			Price: req.Price, CommissionRate: req.CommissionRate, BonusRate: req.BonusRate,
			TotalMinutes: hoursToMinutes(req.TotalHours),
		}
		op := repo.Operator{ID: uid, Name: me.Nickname, Role: me.Role}
		var created *repo.TeachingOrder
		for i := 0; i < 5; i++ {
			o.Code = genTeachingCode()
			created, err = ctx.TeachingRepo.Create(r.Context(), o, op)
			if err == nil {
				break
			}
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		view := toTeachingOrderView(created)
		if newCreds != nil {
			view["newStudentCredentials"] = newCreds
		}
		httpx.OkJsonCtx(r.Context(), w, view)
	})
}

// ListTeachingOrdersHandler GET /api/teaching-orders (owner, service).
func ListTeachingOrdersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		list, err := ctx.TeachingRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(list))
		for _, o := range list {
			views = append(views, toTeachingOrderView(o))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// TeachingOrderDetailHandler GET /api/teaching-orders/:code (owner, service, its teacher, its student).
func TeachingOrderDetailHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, errKey := extractUserID(r, ctx)
		if errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		me, err := ctx.UserRepo.FindByID(r.Context(), uid)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		code, _ := parseTeachingPath(r.URL.Path)
		o, err := ctx.TeachingRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "teaching_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if !canViewTeaching(me, o) {
			writeError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toTeachingOrderView(o))
	}
}

// canViewTeaching: staff, the assigned teacher, or the student.
func canViewTeaching(u *repo.User, o *repo.TeachingOrder) bool {
	if u.Role == "owner" || u.Role == "service" {
		return true
	}
	if u.Role == "enforcer" && o.TeacherID.Valid && o.TeacherID.Int64 == u.ID {
		return true
	}
	if u.Role == "customer" && o.StudentID == u.ID {
		return true
	}
	return false
}

// MyTeachingHandler GET /api/my-teaching (enforcer: taught; customer: learned).
func MyTeachingHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer", "customer"}, func(w http.ResponseWriter, r *http.Request) {
		uid, _ := extractUserID(r, ctx)
		me, _ := ctx.UserRepo.FindByID(r.Context(), uid)
		var list []*repo.TeachingOrder
		var err error
		if me.Role == "enforcer" {
			list, err = ctx.TeachingRepo.ListByTeacher(r.Context(), uid)
		} else {
			list, err = ctx.TeachingRepo.ListByStudent(r.Context(), uid)
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(list))
		for _, o := range list {
			views = append(views, toTeachingOrderView(o))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// SyllabusHandler GET /api/teaching/syllabus (any authed).
func SyllabusHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, errKey := extractUserID(r, ctx); errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, ctx.Config.TeachingSyllabus)
	}
}

// ListCustomersHandler GET /api/customers (owner, service) — for the student picker.
func ListCustomersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		users, err := ctx.UserRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		out := make([]map[string]any, 0)
		for _, u := range users {
			if u.Role == "customer" {
				out = append(out, map[string]any{"id": u.ID, "nickname": u.Nickname, "username": u.Username})
			}
		}
		httpx.OkJsonCtx(r.Context(), w, out)
	})
}

// genTeachingCode returns a prefixed timestamp code, isolated from order codes.
func genTeachingCode() string {
	return "JX" + genOrderCode()
}

func toScorecardView(sc *repo.TeachingScorecard) map[string]any {
	scores := map[string]int{}
	if sc.Scores.Valid && sc.Scores.String != "" {
		_ = json.Unmarshal([]byte(sc.Scores.String), &scores)
	}
	return map[string]any{
		"id":            sc.ID,
		"durationHours": minutesToHours(sc.DurationMinutes),
		"scores":        scores,
		"comment":       sc.Comment.String,
		"teacherName":   sc.TeacherName,
		"createdAt":     sc.CreatedAt,
	}
}

func toTimeLogView(l *repo.TeachingTimeLog) map[string]any {
	return map[string]any{
		"deltaHours":   minutesToHours(l.DeltaMinutes),
		"reason":       l.Reason,
		"priceDelta":   l.PriceDelta,
		"operatorName": l.OperatorName,
		"operatorRole": l.OperatorRole,
		"note":         l.Note.String,
		"createdAt":    l.CreatedAt,
	}
}

func toTeachingReviewView(rv *repo.TeachingReview) map[string]any {
	return map[string]any{
		"attitudeScore":  rv.AttitudeScore,
		"technicalScore": rv.TechnicalScore,
		"total":          rv.AttitudeScore + rv.TechnicalScore,
		"bonusQualified": BonusQualifies(rv.AttitudeScore, rv.TechnicalScore),
		"comment":        rv.Comment.String,
		"reviewerName":   rv.ReviewerName,
		"createdAt":      rv.CreatedAt,
	}
}

// loadTeachingForActor loads the order and the acting user, returning them or writing an error.
func loadTeachingForActor(ctx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) (*repo.User, *repo.TeachingOrder, bool) {
	uid, errKey := extractUserID(r, ctx)
	if errKey != "" {
		writeError(w, r, http.StatusUnauthorized, errKey, nil)
		return nil, nil, false
	}
	me, err := ctx.UserRepo.FindByID(r.Context(), uid)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
		return nil, nil, false
	}
	code, _ := parseTeachingPath(r.URL.Path)
	o, err := ctx.TeachingRepo.GetByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, sqlx.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "teaching_not_found", nil)
			return nil, nil, false
		}
		writeError(w, r, http.StatusInternalServerError, "db_error", err)
		return nil, nil, false
	}
	return me, o, true
}

func isTeacherOf(u *repo.User, o *repo.TeachingOrder) bool {
	return o.TeacherID.Valid && o.TeacherID.Int64 == u.ID
}

// ClaimTeachingOrderHandler POST /api/teaching-orders/:code/claim (enforcer).
func ClaimTeachingOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		uid, _ := extractUserID(r, ctx)
		me, _ := ctx.UserRepo.FindByID(r.Context(), uid)
		code, _ := parseTeachingPath(r.URL.Path)
		if err := ctx.TeachingRepo.Claim(r.Context(), code, uid, me.Nickname); err != nil {
			if errors.Is(err, repo.ErrTeacherTaken) {
				writeError(w, r, http.StatusConflict, "teacher_taken", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		o, _ := ctx.TeachingRepo.GetByCode(r.Context(), code)
		httpx.OkJsonCtx(r.Context(), w, toTeachingOrderView(o))
	})
}

// TopUpTeachingOrderHandler POST /api/teaching-orders/:code/topup (owner).
func TopUpTeachingOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner"}, func(w http.ResponseWriter, r *http.Request) {
		uid, _ := extractUserID(r, ctx)
		me, _ := ctx.UserRepo.FindByID(r.Context(), uid)
		code, _ := parseTeachingPath(r.URL.Path)
		var req topUpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AddHours <= 0 || req.AddPrice < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_topup", nil)
			return
		}
		op := repo.Operator{ID: uid, Name: me.Nickname, Role: me.Role}
		if err := ctx.TeachingRepo.TopUp(r.Context(), code, hoursToMinutes(req.AddHours), req.AddPrice, op); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		o, _ := ctx.TeachingRepo.GetByCode(r.Context(), code)
		httpx.OkJsonCtx(r.Context(), w, toTeachingOrderView(o))
	})
}

// CreateScorecardHandler POST /api/teaching-orders/:code/scorecards (enforcer = teacher).
func CreateScorecardHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if !isTeacherOf(me, o) {
			writeError(w, r, http.StatusForbidden, "not_your_teaching", nil)
			return
		}
		if o.Status != "ongoing" {
			writeError(w, r, http.StatusConflict, "teaching_not_ongoing", nil)
			return
		}
		var req scorecardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DurationHours <= 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_scorecard", nil)
			return
		}
		if errKey := validateScores(req.Scores, config.SyllabusItemKeys(ctx.Config.TeachingSyllabus)); errKey != "" {
			writeError(w, r, http.StatusBadRequest, errKey, nil)
			return
		}
		scoresJSON := sql.NullString{}
		if len(req.Scores) > 0 {
			b, _ := json.Marshal(req.Scores)
			scoresJSON = sql.NullString{String: string(b), Valid: true}
		}
		sc := &repo.TeachingScorecard{
			TeachingOrderCode: o.Code, DurationMinutes: hoursToMinutes(req.DurationHours),
			Scores: scoresJSON, Comment: sql.NullString{String: strings.TrimSpace(req.Comment), Valid: req.Comment != ""},
			TeacherID: me.ID, TeacherName: me.Nickname,
		}
		saved, err := ctx.TeachingRepo.AddScorecard(r.Context(), o.Code, sc)
		if err != nil {
			if errors.Is(err, repo.ErrInsufficientTime) {
				writeError(w, r, http.StatusConflict, "insufficient_time", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toScorecardView(saved))
	})
}

// ListScorecardsHandler GET /api/teaching-orders/:code/scorecards (owner, teacher, student).
func ListScorecardsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if !canViewTeaching(me, o) {
			writeError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		cards, err := ctx.TeachingRepo.ListScorecards(r.Context(), o.Code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(cards))
		for _, c := range cards {
			views = append(views, toScorecardView(c))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	}
}

// SettleTeachingOrderHandler POST /api/teaching-orders/:code/settle (enforcer = teacher).
func SettleTeachingOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if !isTeacherOf(me, o) {
			writeError(w, r, http.StatusForbidden, "not_your_teaching", nil)
			return
		}
		if err := ctx.TeachingRepo.Settle(r.Context(), o.Code); err != nil {
			if errors.Is(err, repo.ErrTeachingNotSettleable) {
				writeError(w, r, http.StatusConflict, "not_settleable", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		updated, _ := ctx.TeachingRepo.GetByCode(r.Context(), o.Code)
		httpx.OkJsonCtx(r.Context(), w, toTeachingOrderView(updated))
	})
}

// ListTimeLogsHandler GET /api/teaching-orders/:code/timelogs (owner).
func ListTimeLogsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseTeachingPath(r.URL.Path)
		logs, err := ctx.TeachingRepo.ListTimeLogs(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(logs))
		for _, l := range logs {
			views = append(views, toTimeLogView(l))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// UpsertTeachingReviewHandler PUT /api/teaching-orders/:code/review (customer = student).
func UpsertTeachingReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"customer"}, func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if o.StudentID != me.ID {
			writeError(w, r, http.StatusForbidden, "not_your_teaching", nil)
			return
		}
		var req teachingReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil ||
			req.AttitudeScore < 0 || req.TechnicalScore < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_review", nil)
			return
		}
		rv := &repo.TeachingReview{
			TeachingOrderCode: o.Code, AttitudeScore: req.AttitudeScore, TechnicalScore: req.TechnicalScore,
			Comment:    sql.NullString{String: strings.TrimSpace(req.Comment), Valid: req.Comment != ""},
			ReviewerID: me.ID, ReviewerName: me.Nickname,
		}
		if err := ctx.TeachingRepo.UpsertReviewWithEffects(r.Context(), rv, BonusQualifies(req.AttitudeScore, req.TechnicalScore)); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		saved, _ := ctx.TeachingRepo.GetReview(r.Context(), o.Code)
		httpx.OkJsonCtx(r.Context(), w, toTeachingReviewView(saved))
	})
}

// TeachingReviewHandler GET /api/teaching-orders/:code/review (owner, teacher, student).
func TeachingReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if !canViewTeaching(me, o) {
			writeError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		rv, err := ctx.TeachingRepo.GetReview(r.Context(), o.Code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "review_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toTeachingReviewView(rv))
	}
}
