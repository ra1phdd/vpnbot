package services

import (
	"database/sql"
	"errors"
	"go.uber.org/zap"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type Promocodes struct {
	pr *repository.Promocodes
}

func NewPromocodes(pr *repository.Promocodes) *Promocodes {
	return &Promocodes{
		pr: pr,
	}
}

func (p *Promocodes) IsWork(code string, onlyNewUsers bool) bool {
	data, err := p.pr.GetByCode(code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		logger.Error("Failed to get promocode by code", zap.String("code", code), zap.Error(err))
		return false
	}

	if data.OnlyNewUsers != onlyNewUsers || !data.IsActive || data.CurrentActivations > *data.TotalActivations {
		return false
	}

	return true
}
