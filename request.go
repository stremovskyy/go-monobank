package go_monobank

import "strings"

// Request is a unified request object for common monobank acquiring flows.
// It is intentionally similar to go-ipay/go-platon style (Merchant + PaymentData + PaymentMethod)
// but exposes fluent/chain setters.
//
// This request is used by:
//   - Verification / VerificationLink (invoice/create + saveCardData)
//   - Status (invoice/status)
//   - Payment (wallet/payment)
//   - PublicKey (pubkey)
type Request struct {
	Merchant      *Merchant
	PaymentData   *PaymentData
	PaymentMethod *PaymentMethod
}

type Merchant struct {
	Token      string
	CMS        *string
	CMSVersion *string
}

type PaymentData struct {
	InvoiceID       *string
	Amount          int64
	Currency        CurrencyCode
	PaymentType     PaymentType
	RedirectURL     *string
	WebHookURL      *string
	ValiditySeconds *int64
	InitiationKind  InitiationKind

	MerchantPaymInfo *MerchantPaymInfo
}

type PaymentMethod struct {
	CardToken *string

	// WalletID is a merchant-defined identifier for a customer wallet.
	// Used for card tokenization (saveCardData.walletId)
	WalletID *string

	// SaveCard enables tokenization for invoice/create.
	SaveCard bool
}

func NewRequest() *Request { return &Request{} }

func (r *Request) ensureMerchant() *Merchant {
	if r.Merchant == nil {
		r.Merchant = &Merchant{}
	}
	return r.Merchant
}

func (r *Request) ensurePaymentData() *PaymentData {
	if r.PaymentData == nil {
		r.PaymentData = &PaymentData{}
	}
	return r.PaymentData
}

func (r *Request) ensurePaymentMethod() *PaymentMethod {
	if r.PaymentMethod == nil {
		r.PaymentMethod = &PaymentMethod{}
	}
	return r.PaymentMethod
}

// WithToken sets X-Token for API calls.
func (r *Request) WithToken(token string) *Request {
	r.ensureMerchant().Token = strings.TrimSpace(token)
	return r
}

func (r *Request) WithCMS(name string) *Request {
	name = strings.TrimSpace(name)
	if name == "" {
		return r
	}
	r.ensureMerchant().CMS = &name
	return r
}

func (r *Request) WithCMSVersion(v string) *Request {
	v = strings.TrimSpace(v)
	if v == "" {
		return r
	}
	r.ensureMerchant().CMSVersion = &v
	return r
}

func (r *Request) WithInvoiceID(invoiceID string) *Request {
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return r
	}
	r.ensurePaymentData().InvoiceID = &invoiceID
	return r
}

func (r *Request) WithAmount(amountMinor int64) *Request {
	r.ensurePaymentData().Amount = amountMinor
	return r
}

func (r *Request) WithCurrency(ccy CurrencyCode) *Request {
	r.ensurePaymentData().Currency = ccy
	return r
}

func (r *Request) WithPaymentType(t PaymentType) *Request {
	r.ensurePaymentData().PaymentType = t
	return r
}

func (r *Request) WithRedirectURL(url string) *Request {
	url = strings.TrimSpace(url)
	if url == "" {
		return r
	}
	r.ensurePaymentData().RedirectURL = &url
	return r
}

func (r *Request) WithWebHookURL(url string) *Request {
	url = strings.TrimSpace(url)
	if url == "" {
		return r
	}
	r.ensurePaymentData().WebHookURL = &url
	return r
}

func (r *Request) WithValiditySeconds(seconds int64) *Request {
	if seconds <= 0 {
		return r
	}
	r.ensurePaymentData().ValiditySeconds = &seconds
	return r
}

func (r *Request) WithInitiationKind(kind InitiationKind) *Request {
	r.ensurePaymentData().InitiationKind = kind
	return r
}

func (r *Request) WithMerchantPaymInfo(info *MerchantPaymInfo) *Request {
	r.ensurePaymentData().MerchantPaymInfo = info
	return r
}

func (r *Request) ensureMerchantPaymInfo() *MerchantPaymInfo {
	pd := r.ensurePaymentData()
	if pd.MerchantPaymInfo == nil {
		pd.MerchantPaymInfo = &MerchantPaymInfo{}
	}
	return pd.MerchantPaymInfo
}

func (r *Request) WithReference(ref string) *Request {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return r
	}
	r.ensureMerchantPaymInfo().Reference = ref
	return r
}

func (r *Request) WithDestination(dest string) *Request {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return r
	}
	r.ensureMerchantPaymInfo().Destination = dest
	return r
}

// SaveCard enables tokenization and sets walletId.
func (r *Request) SaveCard(walletID string) *Request {
	walletID = strings.TrimSpace(walletID)
	if walletID == "" {
		return r
	}
	pm := r.ensurePaymentMethod()
	pm.SaveCard = true
	pm.WalletID = &walletID
	return r
}

func (r *Request) WithCardToken(cardToken string) *Request {
	cardToken = strings.TrimSpace(cardToken)
	if cardToken == "" {
		return r
	}
	r.ensurePaymentMethod().CardToken = &cardToken
	return r
}

// GetToken resolves X-Token from request.
func (r *Request) GetToken() string {
	if r == nil || r.Merchant == nil {
		return ""
	}
	return strings.TrimSpace(r.Merchant.Token)
}

func (r *Request) GetInvoiceID() string {
	if r == nil || r.PaymentData == nil || r.PaymentData.InvoiceID == nil {
		return ""
	}
	return strings.TrimSpace(*r.PaymentData.InvoiceID)
}

func (r *Request) GetAmount() int64 {
	if r == nil || r.PaymentData == nil {
		return 0
	}
	return r.PaymentData.Amount
}

func (r *Request) GetCurrency() CurrencyCode {
	if r == nil || r.PaymentData == nil {
		return 0
	}
	return r.PaymentData.Currency
}

func (r *Request) GetRedirectURL() *string {
	if r == nil || r.PaymentData == nil {
		return nil
	}
	return r.PaymentData.RedirectURL
}

func (r *Request) GetWebHookURL() *string {
	if r == nil || r.PaymentData == nil {
		return nil
	}
	return r.PaymentData.WebHookURL
}

func (r *Request) GetPaymentType() PaymentType {
	if r == nil || r.PaymentData == nil {
		return ""
	}
	return r.PaymentData.PaymentType
}

func (r *Request) GetValiditySeconds() *int64 {
	if r == nil || r.PaymentData == nil {
		return nil
	}
	return r.PaymentData.ValiditySeconds
}

func (r *Request) GetInitiationKind() InitiationKind {
	if r == nil || r.PaymentData == nil {
		return ""
	}
	return r.PaymentData.InitiationKind
}

func (r *Request) GetMerchantPaymInfo() *MerchantPaymInfo {
	if r == nil || r.PaymentData == nil {
		return nil
	}
	return r.PaymentData.MerchantPaymInfo
}

func (r *Request) GetCardToken() string {
	if r == nil || r.PaymentMethod == nil || r.PaymentMethod.CardToken == nil {
		return ""
	}
	return strings.TrimSpace(*r.PaymentMethod.CardToken)
}

func (r *Request) GetWalletID() string {
	if r == nil || r.PaymentMethod == nil || r.PaymentMethod.WalletID == nil {
		return ""
	}
	return strings.TrimSpace(*r.PaymentMethod.WalletID)
}

func (r *Request) ShouldSaveCard() bool {
	if r == nil || r.PaymentMethod == nil {
		return false
	}
	return r.PaymentMethod.SaveCard
}
