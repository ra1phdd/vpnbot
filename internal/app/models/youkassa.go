package models

import "time"

type YoukassaRequest struct {
	Amount       YoukassaAmount       `json:"amount"`
	Capture      bool                 `json:"capture"`
	Confirmation YoukassaConfirmation `json:"confirmation"`
	Description  string               `json:"description"`
	Receipt      YoukassaReceipt      `json:"receipt"`
}

type YoukassaResponse struct {
	ID           string               `json:"id"`
	Status       string               `json:"status"`
	Paid         bool                 `json:"paid"`
	Amount       YoukassaAmount       `json:"amount"`
	Confirmation YoukassaConfirmation `json:"confirmation"`
	CreatedAt    time.Time            `json:"created_at"`
	Description  string               `json:"description"`
	Metadata     interface{}          `json:"metadata"`
	Recipient    YoukassaRecipient    `json:"recipient"`
	Refundable   bool                 `json:"refundable"`
	Test         bool                 `json:"test"`
}

type YoukassaAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type YoukassaConfirmation struct {
	Type             string `json:"type"`
	ConfirmationURL  string `json:"confirmation_url"`
	ReturnURL        string `json:"return_url,omitempty"`
	Enforce          bool   `json:"enforce,omitempty"`
	ConfirmationData string `json:"confirmation_data,omitempty"`
}

type YoukassaRecipient struct {
	AccountID string `json:"account_id"`
	GatewayID string `json:"gateway_id"`
}

type YoukassaReceipt struct {
	Customer YoukassaCustomer `json:"customer"`
	Items    []YoukassaItem   `json:"items"`
}

type YoukassaCustomer struct {
	FullName string `json:"full_name,omitempty"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Inn      string `json:"inn,omitempty"`
}

type YoukassaItem struct {
	Description    string         `json:"description"`
	Quantity       int            `json:"quantity"`
	Amount         YoukassaAmount `json:"amount"`
	VatCode        int            `json:"vat_code"`
	Measure        string         `json:"measure"`
	PaymentMode    string         `json:"payment_mode"`
	PaymentSubject string         `json:"payment_subject"`
}
