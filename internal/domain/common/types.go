package common

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	WarnScopeAmbiguous         = "WARN_SCOPE_AMBIGUOUS"
	WarnScopeFallback          = "WARN_SCOPE_FALLBACK_USED"
	WarnDedupeApplied          = "WARN_DEDUPE_APPLIED"
	WarnHandoffSparse          = "WARN_HANDOFF_SPARSE"
	WarnNoPriorHandoff         = "WARN_NO_PRIOR_HANDOFF"
	WarnNoPriorNotes           = "WARN_NO_PRIOR_NOTES"
	WarnRelatedProjectsSkipped = "WARN_RELATED_PROJECTS_SKIPPED"
	WarnRecoveryHandoffUsed    = "WARN_RECOVERY_HANDOFF_USED"
	ErrInvalidInput            = "ERR_INVALID_INPUT"
	ErrInvalidScope            = "ERR_INVALID_SCOPE"
	ErrScopeConflict           = "ERR_SCOPE_CONFLICT"
	ErrInvalidState            = "ERR_INVALID_STATE"
	ErrSessionNotFound         = "ERR_SESSION_NOT_FOUND"
	ErrStorageUnavailable      = "ERR_STORAGE_UNAVAILABLE"
	ErrWriteFailed             = "ERR_WRITE_FAILED"
	ErrReadFailed              = "ERR_READ_FAILED"
)

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CodedError struct {
	Code    string
	Message string
	Err     error
}

func (e *CodedError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
}

func (e *CodedError) Unwrap() error {
	return e.Err
}

func NewError(code, message string) error {
	return &CodedError{Code: code, Message: message}
}

func WrapError(code, message string, err error) error {
	return &CodedError{Code: code, Message: message, Err: err}
}

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now().UTC()
}

type IDFactory interface {
	New(prefix string) string
}

type DefaultIDFactory struct {
	Clock Clock
}

func (f DefaultIDFactory) New(prefix string) string {
	now := time.Now().UTC()
	if f.Clock != nil {
		now = f.Clock.Now().UTC()
	}

	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		hash := sha1.Sum([]byte(now.Format(time.RFC3339Nano)))
		return fmt.Sprintf("%s_%s_%s", prefix, now.Format("20060102_150405"), hex.EncodeToString(hash[:4]))
	}

	return fmt.Sprintf("%s_%s_%s", prefix, now.Format("20060102_150405"), hex.EncodeToString(buf))
}

func StableID(prefix, key string) string {
	hash := sha1.Sum([]byte(strings.TrimSpace(key)))
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(hash[:6]))
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func Slug(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = nonSlug.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "unknown"
	}
	return slug
}
