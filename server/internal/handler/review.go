package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/svc"
)

// reviewEffects derives the score total, chicken-leg (bonus) qualification,
// and the resulting order complaint status from a review's two scores.
// Rule: total = attitude + technical; total>=8 qualifies for bonus;
// total<6 marks the order complained; total>=6 only reverts an
// auto-set "complained" back to "none" and never touches "resolved".
func reviewEffects(attitude, technical int, currentComplaint string) (total int, bonusQualified bool, newComplaint string) {
	total = attitude + technical
	bonusQualified = total >= 8
	switch {
	case total < 6:
		newComplaint = "complained"
	case currentComplaint == "complained":
		newComplaint = "none"
	default:
		newComplaint = currentComplaint
	}
	return
}

type reviewRequest struct {
	AttitudeScore    int    `json:"attitudeScore"`
	TechnicalScore   int    `json:"technicalScore"`
	ReviewerName     string `json:"reviewerName"`
	BlameBoss        bool   `json:"blameBoss"`
	Late             bool   `json:"late"`
	TempUnavailable  bool   `json:"tempUnavailable"`
	SlowReply        bool   `json:"slowReply"`
	NoBackpack       bool   `json:"noBackpack"`
	NoChannelCard    bool   `json:"noChannelCard"`
	NoSupplyCard     bool   `json:"noSupplyCard"`
	NoEmotionalValue bool   `json:"noEmotionalValue"`
	GrabContainer    bool   `json:"grabContainer"`
	Comment          string `json:"comment"`
}

func normalizeReviewerName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "匿名评价"
	}
	runes := []rune(name)
	if len(runes) > 32 {
		return string(runes[:32])
	}
	return name
}

func publicOrderReviewView(order *repo.Order, rv *repo.Review) map[string]any {
	var review any
	if rv != nil {
		review = toReviewView(rv)
	}
	return map[string]any{
		"order":  toOrderView(order),
		"review": review,
	}
}

// toReviewView builds the canonical JSON view of a review.
func toReviewView(rv *repo.Review) map[string]any {
	total := rv.AttitudeScore + rv.TechnicalScore
	return map[string]any{
		"orderCode":        rv.OrderCode,
		"attitudeScore":    rv.AttitudeScore,
		"technicalScore":   rv.TechnicalScore,
		"total":            total,
		"bonusQualified":   total >= 8,
		"blameBoss":        rv.BlameBoss,
		"late":             rv.Late,
		"tempUnavailable":  rv.TempUnavailable,
		"slowReply":        rv.SlowReply,
		"noBackpack":       rv.NoBackpack,
		"noChannelCard":    rv.NoChannelCard,
		"noSupplyCard":     rv.NoSupplyCard,
		"noEmotionalValue": rv.NoEmotionalValue,
		"grabContainer":    rv.GrabContainer,
		"comment":          rv.Comment.String,
		"reviewerName":     rv.ReviewerName,
		"createdAt":        rv.CreatedAt,
	}
}

// UpsertReviewHandler handles PUT /api/orders/:code/review (owner/service).
// Writes the single review for the order, then updates the order's
// bonus_qualified and complaint_status per reviewEffects.
func UpsertReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		code, _ := parseOrderPath(r.URL.Path)
		var req reviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Comment = strings.TrimSpace(req.Comment)
		if req.AttitudeScore < 1 || req.AttitudeScore > 5 ||
			req.TechnicalScore < 1 || req.TechnicalScore > 5 ||
			len([]rune(req.Comment)) > 500 {
			writeError(w, r, http.StatusBadRequest, "invalid_review", nil)
			return
		}
		order, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "order_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		_, bonusQualified, newComplaint := reviewEffects(req.AttitudeScore, req.TechnicalScore, order.ComplaintStatus)
		rv := &repo.Review{
			OrderCode:        code,
			AttitudeScore:    req.AttitudeScore,
			TechnicalScore:   req.TechnicalScore,
			BlameBoss:        req.BlameBoss,
			Late:             req.Late,
			TempUnavailable:  req.TempUnavailable,
			SlowReply:        req.SlowReply,
			NoBackpack:       req.NoBackpack,
			NoChannelCard:    req.NoChannelCard,
			NoSupplyCard:     req.NoSupplyCard,
			NoEmotionalValue: req.NoEmotionalValue,
			GrabContainer:    req.GrabContainer,
			Comment:          sql.NullString{String: req.Comment, Valid: req.Comment != ""},
			ReviewerID:       userID,
			ReviewerName:     user.Nickname,
		}
		if err := ctx.ReviewRepo.UpsertWithEffects(r.Context(), rv, bonusQualified, newComplaint); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		saved, err := ctx.ReviewRepo.GetByOrderCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toReviewView(saved))
	})
}

// OrderReviewHandler handles GET /api/orders/:code/review (owner/service).
func OrderReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseOrderPath(r.URL.Path)
		rv, err := ctx.ReviewRepo.GetByOrderCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "review_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toReviewView(rv))
	})
}

// PublicUpsertReviewHandler handles PUT /api/public/orders/:code/review without auth.
func PublicUpsertReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := extractCode(r.URL.Path, "/api/public/orders/")
		var req reviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Comment = strings.TrimSpace(req.Comment)
		req.ReviewerName = normalizeReviewerName(req.ReviewerName)
		if req.AttitudeScore < 1 || req.AttitudeScore > 5 ||
			req.TechnicalScore < 1 || req.TechnicalScore > 5 ||
			len([]rune(req.Comment)) > 500 {
			writeError(w, r, http.StatusBadRequest, "invalid_review", nil)
			return
		}
		order, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "order_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		_, bonusQualified, newComplaint := reviewEffects(req.AttitudeScore, req.TechnicalScore, order.ComplaintStatus)
		rv := &repo.Review{
			OrderCode:        code,
			AttitudeScore:    req.AttitudeScore,
			TechnicalScore:   req.TechnicalScore,
			BlameBoss:        req.BlameBoss,
			Late:             req.Late,
			TempUnavailable:  req.TempUnavailable,
			SlowReply:        req.SlowReply,
			NoBackpack:       req.NoBackpack,
			NoChannelCard:    req.NoChannelCard,
			NoSupplyCard:     req.NoSupplyCard,
			NoEmotionalValue: req.NoEmotionalValue,
			GrabContainer:    req.GrabContainer,
			Comment:          sql.NullString{String: req.Comment, Valid: req.Comment != ""},
			ReviewerID:       0,
			ReviewerName:     req.ReviewerName,
		}
		if err := ctx.ReviewRepo.UpsertWithEffects(r.Context(), rv, bonusQualified, newComplaint); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		saved, err := ctx.ReviewRepo.GetByOrderCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toReviewView(saved))
	}
}

// PublicOrderReviewHandler handles GET /api/public/orders/:code/review without auth.
func PublicOrderReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := extractCode(r.URL.Path, "/api/public/orders/")
		order, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "order_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		rv, err := ctx.ReviewRepo.GetByOrderCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				httpx.OkJsonCtx(r.Context(), w, publicOrderReviewView(order, nil))
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, publicOrderReviewView(order, rv))
	}
}
