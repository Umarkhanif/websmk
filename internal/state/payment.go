package state

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"
)

type PaymentResult struct {
	IntentID      string `json:"intentId"`
	InvoiceID     string `json:"invoiceId"`
	Status        string `json:"status"`
	Amount        int64  `json:"amount"`
	VANumber      string `json:"vaNumber"`
	ReceiptNumber string `json:"receiptNumber,omitempty"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
	PaidAt        string `json:"paidAt,omitempty"`
	AlreadyPaid   bool   `json:"alreadyPaid,omitempty"`
}

// CreatePaymentIntent records a pending VA payment without changing the
// invoice. The gateway settles it later through the signed webhook.
func CreatePaymentIntent(doc map[string]any, studentID, invoiceID string, now time.Time) (PaymentResult, error) {
	invoice, _, err := payableInvoice(doc, studentID, invoiceID)
	if err != nil {
		return PaymentResult{}, err
	}
	if intent := pendingIntent(doc, invoiceID, studentID, now); intent != nil {
		return intentResult(intent), nil
	}

	intent := newIntent(invoiceID, studentID, dueAmount(invoice), now)
	doc["paymentIntents"] = append(asList(doc["paymentIntents"]), intent)
	return intentResult(intent), nil
}

// SettlePaymentIntent is the production path after a gateway callback.
func SettlePaymentIntent(doc map[string]any, intentID, providerRef, receiptSecret string, now time.Time) (PaymentResult, error) {
	intent := findIntent(doc, intentID)
	if intent == nil {
		return PaymentResult{}, fmt.Errorf("%w: payment intent", ErrNotFound)
	}
	if text(intent["status"]) == "PAID" {
		result := intentResult(intent)
		result.AlreadyPaid = true
		return result, nil
	}
	if text(intent["status"]) != "PENDING" {
		return PaymentResult{}, fmt.Errorf("%w: status payment intent", ErrConflict)
	}
	if intentExpired(intent, now) {
		intent["status"] = "EXPIRED"
		return PaymentResult{}, fmt.Errorf("%w: virtual account kedaluwarsa", ErrConflict)
	}
	return settleIntent(doc, intent, providerRef, receiptSecret, now)
}

func settleIntent(doc map[string]any, intent map[string]any, providerRef, receiptSecret string, now time.Time) (PaymentResult, error) {
	invoiceID := text(intent["invoiceId"])
	studentID := text(intent["studentId"])
	invoice, student, err := payableInvoice(doc, studentID, invoiceID)
	if err != nil {
		return PaymentResult{}, err
	}

	amount := dueAmount(invoice)
	if amount <= 0 {
		return PaymentResult{}, fmt.Errorf("%w: tagihan sudah lunas", ErrConflict)
	}

	receiptNumber := "KWT-VA-" + now.UTC().Format("20060102") + "-" + randomHex(2)
	createdAt := now.UTC().Format(time.RFC3339)
	signature := ReceiptSignature(receiptSecret, invoiceID, receiptNumber, amount, createdAt)

	invoice["paidAmount"] = float64(invoiceAmount(invoice))
	invoice["status"] = "PAID"

	entry := map[string]any{
		"id":                   nextLedgerID(doc),
		"invoiceId":            invoiceID,
		"receiptNumber":        receiptNumber,
		"txType":               "PAYMENT_VA",
		"amountDelta":          float64(amount),
		"studentName":          text(student["fullName"]),
		"actorUserId":          "payment-gateway",
		"actorName":            "Payment Gateway (VA)",
		"treasurerPinVerified": false,
		"signatureHmac":        signature,
		"createdAt":            createdAt,
	}
	doc["ledger"] = append([]any{entry}, asList(doc["ledger"])...)

	intent["status"] = "PAID"
	intent["amount"] = float64(amount)
	intent["paidAt"] = createdAt
	intent["receiptNumber"] = receiptNumber
	if providerRef != "" {
		intent["providerRef"] = providerRef
	}

	return intentResult(intent), nil
}

func payableInvoice(doc map[string]any, studentID, invoiceID string) (map[string]any, map[string]any, error) {
	student := findStudent(doc, studentID)
	if student == nil {
		return nil, nil, fmt.Errorf("%w: profil siswa", ErrNotFound)
	}
	invoice := findInvoice(doc, invoiceID, studentID)
	if invoice == nil {
		return nil, nil, fmt.Errorf("%w: tagihan", ErrNotFound)
	}
	if text(invoice["status"]) == "PAID" || dueAmount(invoice) <= 0 {
		return nil, nil, fmt.Errorf("%w: tagihan sudah lunas", ErrConflict)
	}
	return invoice, student, nil
}

func findInvoice(doc map[string]any, invoiceID, studentID string) map[string]any {
	for _, item := range asList(doc["invoices"]) {
		invoice := asMap(item)
		if invoice != nil && recordID(invoice["id"]) == invoiceID && text(invoice["studentId"]) == studentID {
			return invoice
		}
	}
	return nil
}

func newIntent(invoiceID, studentID string, amount int64, now time.Time) map[string]any {
	return map[string]any{
		"id":        fmt.Sprintf("pi-%d", now.UnixNano()),
		"invoiceId": invoiceID,
		"studentId": studentID,
		"amount":    float64(amount),
		"status":    "PENDING",
		"vaNumber":  vaNumber(now),
		"provider":  "va",
		"createdAt": now.UTC().Format(time.RFC3339),
		"expiresAt": now.Add(24 * time.Hour).UTC().Format(time.RFC3339),
	}
}

func pendingIntent(doc map[string]any, invoiceID, studentID string, now time.Time) map[string]any {
	for _, item := range asList(doc["paymentIntents"]) {
		intent := asMap(item)
		if intent != nil &&
			text(intent["invoiceId"]) == invoiceID &&
			text(intent["studentId"]) == studentID &&
			text(intent["status"]) == "PENDING" &&
			!intentExpired(intent, now) {
			return intent
		}
	}
	return nil
}

func intentExpired(intent map[string]any, now time.Time) bool {
	expiresAt, err := time.Parse(time.RFC3339, text(intent["expiresAt"]))
	if err != nil {
		return false
	}
	return !now.Before(expiresAt)
}

func findIntent(doc map[string]any, intentID string) map[string]any {
	for _, item := range asList(doc["paymentIntents"]) {
		intent := asMap(item)
		if intent != nil && text(intent["id"]) == intentID {
			return intent
		}
	}
	return nil
}

func intentResult(intent map[string]any) PaymentResult {
	amount, _ := number(intent["amount"])
	return PaymentResult{
		IntentID:      text(intent["id"]),
		InvoiceID:     text(intent["invoiceId"]),
		Status:        text(intent["status"]),
		Amount:        int64(amount),
		VANumber:      text(intent["vaNumber"]),
		ReceiptNumber: text(intent["receiptNumber"]),
		ExpiresAt:     text(intent["expiresAt"]),
		PaidAt:        text(intent["paidAt"]),
	}
}

func dueAmount(invoice map[string]any) int64 {
	finalAmount := invoiceAmount(invoice)
	paid, _ := number(invoice["paidAmount"])
	amount := int64(math.Round(finalAmount)) - int64(math.Round(paid))
	if amount < 0 {
		return 0
	}
	return amount
}

func invoiceAmount(invoice map[string]any) float64 {
	finalAmount, _ := number(invoice["finalAmount"])
	return finalAmount
}

func nextLedgerID(doc map[string]any) int64 {
	var maxID int64
	for _, item := range asList(doc["ledger"]) {
		record := asMap(item)
		if record == nil {
			continue
		}
		if id, ok := number(record["id"]); ok && int64(id) > maxID {
			maxID = int64(id)
		}
	}
	if maxID < 1000 {
		maxID = 1000
	}
	return maxID + 1
}

func vaNumber(now time.Time) string {
	return fmt.Sprintf("88019%06d", now.UnixNano()%1_000_000)
}

func randomHex(bytesCount int) string {
	buf := make([]byte, bytesCount)
	if _, err := rand.Read(buf); err != nil {
		return strings.ToUpper(fmt.Sprintf("%x", time.Now().UnixNano()%0xffff))
	}
	return strings.ToUpper(hex.EncodeToString(buf))
}

// ReceiptSignature is reproducible so a public verification endpoint can
// validate a receipt without trusting the browser.
func ReceiptSignature(secret, invoiceID, receiptNumber string, amount int64, createdAt string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s|%s|%d|%s", invoiceID, receiptNumber, amount, createdAt)
	return hex.EncodeToString(mac.Sum(nil))
}
