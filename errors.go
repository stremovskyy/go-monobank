package go_monobank

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrValidation indicates a client-side validation issue (missing required fields, invalid input).
	ErrValidation = errors.New("monobank: validation error")
	// ErrEncode indicates a client-side encoding issue (failed to marshal JSON, etc).
	ErrEncode = errors.New("monobank: encode error")
	// ErrTransport indicates a network/transport error (timeouts, DNS, TLS, etc).
	ErrTransport = errors.New("monobank: transport error")
	// ErrDecode indicates a client-side decoding issue (failed to unmarshal JSON, empty body, invalid base64, etc).
	ErrDecode = errors.New("monobank: decode error")

	// ErrBadRequest corresponds to HTTP 400 from API.
	ErrBadRequest = errors.New("monobank: bad request")
	// ErrInvalidToken corresponds to HTTP 403 from API (token invalid).
	ErrInvalidToken = errors.New("monobank: invalid token")
	// ErrNotFound corresponds to HTTP 404 from API.
	ErrNotFound = errors.New("monobank: not found")
	// ErrMethodNotAllowed corresponds to HTTP 405 from API.
	ErrMethodNotAllowed = errors.New("monobank: method not allowed")
	// ErrRateLimited corresponds to HTTP 429 from API.
	ErrRateLimited = errors.New("monobank: rate limited")
	// ErrServerError corresponds to HTTP 5xx from API.
	ErrServerError = errors.New("monobank: server error")
	// ErrUnexpectedResponse is returned when API responds in an unexpected way (unknown status code, invalid content).
	ErrUnexpectedResponse = errors.New("monobank: unexpected response")

	// ErrInvalidSignature is returned when webhook signature validation fails (X-Sign).
	ErrInvalidSignature = errors.New("monobank: invalid webhook signature")

	// ErrPaymentError indicates a business/payment failure (errCode/failureReason from webhook/status).
	ErrPaymentError = errors.New("monobank: payment error")
)

// ValidationError indicates that a request is missing required fields or has invalid values.
type ValidationError struct {
	Op    string
	Msg   string
	Cause error
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ErrValidation.Error()
	}
	if e.Op == "" {
		if e.Msg == "" {
			return ErrValidation.Error()
		}
		return fmt.Sprintf("%s: %s", ErrValidation.Error(), e.Msg)
	}
	if e.Msg == "" {
		return fmt.Sprintf("%s: %s", ErrValidation.Error(), e.Op)
	}
	return fmt.Sprintf("%s: %s: %s", ErrValidation.Error(), e.Op, e.Msg)
}

func (e *ValidationError) Unwrap() error { return e.Cause }
func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

// EncodeError indicates a request/response encoding issue (JSON marshal, etc).
type EncodeError struct {
	Op    string
	Msg   string
	Cause error
}

func (e *EncodeError) Error() string {
	if e == nil {
		return ErrEncode.Error()
	}
	base := ErrEncode.Error()
	if e.Op != "" {
		base += ": " + e.Op
	}
	if e.Msg != "" {
		base += ": " + e.Msg
	}
	return base
}

func (e *EncodeError) Unwrap() error { return e.Cause }
func (e *EncodeError) Is(target error) bool {
	return target == ErrEncode
}

// TransportError indicates a networking error while calling the API.
type TransportError struct {
	Op     string
	Method string
	URL    string
	Cause  error
}

