package go_monobank

import (
	"fmt"
	"strings"
)

// PaymentErrorMeta is a human-friendly description for a payment errCode.
// The values are taken from monobank acquiring docs ("Помилки в процесі оплати").
// Note: failureReason from API/webhook is still the most precise explanation.
type PaymentErrorMeta struct {
	Code    string
	Text    string
	Contact string
}

// PaymentError is a business-level error parsed from webhook/status response fields:
//   - errCode
//   - failureReason
//
// It is NOT an HTTP/transport error; those are represented by APIError/TransportError/etc.
type PaymentError struct {
	InvoiceID     string
	Status        InvoiceStatus
	ErrCode       string
	FailureReason string

	// Metas are best-effort lookup results from PaymentErrorCatalog by ErrCode.
	// Some codes are duplicated in the docs, so we keep a slice.
	Metas []PaymentErrorMeta
}

func (e *PaymentError) Error() string {
	if e == nil {
		return ErrPaymentError.Error()
	}

	parts := []string{ErrPaymentError.Error()}

	if strings.TrimSpace(e.InvoiceID) != "" {
		parts = append(parts, "invoiceId="+strings.TrimSpace(e.InvoiceID))
	}
	if strings.TrimSpace(string(e.Status)) != "" {
		parts = append(parts, "status="+strings.TrimSpace(string(e.Status)))
	}
	if strings.TrimSpace(e.ErrCode) != "" {
		parts = append(parts, "errCode="+strings.TrimSpace(e.ErrCode))
	}
	if strings.TrimSpace(e.FailureReason) != "" {
		parts = append(parts, "reason="+strings.TrimSpace(e.FailureReason))
	}

	// If we have at least one meta, add one-line hint (without duplicating too much).
	if len(e.Metas) == 1 {
		if strings.TrimSpace(e.Metas[0].Contact) != "" {
			parts = append(parts, "contact="+strings.TrimSpace(e.Metas[0].Contact))
		}
	} else if len(e.Metas) > 1 {
		parts = append(parts, fmt.Sprintf("contact=%d-options", len(e.Metas)))
	}

	return strings.Join(parts, " ")
}

func (e *PaymentError) Is(target error) bool {
	return target == ErrPaymentError
}

