package go_monobank

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var errATokenNotSet = errors.New("aToken is not set")

// NewApplePayMethod builds an Apple Pay payment method from any supported
// backend payload shape:
// - raw JSON string of ApplePayPaymentToken
// - raw JSON string of full ApplePayPayment object
// - legacy base64-encoded Apple payload
func NewApplePayMethod(payload string) (*PaymentMethod, error) {
	token, err := normalizeApplePayPayload(payload)
	if err != nil {
		return nil, err
	}

	return &PaymentMethod{ApplePayToken: &token}, nil
}

// NewGooglePayMethod builds a Google Pay payment method from any supported
// backend payload shape:
// - raw tokenizationData.token string
// - raw JSON string of full Google Pay PaymentData payload
// - legacy base64-encoded Google payload
func NewGooglePayMethod(payload string) (*PaymentMethod, error) {
	token, err := normalizeGooglePayPayload(payload)
	if err != nil {
		return nil, err
	}

	return &PaymentMethod{GooglePayToken: &token}, nil
}

func (r *Request) IsApplePay() bool {
	if r == nil || r.PaymentMethod == nil {
		return false
	}

	return firstNonEmptyStringPtr(
		r.PaymentMethod.ApplePayToken,
		r.PaymentMethod.ApplePayPayment,
		r.PaymentMethod.AppleContainer,
	) != nil
}

func (r *Request) IsGooglePay() bool {
	if r == nil || r.PaymentMethod == nil {
		return false
	}

	return firstNonEmptyStringPtr(
		r.PaymentMethod.GooglePayToken,
		r.PaymentMethod.GooglePayPaymentData,
		r.PaymentMethod.GoogleToken,
	) != nil
}

func (r *Request) GetAToken() (string, error) {
	if r == nil {
		return "", fmt.Errorf("request is nil")
	}
	if r.PaymentMethod == nil {
		return "", errATokenNotSet
	}

	direct := firstNonEmptyStringPtr(r.PaymentMethod.AToken)
	apple := firstNonEmptyStringPtr(
		r.PaymentMethod.ApplePayToken,
		r.PaymentMethod.ApplePayPayment,
		r.PaymentMethod.AppleContainer,
	)
	google := firstNonEmptyStringPtr(
		r.PaymentMethod.GooglePayToken,
		r.PaymentMethod.GooglePayPaymentData,
		r.PaymentMethod.GoogleToken,
	)

	sources := 0
	for _, source := range []*string{direct, apple, google} {
		if source != nil {
			sources++
		}
	}
	if sources == 0 {
		return "", errATokenNotSet
	}
	if sources > 1 {
		return "", fmt.Errorf("multiple wallet token sources are set")
	}

	switch {
	case direct != nil:
		return *direct, nil
	case apple != nil:
		token, err := normalizeApplePayPayload(*apple)
		if err != nil {
			return "", fmt.Errorf("cannot normalize Apple Pay payload: %w", err)
		}
		return token, nil
	case google != nil:
		token, err := normalizeGooglePayPayload(*google)
		if err != nil {
			return "", fmt.Errorf("cannot normalize Google Pay payload: %w", err)
		}
		return token, nil
	default:
		return "", errATokenNotSet
	}
}

func firstNonEmptyStringPtr(values ...*string) *string {
	for _, value := range values {
		if value == nil {
			continue
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed == "" {
			continue
		}
		return &trimmed
	}
	return nil
}

func decodeMaybeBase64JSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return trimmed
	}

	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}

	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(trimmed)
		if err != nil {
			continue
		}
		decoded = []byte(strings.TrimSpace(string(decoded)))
		if json.Valid(decoded) {
			return string(decoded)
		}
	}

	return trimmed
}

func unwrapJSONString(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return trimmed
	}

	var unwrapped string
	if err := json.Unmarshal([]byte(trimmed), &unwrapped); err == nil {
		unwrapped = strings.TrimSpace(unwrapped)
		if unwrapped != "" {
			return unwrapped
		}
	}

	return trimmed
}

func normalizeApplePayPayload(raw string) (string, error) {
	normalized := unwrapJSONString(decodeMaybeBase64JSON(raw))
	if !json.Valid([]byte(normalized)) {
		return "", fmt.Errorf("apple pay payload must be valid JSON or base64-encoded JSON")
	}

	var payment struct {
		Token json.RawMessage `json:"token"`
	}
	if err := json.Unmarshal([]byte(normalized), &payment); err == nil {
		token := strings.TrimSpace(string(payment.Token))
		if token != "" && token != "null" {
			return token, nil
		}
	}

	return normalized, nil
}

func normalizeGooglePayPayload(raw string) (string, error) {
	normalized := unwrapJSONString(decodeMaybeBase64JSON(raw))
	if !json.Valid([]byte(normalized)) {
		return normalizeGooglePayTokenString(normalized)
	}

	var paymentData struct {
		PaymentMethodData struct {
			TokenizationData struct {
				Token string `json:"token"`
			} `json:"tokenizationData"`
		} `json:"paymentMethodData"`
	}
	if err := json.Unmarshal([]byte(normalized), &paymentData); err == nil {
		token := strings.TrimSpace(paymentData.PaymentMethodData.TokenizationData.Token)
		if token != "" {
			return normalizeGooglePayTokenString(token)
		}
	}

	return normalizeGooglePayTokenString(normalized)
}

func normalizeGooglePayTokenString(raw string) (string, error) {
	token := unwrapJSONString(strings.TrimSpace(raw))
	if token == "" {
		return "", fmt.Errorf("google pay tokenizationData.token is empty")
	}
	if json.Valid([]byte(token)) {
		return token, nil
	}

	unescaped, err := strconv.Unquote(fmt.Sprintf("%q", token))
	if err == nil {
		unescaped = unwrapJSONString(strings.TrimSpace(unescaped))
		if json.Valid([]byte(unescaped)) {
			return unescaped, nil
		}
	}

	return token, nil
}
