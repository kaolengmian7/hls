package handler

import (
	"math"
	"strings"
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
