package go_monobank

// PaymentError returns business-level payment error (if any) extracted from status/webhook payload.
func (r *InvoiceStatusResponse) PaymentError() *PaymentError {
	if r == nil {
		return nil
	}
	var code, reason string
	if r.ErrCode != nil {
		code = *r.ErrCode
	}
	if r.FailureReason != nil {
		reason = *r.FailureReason
	}
	return NewPaymentError(r.InvoiceID, r.Status, code, reason)
}

// PaymentError returns business-level payment error (if any) extracted from wallet/payment response.
// Note: wallet/payment response usually contains failureReason but not errCode.
// For detailed errCode, call Status(...) or rely on webhook payload.
func (r *WalletPaymentResponse) PaymentError() *PaymentError {
	if r == nil {
		return nil
	}
	var reason string
	if r.FailureReason != nil {
		reason = *r.FailureReason
	}
	// No errCode in this response.
	return NewPaymentError(r.InvoiceID, r.Status, "", reason)
}
