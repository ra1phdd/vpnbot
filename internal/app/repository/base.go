package repository

import (
	"gorm.io/gorm"
	"log/slog"
	"nsvpn/pkg/logger"
	"reflect"
)

func updateField(log *logger.Logger, tx *gorm.DB, dest any, fieldName string, currentVal, newVal interface{}) error {
	if reflect.DeepEqual(newVal, currentVal) {
		return nil
	}

	updateMap := map[string]interface{}{fieldName: newVal}
	if err := tx.Model(dest).Updates(updateMap).Error; err != nil {
		log.Error("Failed to update field", err, slog.String("field", fieldName), slog.Any("dest", dest), slog.Any("oldValue", currentVal), slog.Any("newValue", newVal))
		return err
	}

	log.Debug("Updated currency field", slog.String("field", fieldName), slog.Any("oldValue", currentVal), slog.Any("newValue", newVal))
	return nil
}
