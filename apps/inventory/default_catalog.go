package inventory

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type catalogCategory struct {
	name        string
	description string
}

var retailCategories = []catalogCategory{
	{"Food & Groceries", "Food and everyday grocery items"},
	{"Beverages", "Drinks and beverages"},
	{"Household", "Household goods and cleaning supplies"},
	{"Personal Care", "Personal care and hygiene products"},
	{"Clothing & Accessories", "Clothing, footwear and accessories"},
	{"Electronics", "Electronic goods and accessories"},
	{"Stationery", "Office and school supplies"},
	{"Other", "Other products"},
}

var wholesaleCategories = []catalogCategory{
	{"Food & Groceries", "Food and grocery goods sold in bulk"},
	{"Beverages", "Drinks and beverages sold in bulk"},
	{"Household & Cleaning", "Household goods and cleaning supplies"},
	{"Personal Care", "Personal care and hygiene products"},
	{"Building Materials", "Construction and hardware supplies"},
	{"Agricultural Supplies", "Agricultural goods and supplies"},
	{"Other", "Other products"},
}

var pharmacyCategories = []catalogCategory{
	{"Prescription Medicines", "Medicines requiring a prescription"},
	{"Over-the-counter Medicines", "Non-prescription medicines"},
	{"Vitamins & Supplements", "Vitamins, supplements and nutrition"},
	{"Medical Devices", "Medical equipment and devices"},
	{"First Aid", "First aid and wound care"},
	{"Personal Care", "Personal care and hygiene products"},
	{"Mother & Baby", "Mother and baby health products"},
	{"Other", "Other pharmacy products"},
}

var retailUnits = []string{
	"piece", "unit", "pack", "box", "bottle", "carton", "case", "bag", "sack", "kg", "g", "mg",
	"tonne", "litre", "ml", "cl", "dozen", "pair", "set", "bundle", "bunch", "crate", "tray",
	"can", "tin", "jar", "tub", "bucket", "roll", "sheet", "ream", "metre", "cm", "bar",
}
var wholesaleUnits = []string{
	"piece", "unit", "pack", "box", "carton", "case", "pallet", "bag", "sack", "bale", "bundle",
	"kg", "g", "tonne", "litre", "ml", "drum", "crate", "container", "dozen", "pair", "set",
	"roll", "sheet", "ream", "tray", "can", "tin", "bucket", "metre",
}
var pharmacyUnits = []string{
	"tablet", "capsule", "caplet", "softgel", "chewable tablet", "dispersible tablet", "extended-release tablet",
	"strip", "blister pack", "sachet", "lozenge", "suppository", "pessary", "dose", "drop", "spray",
	"bottle", "vial", "ampoule", "prefilled syringe", "cartridge", "infusion bag", "tube", "inhaler",
	"nebuliser solution", "cream", "ointment", "gel", "lotion", "patch", "eye drops", "ear drops",
	"nasal spray", "oral solution", "syrup", "suspension", "powder", "granules", "gum", "piece",
	"box", "pack", "ml", "litre",
}

const inventoryDefaultsVersion = 1

func catalogForBusinessType(businessType string) ([]catalogCategory, []string) {
	switch businessType {
	case "retail":
		return retailCategories, retailUnits
	case "wholesale":
		return wholesaleCategories, wholesaleUnits
	case "retail_wholesale":
		categories := append(append([]catalogCategory{}, retailCategories...), wholesaleCategories...)
		return categories, append(append([]string{}, retailUnits...), wholesaleUnits...)
	case "pharmacy":
		return pharmacyCategories, pharmacyUnits
	default:
		return nil, nil
	}
}

// ensureBusinessDefaults adds a business-specific starter catalogue once per
// organisation and business type, preserving any existing custom entries.
func (s *Service) ensureBusinessDefaults(ctx context.Context, orgID uuid.UUID) error {
	var organisation struct {
		BusinessType             string `gorm:"column:business_type"`
		InventoryDefaultsType    string `gorm:"column:inventory_defaults_type"`
		InventoryDefaultsVersion int    `gorm:"column:inventory_defaults_version"`
	}
	if err := s.repo.db.WithContext(ctx).Table("organisations").Select("business_type", "inventory_defaults_type", "inventory_defaults_version").Where("id = ?", orgID).Scan(&organisation).Error; err != nil {
		return err
	}
	categories, units := catalogForBusinessType(organisation.BusinessType)
	if len(categories) == 0 || len(units) == 0 {
		return nil
	}
	if organisation.InventoryDefaultsType == organisation.BusinessType && organisation.InventoryDefaultsVersion >= inventoryDefaultsVersion {
		return nil
	}
	seeded := false
	err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var categoryNames, unitNames []string
		if err := tx.Model(&Category{}).Where("organisation_id = ?", orgID).Pluck("name", &categoryNames).Error; err != nil {
			return err
		}
		if err := tx.Model(&Unit{}).Where("organisation_id = ?", orgID).Pluck("name", &unitNames).Error; err != nil {
			return err
		}
		knownCategories := make(map[string]bool, len(categoryNames))
		for _, name := range categoryNames {
			knownCategories[strings.ToLower(name)] = true
		}
		defaults := make([]Category, 0, len(categories))
		for index, category := range categories {
			key := strings.ToLower(category.name)
			if knownCategories[key] {
				continue
			}
			knownCategories[key] = true
			defaults = append(defaults, Category{
				OrganisationID: orgID,
				Name:           category.name,
				Description:    category.description,
				SortOrder:      index + 1,
				IsActive:       true,
				IsDefault:      true,
			})
		}
		if len(defaults) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&defaults).Error; err != nil {
				return err
			}
			seeded = true
		}
		knownUnits := make(map[string]bool, len(unitNames))
		for _, name := range unitNames {
			knownUnits[strings.ToLower(name)] = true
		}
		unitDefaults := make([]Unit, 0, len(units))
		for _, name := range units {
			key := strings.ToLower(name)
			if knownUnits[key] {
				continue
			}
			knownUnits[key] = true
			unitDefaults = append(unitDefaults, Unit{OrganisationID: orgID, Name: name, IsActive: true, IsDefault: true})
		}
		if len(unitDefaults) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&unitDefaults).Error; err != nil {
				return err
			}
			seeded = true
		}
		if err := tx.Table("organisations").Where("id = ?", orgID).Updates(map[string]interface{}{
			"inventory_defaults_type":    organisation.BusinessType,
			"inventory_defaults_version": inventoryDefaultsVersion,
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err == nil && seeded {
		s.invalidate(nil, &orgID)
	}
	return err
}
