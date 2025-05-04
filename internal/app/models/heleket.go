package models

type HeleketRequest struct {
	Amount                 string   `json:"amount"`                             // Сумма, подлежащая выплате (с разделителем '.')
	Currency               string   `json:"currency"`                           // Код валюты
	OrderID                string   `json:"order_id"`                           // Уникальный идентификатор заказа (1-128 символов, alpha_dash)
	Network                string   `json:"network,omitempty"`                  // Сетевой код блокчейна
	ReturnURL              string   `json:"url_return,omitempty"`               // URL для возврата в магазин (6-255 символов)
	SuccessURL             string   `json:"url_success,omitempty"`              // URL после успешной оплаты (6-255 символов)
	CallbackURL            string   `json:"url_callback,omitempty"`             // URL для webhooks (6-255 символов)
	IsPaymentMultiple      bool     `json:"is_payment_multiple,omitempty"`      // Разрешена ли оплата частями
	Lifetime               int      `json:"lifetime,omitempty"`                 // Срок действия счета в секундах (300-43200)
	ToCurrency             string   `json:"to_currency,omitempty"`              // Целевая валюта (код криптовалюты)
	Subtract               int      `json:"subtract,omitempty"`                 // Процент комиссии с клиента (0-100)
	AccuracyPaymentPercent float64  `json:"accuracy_payment_percent,omitempty"` // Допустимая неточность оплаты (0-5)
	AdditionalData         string   `json:"additional_data,omitempty"`          // Доп. информация (макс. 255 символов)
	Currencies             []string `json:"currencies,omitempty"`               // Разрешенные валюты для оплаты
	ExceptCurrencies       []string `json:"except_currencies,omitempty"`        // Исключенные валюты
	CourseSource           string   `json:"course_source,omitempty"`            // Источник обменного курса
	FromReferralCode       string   `json:"from_referral_code,omitempty"`       // Реферальный код
	DiscountPercent        *int     `json:"discount_percent,omitempty"`         // Скидка/доп. комиссия (-99 до 100)
	IsRefresh              bool     `json:"is_refresh,omitempty"`               // Обновить адрес и срок действия
}

type HeleketResponse struct {
	State  int           `json:"state"`
	Result HeleketResult `json:"result"`
}

type HeleketResult struct {
	UUID            string  `json:"uuid"`
	OrderID         string  `json:"order_id"`
	Amount          string  `json:"amount"`
	PaymentAmount   *string `json:"payment_amount"`
	PayerAmount     *string `json:"payer_amount"`
	DiscountPercent *int    `json:"discount_percent"`
	Discount        string  `json:"discount"`
	PayerCurrency   *string `json:"payer_currency"`
	Currency        string  `json:"currency"`
	MerchantAmount  *string `json:"merchant_amount"`
	Network         *string `json:"network"`
	Address         *string `json:"address"`
	From            *string `json:"from"`
	Txid            *string `json:"txid"`
	PaymentStatus   string  `json:"payment_status"`
	URL             string  `json:"url"`
	ExpiredAt       int64   `json:"expired_at"`
	Status          string  `json:"status"`
	IsFinal         bool    `json:"is_final"`
	AdditionalData  *string `json:"additional_data"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type HeleketInfoRequest struct {
	UUID    string `json:"uuid"`     // UUID счета-фактуры
	OrderID string `json:"order_id"` // Уникальный идентификатор заказа (1-128 символов, alpha_dash)
}
