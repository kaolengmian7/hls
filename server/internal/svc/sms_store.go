package svc

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

type codeEntry struct {
	code      string
	expiresAt time.Time
	failCount int
}

// SmsStore is an in-memory store for SMS verification codes and rate-limit counters.
// All state resets on process restart. Counters reset at midnight UTC+8 each day.
type SmsStore struct {
	nowFn      func() time.Time
	mu         sync.Mutex
	codes      map[string]*codeEntry // key: phone
	lastSentAt map[string]time.Time  // key: phone; persists across verifies for cooldown
	phoneCount map[string]int        // key: "phone|YYYYMMDD"
	ipCount    map[string]int        // key: "ip|YYYYMMDD"
	globalDay  string
	globalCnt  int
}

// NewSmsStore creates an empty SmsStore using real time.
func NewSmsStore() *SmsStore {
	return &SmsStore{
		nowFn:      time.Now,
		codes:      make(map[string]*codeEntry),
		lastSentAt: make(map[string]time.Time),
		phoneCount: make(map[string]int),
		ipCount:    make(map[string]int),
	}
}

// NewSmsStoreWithClock creates an SmsStore using the provided clock. Use in tests only.
func NewSmsStoreWithClock(nowFn func() time.Time) *SmsStore {
	s := NewSmsStore()
	s.nowFn = nowFn
	return s
}

func (s *SmsStore) dayKey() string {
	return s.nowFn().In(time.FixedZone("CST", 8*3600)).Format("20060102")
}

// CheckAndSend performs all rate-limit checks and, if they pass, generates and stores
// a 6-digit code. Returns (code, retryAfter, errKey). errKey is "" on success.
func (s *SmsStore) CheckAndSend(phone, ip string, maxPhone, maxIP, maxGlobal int) (code string, retryAfter int, errKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	day := s.dayKey()

	// global daily hard cap
	if s.globalDay != day {
		s.globalCnt = 0
		s.globalDay = day
	}
	if s.globalCnt >= maxGlobal {
		return "", 0, "global_limit"
	}

	// 60-second cooldown per phone
	if sentAt, ok := s.lastSentAt[phone]; ok {
		if until := sentAt.Add(60 * time.Second); s.nowFn().Before(until) {
			remaining := int(until.Sub(s.nowFn()).Seconds()) + 1
			return "", remaining, "cooldown"
		}
	}

	// phone daily limit
	pk := phone + "|" + day
	if s.phoneCount[pk] >= maxPhone {
		return "", 0, "phone_limit"
	}

	// IP daily limit
	ik := ip + "|" + day
	if s.ipCount[ik] >= maxIP {
		return "", 0, "ip_limit"
	}

	newCode, err := genSmsCode()
	if err != nil {
		return "", 0, "internal"
	}

	now := s.nowFn()
	s.codes[phone] = &codeEntry{
		code:      newCode,
		expiresAt: now.Add(10 * time.Minute),
	}
	s.lastSentAt[phone] = now
	s.phoneCount[pk]++
	s.ipCount[ik]++
	s.globalCnt++

	return newCode, 0, ""
}

// Verify checks the provided code for the given phone.
// Returns "" on success, or "invalid_code" / "code_expired" on failure.
func (s *SmsStore) Verify(phone, code string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.codes[phone]
	if !ok {
		return "invalid_code"
	}
	if s.nowFn().After(e.expiresAt) {
		delete(s.codes, phone)
		return "code_expired"
	}
	if e.code != code {
		e.failCount++
		if e.failCount >= 5 {
			delete(s.codes, phone)
		}
		return "invalid_code"
	}
	delete(s.codes, phone)
	return ""
}

func genSmsCode() (string, error) {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}
