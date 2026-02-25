package go_monobank

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stremovskyy/go-monobank/consts"
	internalhttp "github.com/stremovskyy/go-monobank/internal/http"
	"github.com/stremovskyy/go-monobank/log"
)

type client struct {
	http *internalhttp.Client
	cfg  *clientConfig

	pubKeyMu sync.Mutex
	pubKey   *ecdsa.PublicKey
}

var _ Monobank = (*client)(nil)

var logger = log.NewLogger("Monobank:")

func (c *client) SetLogLevel(level log.Level) {
	log.SetLevel(level)
}

// Verification creates an invoice with saveCardData (tokenization).
// Under the hood: POST /api/merchant/invoice/create.
func (c *client) Verification(request *Request, runOpts ...RunOption) (*InvoiceCreateResponse, error) {
	if request == nil {
		return nil, &ValidationError{Op: "verification", Msg: "request is nil"}
	}

	token := c.resolveToken(request)
	if token == "" {
		return nil, &ValidationError{Op: "verification", Msg: "X-Token is required (set request.WithToken(...) or client WithToken(...))"}
	}

	amount := request.GetAmount()
	if amount <= 0 {
		return nil, &ValidationError{Op: "verification", Msg: "amount (minor units) must be > 0"}
	}

	ccy := request.GetCurrency()
	if ccy == 0 {
		ccy = CurrencyUAH
	}

	if request.ShouldSaveCard() {
		walletID := request.GetWalletID()
		if strings.TrimSpace(walletID) == "" {
			return nil, &ValidationError{Op: "verification", Msg: "walletId is required when SaveCard is enabled"}
		}
	}

	payload := mapToInvoiceCreatePayload(request, amount, ccy)

	opts := collectRunOptions(runOpts)
	endpoint := c.cfg.baseURL + consts.PathInvoiceCreate
	if opts.isDryRun() {
		opts.handleDryRun(endpoint, payload)
		return nil, nil
	}

	var resp InvoiceCreateResponse
	if err := c.doJSON(context.Background(), http.MethodPost, consts.PathInvoiceCreate, token, request, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) VerificationLink(request *Request, runOpts ...RunOption) (*url.URL, error) {
	resp, err := c.Verification(request, runOpts...)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	parsed, err := url.Parse(strings.TrimSpace(resp.PageURL))
	if err != nil {
		return nil, fmt.Errorf("verification: cannot parse pageUrl %q: %w", resp.PageURL, err)
	}
	if !parsed.IsAbs() {
		return nil, fmt.Errorf("verification: pageUrl is not absolute: %q", resp.PageURL)
	}
	return parsed, nil
}

// Payment performs a charge by tokenized card.
// Under the hood: POST /api/merchant/wallet/payment.
func (c *client) Payment(request *Request, runOpts ...RunOption) (*WalletPaymentResponse, error) {
	if request == nil {
		return nil, &ValidationError{Op: "payment", Msg: "request is nil"}
	}
	// Token
	token := c.resolveToken(request)
	if token == "" {
		return nil, &ValidationError{Op: "payment", Msg: "X-Token is required (set request.WithToken(...) or client WithToken(...))"}
	}

	cardToken := request.GetCardToken()
	if cardToken == "" {
		return nil, &ValidationError{Op: "payment", Msg: "cardToken is required (set request.WithCardToken(...))"}
	}

	amount := request.GetAmount()
	if amount <= 0 {
		return nil, &ValidationError{Op: "payment", Msg: "amount (minor units) must be > 0"}
	}

	ccy := request.GetCurrency()
	if ccy == 0 {
		ccy = CurrencyUAH
	}

	initKind := request.GetInitiationKind()
	if strings.TrimSpace(string(initKind)) == "" {
		return nil, &ValidationError{Op: "payment", Msg: "initiationKind is required (merchant|client)"}
	}
	if initKind == InitiationClient {
		// docs: redirectUrl is required when initiationKind=client
		if request.GetRedirectURL() == nil || strings.TrimSpace(*request.GetRedirectURL()) == "" {
			return nil, &ValidationError{Op: "payment", Msg: "redirectUrl is required when initiationKind=client"}
		}
	}

	payload := mapToWalletPaymentPayload(request, cardToken, amount, ccy, initKind)

	opts := collectRunOptions(runOpts)
	endpoint := c.cfg.baseURL + consts.PathWalletPayment
	if opts.isDryRun() {
		opts.handleDryRun(endpoint, payload)
		return nil, nil
	}

	var resp WalletPaymentResponse
	if err := c.doJSON(context.Background(), http.MethodPost, consts.PathWalletPayment, token, request, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Status returns invoice status.
// Under the hood: GET /api/merchant/invoice/status?invoiceId=...
func (c *client) Status(request *Request, runOpts ...RunOption) (*InvoiceStatusResponse, error) {
	if request == nil {
		return nil, &ValidationError{Op: "status", Msg: "request is nil"}
	}
	// Token
	token := c.resolveToken(request)
	if token == "" {
		return nil, &ValidationError{Op: "status", Msg: "X-Token is required (set request.WithToken(...) or client WithToken(...))"}
	}

	invoiceID := request.GetInvoiceID()
	if invoiceID == "" {
		return nil, &ValidationError{Op: "status", Msg: "invoiceId is required (set request.WithInvoiceID(...))"}
	}

	opts := collectRunOptions(runOpts)
	endpoint := c.cfg.baseURL + consts.PathInvoiceStatus + "?invoiceId=" + url.QueryEscape(invoiceID)
	if opts.isDryRun() {
		opts.handleDryRun(endpoint, map[string]string{"invoiceId": invoiceID})
		return nil, nil
	}

	var resp InvoiceStatusResponse
	if err := c.doJSON(context.Background(), http.MethodGet, consts.PathInvoiceStatus+"?invoiceId="+url.QueryEscape(invoiceID), token, request, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PublicKey fetches pubkey (base64-encoded PEM) used for webhook signature verification.
func (c *client) PublicKey(request *Request, runOpts ...RunOption) (*PublicKeyResponse, error) {
	if request == nil {
		request = &Request{}
	}
	// Token
	token := c.resolveToken(request)
	if token == "" {
		return nil, &ValidationError{Op: "pubkey", Msg: "X-Token is required (set request.WithToken(...) or client WithToken(...))"}
	}

	opts := collectRunOptions(runOpts)
	endpoint := c.cfg.baseURL + consts.PathPubKey
	if opts.isDryRun() {
		opts.handleDryRun(endpoint, nil)
		return nil, nil
	}

	var resp PublicKeyResponse
	if err := c.doJSON(context.Background(), http.MethodGet, consts.PathPubKey, token, request, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) ParseWebhook(body []byte) (*InvoiceStatusResponse, error) {
	logger.Debug("Webhook parse: body_size=%d", len(body))
	if len(body) == 0 {
		logger.Error("Webhook parse: body is empty")
		return nil, &ValidationError{Op: "webhook", Msg: "body is empty"}
	}
	var event InvoiceStatusResponse
	if err := json.Unmarshal(body, &event); err != nil {
		logger.Error("Webhook parse: decode error: %v", err)
		return nil, &DecodeError{Op: "webhook", Msg: "json unmarshal", Body: trimBody(body, 4096), Cause: err}
	}
	logger.Info("Webhook parse: status=%s invoice_id=%s", event.Status, event.InvoiceID)
	return &event, nil
}

func (c *client) ParseAndVerifyWebhook(body []byte, xSign string) (*InvoiceStatusResponse, error) {
	if err := c.VerifyWebhook(body, xSign); err != nil {
		return nil, err
	}
	return c.ParseWebhook(body)
}

func (c *client) VerifyWebhook(body []byte, xSign string) error {
	logger.Debug("Webhook verify: body_size=%d", len(body))
	if len(body) == 0 {
		logger.Error("Webhook verify: body is empty")
		return &ValidationError{Op: "verify", Msg: "body is empty"}
	}
	xSign = strings.TrimSpace(xSign)
	if xSign == "" {
		logger.Error("Webhook verify: X-Sign header is empty")
		return &ValidationError{Op: "verify", Msg: "X-Sign header is empty"}
	}

	pub, err := c.ensureWebhookPublicKey(context.Background())
	if err != nil {
		logger.Error("Webhook verify: cannot resolve public key: %v", err)
		return err
	}

	sig, err := base64.StdEncoding.DecodeString(xSign)
	if err != nil {
		logger.Error("Webhook verify: X-Sign decode error: %v", err)
		return &DecodeError{Op: "verify", Msg: "base64 decode X-Sign", Cause: err}
	}

	h := sha256.Sum256(body)
	if !ecdsa.VerifyASN1(pub, h[:], sig) {
		logger.Warn("Webhook verify: invalid signature")
		return &WebhookSignatureError{Op: "verify", Msg: "invalid signature"}
	}
	logger.Info("Webhook verify: signature is valid")
	return nil
}

// --- internal helpers ---

func (c *client) resolveToken(request *Request) string {
	if request != nil {
		if t := strings.TrimSpace(request.GetToken()); t != "" {
			return t
		}
	}
	if c == nil || c.cfg == nil {
		return ""
	}
	return strings.TrimSpace(c.cfg.defaultToken)
}

func (c *client) doJSON(ctx context.Context, method, path string, token string, request *Request, payload any, out any) error {
	// Base URL comes from client config (WithBaseURL). If it's empty, fall back to default.
	baseURL := ""
	if c != nil && c.cfg != nil {
		baseURL = strings.TrimRight(strings.TrimSpace(c.cfg.baseURL), "/")
	}
	if baseURL == "" {
		baseURL = strings.TrimRight(consts.DefaultBaseURL, "/")
	}
	endpoint := baseURL + path
	logger.Info("HTTP request: method=%s path=%s", method, path)
	logger.Debug("HTTP request: endpoint=%s", endpoint)
	if payload == nil {
		logger.Debug("HTTP request: payload=<nil>")
	} else if body, err := json.Marshal(payload); err != nil {
		logger.Debug("HTTP request: payload marshal error for %T: %v", payload, err)
	} else {
		logger.Debug("HTTP request: payload=%s", trimBody(body, 4096))
	}

	// Token precedence: explicit arg > request token > client default token.
	tok := strings.TrimSpace(token)
	if tok == "" && request != nil {
		tok = strings.TrimSpace(request.GetToken())
	}
	if tok == "" && c != nil && c.cfg != nil {
		tok = strings.TrimSpace(c.cfg.defaultToken)
	}
	if tok == "" {
		logger.Error("HTTP request: token is empty for method=%s path=%s", method, path)
		return &ValidationError{Op: "auth", Msg: "token is empty"}
	}

	// Apply timeout from client config. If timeout is 0, internalhttp.WithTimeout returns original ctx.
	timeout := time.Duration(0)
	if c != nil && c.cfg != nil && c.cfg.httpOptions != nil {
		timeout = c.cfg.httpOptions.Timeout
	}
	ctx, cancel := internalhttp.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := internalhttp.NewJSONRequest(ctx, method, endpoint, payload)
	if err != nil {
		logger.Error("HTTP request: cannot build request method=%s path=%s err=%v", method, path, err)
		return &EncodeError{Op: "request", Msg: "build json request", Cause: err}
	}

	// Headers
	req.Header.Set("X-Token", tok)
	if request != nil && request.Merchant != nil {
		if request.Merchant.CMS != nil && strings.TrimSpace(*request.Merchant.CMS) != "" {
			req.Header.Set("X-Cms", strings.TrimSpace(*request.Merchant.CMS))
		}
		if request.Merchant.CMSVersion != nil && strings.TrimSpace(*request.Merchant.CMSVersion) != "" {
			req.Header.Set("X-Cms-Version", strings.TrimSpace(*request.Merchant.CMSVersion))
		}
	}

	if c == nil || c.http == nil {
		logger.Error("HTTP request: http client is nil method=%s path=%s", method, path)
		return &UnexpectedResponseError{Op: "client", Method: method, Endpoint: path, Msg: "http client is nil"}
	}

	resp, body, err := c.http.Do(req)
	if err != nil {
		logger.Error("HTTP request: transport error method=%s path=%s err=%v", method, path, err)
		return &TransportError{Op: "http.do", Method: method, URL: endpoint, Cause: err}
	}
	if resp == nil {
		logger.Error("HTTP request: nil response method=%s path=%s", method, path)
		return &UnexpectedResponseError{Op: "http.do", Method: method, Endpoint: path, Msg: "response is nil"}
	}
	logger.Info("HTTP response: method=%s path=%s status=%d", method, path, resp.StatusCode)
	logger.Debug("HTTP response: method=%s path=%s body=%s", method, path, trimBody(body, 4096))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errCode, desc := parseAPIErrorBody(body)

		apiErr := &APIError{
			Kind:        kindFromStatus(resp.StatusCode),
			Method:      method,
			Endpoint:    path,
			StatusCode:  resp.StatusCode,
			ContentType: resp.Header.Get("Content-Type"),
			ErrCode:     errCode,
			Description: desc,
			Body:        trimBody(body, 4096),
		}

		if resp.StatusCode == 429 {
			if d, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
				apiErr.RetryAfter = &d
			}
		}

		if resp.StatusCode >= 500 {
			logger.Error(
				"HTTP response: non-2xx method=%s path=%s status=%d err_code=%s description=%s",
				method,
				path,
				resp.StatusCode,
				errCode,
				desc,
			)
		} else {
			logger.Warn(
				"HTTP response: non-2xx method=%s path=%s status=%d err_code=%s description=%s",
				method,
				path,
				resp.StatusCode,
				errCode,
				desc,
			)
		}
		if apiErr.RetryAfter != nil {
			logger.Warn("HTTP response: retry_after=%s for method=%s path=%s", apiErr.RetryAfter.String(), method, path)
		}

		return apiErr
	}

	if out == nil {
		logger.Debug("HTTP response: out target is nil, skipping decode")
		return nil
	}
	if len(body) == 0 {
		logger.Error("HTTP response: empty body method=%s path=%s status=%d", method, path, resp.StatusCode)
		return &UnexpectedResponseError{Op: "decode", Method: method, Endpoint: path, StatusCode: resp.StatusCode, Msg: "empty response body"}
	}
	if err := json.Unmarshal(body, out); err != nil {
		logger.Error("HTTP response: decode error method=%s path=%s err=%v", method, path, err)
		return &DecodeError{Op: "decode", Msg: "json unmarshal response", Body: trimBody(body, 4096), Cause: err}
	}
	logger.Debug("HTTP response: decoded into %T", out)

	return nil
}

type apiErrorBody struct {
	ErrCode          any    `json:"errCode"`
	ErrText          string `json:"errText"`
	ErrorDescription string `json:"errorDescription"`
	Message          string `json:"message"`
	Error            string `json:"error"`
	Description      string `json:"description"`
	Detail           string `json:"detail"`
}

func parseAPIErrorBody(body []byte) (errCode string, desc string) {
	if len(body) == 0 {
		return "", ""
	}

	// Try structured JSON first
	var parsed apiErrorBody
	if err := json.Unmarshal(body, &parsed); err == nil {
		errCode = anyToString(parsed.ErrCode)
		desc = firstNonEmptyString(
			strings.TrimSpace(parsed.ErrText),
			strings.TrimSpace(parsed.ErrorDescription),
			strings.TrimSpace(parsed.Message),
			strings.TrimSpace(parsed.Error),
			strings.TrimSpace(parsed.Description),
			strings.TrimSpace(parsed.Detail),
		)

		// If nothing extracted, fall back to raw string.
		if desc == "" {
			desc = strings.TrimSpace(string(body))
		}
		return errCode, desc
	}

	// Fallback: attempt as generic map
	var m map[string]any
	if err := json.Unmarshal(body, &m); err == nil {
		if v, ok := m["errCode"]; ok {
			errCode = anyToString(v)
		}
		desc = firstNonEmptyString(
			anyToString(m["errText"]),
			anyToString(m["errorDescription"]),
			anyToString(m["message"]),
			anyToString(m["error"]),
			anyToString(m["description"]),
			anyToString(m["detail"]),
		)
		if desc == "" {
			// single-field object fallback
			if len(m) == 1 {
				for _, v := range m {
					desc = anyToString(v)
				}
			}
		}
		if desc == "" {
			desc = strings.TrimSpace(string(body))
		}
		return errCode, strings.TrimSpace(desc)
	}

	// Last resort: raw body as string (could be text/html or plain text)
	return "", strings.TrimSpace(string(body))
}

func parseRetryAfter(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	// Retry-After can be either "delta-seconds" or an HTTP date.
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			secs = 0
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

func anyToString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case float64:
		// JSON numbers are float64 by default.
		// We treat them as int-like where possible.
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%v", x)
	case int:
		return fmt.Sprintf("%d", x)
	case int32:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	case uint:
		return fmt.Sprintf("%d", x)
	case uint32:
		return fmt.Sprintf("%d", x)
	case uint64:
		return fmt.Sprintf("%d", x)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func mapToInvoiceCreatePayload(r *Request, amount int64, ccy CurrencyCode) any {
	payload := struct {
		Amount           int64             `json:"amount"`
		Currency         CurrencyCode      `json:"ccy"`
		MerchantPaymInfo *MerchantPaymInfo `json:"merchantPaymInfo,omitempty"`
		RedirectURL      *string           `json:"redirectUrl,omitempty"`
		WebHookURL       *string           `json:"webHookUrl,omitempty"`
		Validity         *int64            `json:"validity,omitempty"`
		PaymentType      PaymentType       `json:"paymentType,omitempty"`
		SaveCardData     *SaveCardData     `json:"saveCardData,omitempty"`
	}{
		Amount:   amount,
		Currency: ccy,
	}

	if r != nil {
		payload.MerchantPaymInfo = r.GetMerchantPaymInfo()
		payload.RedirectURL = r.GetRedirectURL()
		payload.WebHookURL = r.GetWebHookURL()
		payload.Validity = r.GetValiditySeconds()
		payload.PaymentType = r.GetPaymentType()
		if payload.PaymentType == "" {
			payload.PaymentType = PaymentTypeDebit
		}
		if r.ShouldSaveCard() {
			payload.SaveCardData = &SaveCardData{SaveCard: true, WalletID: r.GetWalletID()}
		}
	}
	return payload
}

func mapToWalletPaymentPayload(r *Request, cardToken string, amount int64, ccy CurrencyCode, kind InitiationKind) any {
	payload := struct {
		CardToken        string            `json:"cardToken"`
		Amount           int64             `json:"amount"`
		Currency         CurrencyCode      `json:"ccy"`
		RedirectURL      *string           `json:"redirectUrl,omitempty"`
		WebHookURL       *string           `json:"webHookUrl,omitempty"`
		InitiationKind   InitiationKind    `json:"initiationKind"`
		MerchantPaymInfo *MerchantPaymInfo `json:"merchantPaymInfo,omitempty"`
		PaymentType      PaymentType       `json:"paymentType,omitempty"`
	}{
		CardToken:      cardToken,
		Amount:         amount,
		Currency:       ccy,
		InitiationKind: kind,
	}

	if r != nil {
		payload.RedirectURL = r.GetRedirectURL()
		payload.WebHookURL = r.GetWebHookURL()
		payload.MerchantPaymInfo = r.GetMerchantPaymInfo()
		payload.PaymentType = r.GetPaymentType()
		if payload.PaymentType == "" {
			payload.PaymentType = PaymentTypeDebit
		}
	}
	return payload
}

func (c *client) ensureWebhookPublicKey(ctx context.Context) (*ecdsa.PublicKey, error) {
	c.pubKeyMu.Lock()
	defer c.pubKeyMu.Unlock()

	if c.pubKey != nil {
		return c.pubKey, nil
	}

	// 1) Raw PEM provided
	if len(c.cfg.webhookPublicKeyPEM) > 0 {
		pub, err := parseECDSAPublicKeyFromPEM(c.cfg.webhookPublicKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("pubkey: parse PEM: %w", err)
		}
		c.pubKey = pub
		return pub, nil
	}

	// 2) Base64 PEM provided
	if strings.TrimSpace(c.cfg.webhookPublicKeyBase64) != "" {
		pemBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(c.cfg.webhookPublicKeyBase64))
		if err != nil {
			return nil, fmt.Errorf("pubkey: base64 decode: %w", err)
		}
		pub, err := parseECDSAPublicKeyFromPEM(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("pubkey: parse decoded PEM: %w", err)
		}
		c.pubKey = pub
		return pub, nil
	}

	// 3) Fetch from API using default token
	token := strings.TrimSpace(c.cfg.defaultToken)
	if token == "" {
		return nil, &ValidationError{Op: "pubkey", Msg: "public key not configured and default token is empty; set WithWebhookPublicKeyBase64(...) or WithToken(...)"}
	}
	var resp PublicKeyResponse
	if err := c.doJSON(ctx, http.MethodGet, consts.PathPubKey, token, nil, nil, &resp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.Key) == "" {
		return nil, fmt.Errorf("pubkey: empty key in response")
	}
	pemBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(resp.Key))
	if err != nil {
		return nil, fmt.Errorf("pubkey: base64 decode fetched key: %w", err)
	}
	pub, err := parseECDSAPublicKeyFromPEM(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("pubkey: parse fetched PEM: %w", err)
	}
	c.cfg.webhookPublicKeyBase64 = strings.TrimSpace(resp.Key)
	c.pubKey = pub
	return pub, nil
}

func parseECDSAPublicKeyFromPEM(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unexpected public key type %T (expected ECDSA)", pubAny)
	}
	return pub, nil
}