func (e *TransportError) Error() string {
	if e == nil {
		return ErrTransport.Error()
	}
	parts := []string{ErrTransport.Error()}
	if e.Op != "" {
		parts = append(parts, e.Op)
	}
	if e.Method != "" && e.URL != "" {
		parts = append(parts, e.Method+" "+e.URL)
	} else if e.URL != "" {
		parts = append(parts, e.URL)
	}
	if e.Cause != nil {
		parts = append(parts, e.Cause.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *TransportError) Unwrap() error { return e.Cause }
func (e *TransportError) Is(target error) bool {
	return target == ErrTransport
}

// DecodeError indicates an issue decoding API/webhook response, signature header, etc.
type DecodeError struct {
	Op    string
	Msg   string
	Body  []byte
	Cause error
}

func (e *DecodeError) Error() string {
	if e == nil {
		return ErrDecode.Error()
	}
	base := ErrDecode.Error()
	if e.Op != "" {
		base += ": " + e.Op
	}
	if e.Msg != "" {
		base += ": " + e.Msg
	}
	if e.Cause != nil {
		base += ": " + e.Cause.Error()
	}
	return base
}

func (e *DecodeError) Unwrap() error { return e.Cause }
func (e *DecodeError) Is(target error) bool {
	return target == ErrDecode
}

// UnexpectedResponseError indicates an API response that doesn't match expectations (e.g., empty body).
type UnexpectedResponseError struct {
	Op         string
	Method     string
	Endpoint   string
	StatusCode int
	Msg        string
	Body       []byte
}

func (e *UnexpectedResponseError) Error() string {
	if e == nil {
		return ErrUnexpectedResponse.Error()
	}
	base := ErrUnexpectedResponse.Error()
	if e.Op != "" {
		base += ": " + e.Op
	}
	if e.Method != "" && e.Endpoint != "" {
		base += ": " + e.Method + " " + e.Endpoint
	} else if e.Endpoint != "" {
		base += ": " + e.Endpoint
	}
	if e.StatusCode != 0 {
		base += fmt.Sprintf(": status=%d", e.StatusCode)
	}
	if e.Msg != "" {
		base += ": " + e.Msg
	}
	return base
}

func (e *UnexpectedResponseError) Is(target error) bool { return target == ErrUnexpectedResponse }

// APIError represents a non-2xx response from monobank API.
type APIError struct {
	// Kind classifies the error for errors.Is(...) matching (ErrBadRequest, ErrInvalidToken, ...).
	Kind error

	Method   string
	Endpoint string

	StatusCode  int
	ContentType string

	// ErrCode/Description are best-effort parsed from the response body.
	ErrCode     string
	Description string

	// Body is a truncated response body (best-effort).
	Body []byte

	// RetryAfter is best-effort parsed from Retry-After header when status=429.
	RetryAfter *time.Duration
}

func (e *APIError) Error() string {
	if e == nil {
		return "api error"
	}

	kind := "api error"
	if e.Kind != nil {
		kind = e.Kind.Error()
	}

	msg := fmt.Sprintf("%s: status=%d", kind, e.StatusCode)
	if e.Method != "" && e.Endpoint != "" {
		msg += " " + e.Method + " " + e.Endpoint
	} else if e.Endpoint != "" {
		msg += " endpoint=" + e.Endpoint
	}

	if strings.TrimSpace(e.ErrCode) != "" {
		msg += " errCode=" + strings.TrimSpace(e.ErrCode)
	}
	if strings.TrimSpace(e.Description) != "" {
		msg += " desc=" + strings.TrimSpace(e.Description)
	}

	if e.RetryAfter != nil {
		msg += fmt.Sprintf(" retryAfter=%s", e.RetryAfter.String())
	}

	return msg
}

func (e *APIError) Is(target error) bool {
	if e == nil {
		return false
	}
	if e.Kind != nil && target == e.Kind {
		return true
	}
	// Convenience: allow matching generic unexpected response
	if target == ErrUnexpectedResponse {
		return e.Kind == nil
	}
	return false
}

// WebhookSignatureError indicates that webhook signature verification failed.
type WebhookSignatureError struct {
	Op    string
	Msg   string
	Cause error
}

func (e *WebhookSignatureError) Error() string {
	if e == nil {
		return ErrInvalidSignature.Error()
	}
	base := ErrInvalidSignature.Error()
	if e.Op != "" {
		base += ": " + e.Op
	}
	if e.Msg != "" {
		base += ": " + e.Msg
	}
	if e.Cause != nil {
		base += ": " + e.Cause.Error()
	}
	return base
}

func (e *WebhookSignatureError) Unwrap() error { return e.Cause }
func (e *WebhookSignatureError) Is(target error) bool {
	return target == ErrInvalidSignature
}

// --- internal helpers ---

func kindFromStatus(status int) error {
	switch status {
	case 400:
		return ErrBadRequest
	case 403:
		return ErrInvalidToken
	case 404:
		return ErrNotFound
	case 405:
		return ErrMethodNotAllowed
	case 429:
		return ErrRateLimited
	default:
		if status >= 500 && status <= 599 {
			return ErrServerError
		}
		return ErrUnexpectedResponse
	}
}

func trimBody(b []byte, max int) []byte {
	if max <= 0 || len(b) <= max {
		return b
	}
	out := make([]byte, max)
	copy(out, b[:max])
	return out
}