// PaymentErrorCatalog maps errCode -> one or more possible meta descriptions.
// Source: monobank acquiring docs page "Помилки в процесі оплати".
var PaymentErrorCatalog = map[string][]PaymentErrorMeta{
	"6":    {{Code: "6", Text: "Операцію заблоковано банком-емітентом", Contact: "банк, який випустив картку"}},
	"40":   {{Code: "40", Text: "Карта втрачена. Витрати обмежені", Contact: "банк, який випустив картку"}},
	"41":   {{Code: "41", Text: "Карта втрачена. Витрати обмежені", Contact: "банк, який випустив картку"}},
	"50":   {{Code: "50", Text: "Витрати по карті обмежені", Contact: "банк, який випустив картку"}},
	"51":   {{Code: "51", Text: "Закінчився строк дії картки", Contact: "банк, який випустив картку"}},
	"52":   {{Code: "52", Text: "Номер картки вказано невірно", Contact: "банк, який випустив картку"}},
	"54":   {{Code: "54", Text: "Стався технічний збій", Contact: "банк, який випустив картку"}},
	"55":   {{Code: "55", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"56":   {{Code: "56", Text: "Тип карти не підтримує подібні оплати", Contact: "банк, який випустив картку"}},
	"57":   {{Code: "57", Text: "Транзакція не підтримується", Contact: "банк, який випустив картку"}, {Code: "57", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"58":   {{Code: "58", Text: "Витрати по карті обмежені на покупку", Contact: "банк, який випустив картку"}, {Code: "58", Text: "Витрати по карті обмежені", Contact: "банк, який випустив картку"}},
	"59":   {{Code: "59", Text: "На картці недостатньо коштів для завершення покупки", Contact: "банк, який випустив картку"}},
	"60":   {{Code: "60", Text: "Перевищено ліміт кількості видаткових операцій", Contact: "банк, який випустив картку"}},
	"61":   {{Code: "61", Text: "На картці перевищено інтернет-ліміт", Contact: "банк, який випустив картку"}},
	"62":   {{Code: "62", Text: "Перевищено ліміт неправильних вводів PIN-коду", Contact: "банк, який випустив картку"}},
	"63":   {{Code: "63", Text: "На картці перевищено інтернет-ліміт", Contact: "банк, який випустив картку"}},
	"67":   {{Code: "67", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"68":   {{Code: "68", Text: "Відмова в проведенні операції з боку МПС", Contact: "банк, який випустив картку"}},
	"71":   {{Code: "71", Text: "Операцію заблоковано банком-емітентом", Contact: "банк, який випустив картку"}},
	"72":   {{Code: "72", Text: "Операцію заблоковано банком-емітентом", Contact: "банк, який випустив картку"}},
	"73":   {{Code: "73", Text: "Помилка маршрутизації", Contact: "monobank"}},
	"74":   {{Code: "74", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"75":   {{Code: "75", Text: "Операцію заблоковано банком-емітентом", Contact: "банк, який випустив картку"}},
	"80":   {{Code: "80", Text: "Неправильний CVV код", Contact: "банк, який випустив картку"}},
	"81":   {{Code: "81", Text: "Неправильний CVV2 код", Contact: "банк, який випустив картку"}},
	"82":   {{Code: "82", Text: "Транзакція не дозволена з такими умовами проведення", Contact: "банк, який випустив картку"}, {Code: "82", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"83":   {{Code: "83", Text: "Перевищені ліміти спроб оплати з карт", Contact: "банк, який випустив картку"}},
	"84":   {{Code: "84", Text: "Неправильне значення перевірочного числа 3D Secure", Contact: "monobank"}},
	"98":   {{Code: "98", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"1000": {{Code: "1000", Text: "Стався технічний збій", Contact: "monobank"}},
	"1005": {{Code: "1005", Text: "Стався технічний збій", Contact: "monobank"}},
	"1010": {{Code: "1010", Text: "Стався технічний збій", Contact: "monobank"}},
	"1014": {{Code: "1014", Text: "Для проведення оплати потрібно вказати повні реквізити карти", Contact: "покупець"}},
	"1034": {{Code: "1034", Text: "3-D Secure перевірку не пройдено", Contact: "банк, який випустив картку"}},
	"1035": {{Code: "1035", Text: "3-D Secure перевірку не пройдено", Contact: "банк, який випустив картку"}},
	"1036": {{Code: "1036", Text: "Стався технічний збій", Contact: "monobank"}},
	"1044": {{Code: "1044", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"1045": {{Code: "1045", Text: "3-D Secure перевірку не пройдено", Contact: "банк, який випустив картку"}},
	"1053": {{Code: "1053", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"1054": {{Code: "1054", Text: "3-D Secure перевірку не пройдено", Contact: "monobank"}},
	"1056": {{Code: "1056", Text: "Переказ можливий тільки на картку українського банку", Contact: "monobank"}},
	"1064": {{Code: "1064", Text: "Оплата можлива лише з використанням карток Mastercard або Visa", Contact: "банк, який випустив картку"}},
	"1066": {{Code: "1066", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"1077": {{Code: "1077", Text: "Сума оплати менша ніж допустима сума (налаштування МПС)", Contact: "API"}},
	"1080": {{Code: "1080", Text: "Термін дії карти вказаний невірно", Contact: "банк, який випустив картку"}},
	"1090": {{Code: "1090", Text: "Інформація про клієнта не знайдена", Contact: "monobank"}},
	"1115": {{Code: "1115", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"1121": {{Code: "1121", Text: "Помилка налаштувань торгівельної точки", Contact: "monobank"}},
	"1145": {{Code: "1145", Text: "Мінімальна сума переказу", Contact: "monobank"}},
	"1165": {{Code: "1165", Text: "Операцію заблоковано банком-емітентом", Contact: "банк, який випустив картку"}},
	"1187": {{Code: "1187", Text: "Треба вказати імʼя отримувача", Contact: "API"}},
	"1193": {{Code: "1193", Text: "Операцію заблоковано банком-емітентом", Contact: "банк, який випустив картку"}},
	"1194": {{Code: "1194", Text: "Цей спосіб поповнення працює тільки з картами інших банків", Contact: "monobank"}},
	"1200": {{Code: "1200", Text: "Обов'язкова наявність CVV коду", Contact: "банк, який випустив картку"}},
	"1405": {{Code: "1405", Text: "Платіжна система обмежила перекази", Contact: "банк, який випустив картку"}},
	"1406": {{Code: "1406", Text: "Карта заблокована ризик-менеджментом", Contact: "банк, який випустив картку"}},
	"1407": {{Code: "1407", Text: "Операцію заблоковано ризик-менеджментом", Contact: "monobank"}},
	"1408": {{Code: "1408", Text: "Операцію заблоковано банком-емітентом", Contact: "банк, який випустив картку"}},
	"1411": {{Code: "1411", Text: "Цей вид операцій з гривневих карток тимчасово обмежений", Contact: "monobank"}},
	"1413": {{Code: "1413", Text: "Операцію заблоковано банком-емітентом", Contact: "банк, який випустив картку"}},
	"1419": {{Code: "1419", Text: "Термін дії карти вказаний невірно", Contact: "банк, який випустив картку"}},
	"1420": {{Code: "1420", Text: "Стався технічний збій", Contact: "monobank"}},
	"1421": {{Code: "1421", Text: "3-D Secure перевірку не пройдено", Contact: "банк, який випустив картку"}},
	"1422": {{Code: "1422", Text: "Виникла помилка на етапі 3-D Secure", Contact: "банк, який випустив картку"}},
	"1425": {{Code: "1425", Text: "Виникла помилка на етапі 3-D Secure", Contact: "банк, який випустив картку"}},
	"1428": {{Code: "1428", Text: "Операцію заблоковано банком-емітентом", Contact: "банк, який випустив картку"}},
	"1429": {{Code: "1429", Text: "3-D Secure перевірку не пройдено", Contact: "банк, який випустив картку"}},
	"1433": {{Code: "1433", Text: "Перевірте імʼя та прізвище отримувача", Contact: "monobank"}},
	"1436": {{Code: "1436", Text: "Платіж відхилено (обмеження за політикою)", Contact: "monobank"}},
	"1439": {{Code: "1439", Text: "Недопустима операція для використання за програмою єВідновлення", Contact: "monobank"}},
	"1458": {{Code: "1458", Text: "Операцію відхилено на кроці 3DS", Contact: "банк, який випустив картку"}},
	"8001": {{Code: "8001", Text: "Минув термін дії посилання на оплату", Contact: "покупець"}},
	"8002": {{Code: "8002", Text: "Клієнт відмінив оплату", Contact: "покупець"}},
	"8003": {{Code: "8003", Text: "Стався технічний збій", Contact: "monobank"}},
	"8004": {{Code: "8004", Text: "Проблеми з проведенням 3-D Secure", Contact: "банк, який випустив картку"}},
	"8005": {{Code: "8005", Text: "Перевищено ліміти на прийом оплат", Contact: "monobank"}},
	"8006": {{Code: "8006", Text: "Перевищено ліміти на прийом оплат", Contact: "monobank"}},
}

// LookupPaymentErrorMetas returns meta info for the given errCode.
func LookupPaymentErrorMetas(code string) ([]PaymentErrorMeta, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, false
	}
	metas, ok := PaymentErrorCatalog[code]
	if !ok || len(metas) == 0 {
		return nil, false
	}
	// Return a copy to avoid accidental mutation from outside.
	out := make([]PaymentErrorMeta, len(metas))
	copy(out, metas)
	return out, true
}

// NewPaymentError builds a PaymentError from invoice status/webhook fields.
func NewPaymentError(invoiceID string, status InvoiceStatus, errCode string, failureReason string) *PaymentError {
	code := strings.TrimSpace(errCode)
	reason := strings.TrimSpace(failureReason)

	if code == "" && reason == "" {
		return nil
	}

	pe := &PaymentError{
		InvoiceID:     strings.TrimSpace(invoiceID),
		Status:        status,
		ErrCode:       code,
		FailureReason: reason,
	}

	if metas, ok := LookupPaymentErrorMetas(code); ok {
		pe.Metas = metas
	}
	return pe
}
