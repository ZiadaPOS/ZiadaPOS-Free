package pharmacy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/apps/notifications"
)

type pharmacyStoreScope struct {
	StoreID uuid.UUID
	OrgID   uuid.UUID
}

// ScheduleExpiryAlerts creates one deduplicated in-app expiry digest per
// pharmacy store each morning in the configured application timezone.
func (s *Service) ScheduleExpiryAlerts(db *gorm.DB, scheduler interface {
	AddDaily(hour, minute int, name string, fn func()) error
}) error {
	return scheduler.AddDaily(8, 0, "pharmacy-expiry-alerts", func() {
		if err := s.RunExpiryAlerts(context.Background(), db); err != nil {
			fmt.Printf("pharmacy expiry alerts: %v\n", err)
		}
	})
}

func (s *Service) RunExpiryAlerts(ctx context.Context, db *gorm.DB) error {
	location, err := time.LoadLocation("Africa/Dar_es_Salaam")
	if err != nil {
		location = time.FixedZone("EAT", 3*60*60)
	}
	today := time.Now().In(location)
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, location)
	var stores []pharmacyStoreScope
	if err := db.WithContext(ctx).Table("stores").Select("stores.id AS store_id, stores.organisation_id AS org_id").Joins("JOIN organisations ON organisations.id = stores.organisation_id").Where("organisations.business_type = ?", "pharmacy").Find(&stores).Error; err != nil {
		return err
	}
	for _, store := range stores {
		var count int64
		if err := db.WithContext(ctx).Table("medicine_batches").Where("store_id = ? AND quantity > 0 AND expiry_date <= ?", store.StoreID, day.AddDate(0, 0, 30)).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			continue
		}
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			run := ExpiryAlertRun{StoreID: store.StoreID, AlertDay: day}
			result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "store_id"}, {Name: "alert_day"}}, DoNothing: true}).Create(&run)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}
			notice := notifications.Notification{OrganisationID: store.OrgID, StoreID: store.StoreID, Category: notifications.CatStock, Title: "Medicine batches need attention", Body: fmt.Sprintf("%d batch(es) are expired or expire within 30 days. Review quantities and expiry dates before dispensing.", count), Link: "/dashboard/pharmacy/expiry"}
			return tx.Create(&notice).Error
		})
		if err != nil {
			return err
		}
	}
	return nil
}
