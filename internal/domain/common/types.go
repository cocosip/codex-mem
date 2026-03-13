// Package common provides shared error, warning, time, and identifier helpers.
package common

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	// WarnScopeAmbiguous indicates scope resolution succeeded with ambiguous identity evidence.
	WarnScopeAmbiguous = "WARN_SCOPE_AMBIGUOUS"
	// WarnScopeFallback indicates weaker scope resolution evidence was used.
	WarnScopeFallback = "WARN_SCOPE_FALLBACK_USED"
	// WarnDedupeApplied indicates duplicate results were suppressed before returning.
	WarnDedupeApplied = "WARN_DEDUPE_APPLIED"
	// WarnHandoffSparse indicates a handoff was stored with minimal detail.
	WarnHandoffSparse = "WARN_HANDOFF_SPARSE"
	// WarnNoPriorHandoff indicates bootstrap found no prior open handoff.
	WarnNoPriorHandoff = "WARN_NO_PRIOR_HANDOFF"
	// WarnNoPriorNotes indicates bootstrap found no recent durable notes.
	WarnNoPriorNotes = "WARN_NO_PRIOR_NOTES"
	// WarnRelatedProjectsSkipped indicates related-project retrieval was requested but skipped.
	WarnRelatedProjectsSkipped = "WARN_RELATED_PROJECTS_SKIPPED"
	// WarnRelatedProjectsEmpty indicates related-project retrieval ran but returned no results.
	WarnRelatedProjectsEmpty = "WARN_RELATED_PROJECTS_EMPTY"
	// WarnRecoveryHandoffUsed indicates bootstrap fell back to a recovery handoff.
	WarnRecoveryHandoffUsed = "WARN_RECOVERY_HANDOFF_USED"
	// WarnRelatedRefIgnored indicates a related-project reference could not be used.
	WarnRelatedRefIgnored = "WARN_RELATED_REFERENCE_IGNORED"
	// WarnExistingAgentsSkipped indicates AGENTS writing was skipped because content already existed.
	WarnExistingAgentsSkipped = "WARN_EXISTING_AGENTS_SKIPPED"
	// WarnPlaceholdersUnresolved indicates AGENTS output still contains manual placeholders.
	WarnPlaceholdersUnresolved = "WARN_PLACEHOLDERS_UNRESOLVED"
	// WarnImportSuppressed indicates an import was suppressed due to dedupe or policy.
	WarnImportSuppressed = "WARN_IMPORT_SUPPRESSED"
	// ErrInvalidInput indicates caller input failed validation.
	ErrInvalidInput = "ERR_INVALID_INPUT"
	// ErrInvalidScope indicates the supplied scope is missing required identity.
	ErrInvalidScope = "ERR_INVALID_SCOPE"
	// ErrScopeConflict indicates canonical identity collides with existing scope data.
	ErrScopeConflict = "ERR_SCOPE_CONFLICT"
	// ErrInvalidState indicates a lifecycle state is not allowed.
	ErrInvalidState = "ERR_INVALID_STATE"
	// ErrSessionNotFound indicates a referenced session record does not exist.
	ErrSessionNotFound = "ERR_SESSION_NOT_FOUND"
	// ErrRecordNotFound indicates a durable record could not be found by id.
	ErrRecordNotFound = "ERR_RECORD_NOT_FOUND"
	// ErrStorageUnavailable indicates the backing store could not be reached.
	ErrStorageUnavailable = "ERR_STORAGE_UNAVAILABLE"
	// ErrWriteFailed indicates a durable write operation failed.
	ErrWriteFailed = "ERR_WRITE_FAILED"
	// ErrReadFailed indicates a durable read operation failed.
	ErrReadFailed = "ERR_READ_FAILED"
	// ErrAgentsWriteDenied indicates AGENTS output could not be written to disk.
	ErrAgentsWriteDenied = "ERR_AGENTS_WRITE_DENIED"
	// ErrInvalidTarget indicates a target or mode selection is unsupported.
	ErrInvalidTarget = "ERR_INVALID_TARGET"
)

// Warning represents a non-fatal condition returned alongside successful results.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CodedError wraps an error with a stable machine-readable code and human-readable message.
type CodedError struct {
	Code    string
	Message string
	Err     error
}

// ErrorPayload is the stable error shape returned to transport callers.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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

// NewError constructs a coded error without a wrapped cause.
func NewError(code, message string) error {
	return &CodedError{Code: code, Message: message}
}

// WrapError constructs a coded error that wraps an underlying cause.
func WrapError(code, message string, err error) error {
	return &CodedError{Code: code, Message: message, Err: err}
}

// ErrorCode extracts the stable code from a coded error.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}

	var coded *CodedError
	if errors.As(err, &coded) {
		return coded.Code
	}
	return ""
}

// ErrorMessage extracts the stable message from a coded error when available.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	var coded *CodedError
	if errors.As(err, &coded) && strings.TrimSpace(coded.Message) != "" {
		return coded.Message
	}
	return err.Error()
}

// ErrorDetails returns a transport-safe error payload with fallback values when needed.
func ErrorDetails(err error, fallbackCode string, fallbackMessage string) ErrorPayload {
	if err == nil {
		return ErrorPayload{}
	}

	code := strings.TrimSpace(ErrorCode(err))
	if code == "" {
		code = strings.TrimSpace(fallbackCode)
	}
	message := strings.TrimSpace(ErrorMessage(err))
	if message == "" {
		message = strings.TrimSpace(fallbackMessage)
	}
	if message == "" {
		message = "operation failed"
	}
	return ErrorPayload{
		Code:    code,
		Message: message,
	}
}

// EnsureCoded wraps err with the fallback code and message when it is not already coded.
func EnsureCoded(err error, fallbackCode string, fallbackMessage string) error {
	if err == nil || strings.TrimSpace(ErrorCode(err)) != "" {
		return err
	}
	message := strings.TrimSpace(fallbackMessage)
	if message == "" {
		message = "operation failed"
	}
	return WrapError(fallbackCode, message, err)
}

// MergeWarnings deduplicates warnings while preserving input order.
func MergeWarnings(groups ...[]Warning) []Warning {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	if total == 0 {
		return nil
	}

	seen := make(map[string]struct{}, total)
	merged := make([]Warning, 0, total)
	for _, group := range groups {
		for _, warning := range group {
			key := strings.TrimSpace(warning.Code) + "\x00" + strings.TrimSpace(warning.Message)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, warning)
		}
	}
	return merged
}

// Clock abstracts access to the current time.
type Clock interface {
	Now() time.Time
}

// RealClock returns the current UTC time.
type RealClock struct{}

// Now returns the current UTC timestamp.
func (RealClock) Now() time.Time {
	return time.Now().UTC()
}

// IDFactory produces record identifiers with a caller-provided prefix.
type IDFactory interface {
	New(prefix string) string
}

// DefaultIDFactory builds time-based identifiers with a random suffix.
type DefaultIDFactory struct {
	Clock Clock
}

// New returns a new identifier for the given prefix.
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

// StableID returns a deterministic identifier derived from prefix and key.
func StableID(prefix, key string) string {
	hash := sha1.Sum([]byte(strings.TrimSpace(key)))
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(hash[:6]))
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slug normalizes free-form text into a lowercase dash-separated slug.
func Slug(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = nonSlug.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "unknown"
	}
	return slug
}
