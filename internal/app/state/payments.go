package state

import "nsvpn/internal/app/models"

type PaymentsState struct {
	Amount            float64
	Payload           string
	Description       string
	Note              string
	Promocode         *models.Promocode
	IsBuySubscription bool
}

type PaginationState struct {
	CurrentPage int
	TotalPages  int
}
