package expenses

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/internal/apierr"
)

var expenseCategoriesByBusiness = map[string][]string{
	"retail":           {"Rent", "Utilities", "Salaries", "Stock purchases", "Transport", "Packaging", "Repairs & maintenance", "Marketing", "Licences & permits", "Insurance", "Other"},
	"wholesale":        {"Rent", "Utilities", "Salaries", "Stock purchases", "Transport & delivery", "Packaging", "Warehouse", "Repairs & maintenance", "Licences & permits", "Insurance", "Other"},
	"retail_wholesale": {"Rent", "Utilities", "Salaries", "Stock purchases", "Transport", "Transport & delivery", "Packaging", "Warehouse", "Repairs & maintenance", "Marketing", "Licences & permits", "Insurance", "Other"},
	"pharmacy":         {"Rent", "Utilities", "Salaries", "Medicine purchases", "Medical supplies", "Transport", "Repairs & maintenance", "Licences & permits", "Insurance", "Other"},
}

func (s *Service) ensureDefaultCategories(ctx context.Context, orgID uuid.UUID) error {
	var organisation struct {
		BusinessType        string `gorm:"column:business_type"`
		ExpenseDefaultsType string `gorm:"column:expense_defaults_type"`
	}
	if err := s.repo.db.WithContext(ctx).Table("organisations").Select("business_type", "expense_defaults_type").Where("id = ?", orgID).Scan(&organisation).Error; err != nil {
		return err
	}
	defaults := expenseCategoriesByBusiness[organisation.BusinessType]
	if len(defaults) == 0 || organisation.ExpenseDefaultsType == organisation.BusinessType {
		return nil
	}
	return s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing []string
		if err := tx.Model(&ExpenseCategory{}).Where("organisation_id = ?", orgID).Pluck("name", &existing).Error; err != nil {
			return err
		}
		known := make(map[string]bool, len(existing))
		for _, name := range existing {
			known[strings.ToLower(name)] = true
		}
		var historical []string
		if err := tx.Model(&Expense{}).Where("organisation_id = ?", orgID).Distinct("category").Pluck("category", &historical).Error; err != nil {
			return err
		}
		legacyCategories := make([]ExpenseCategory, 0, len(historical))
		for _, name := range historical {
			name = strings.TrimSpace(name)
			key := strings.ToLower(name)
			if name == "" || known[key] {
				continue
			}
			known[key] = true
			legacyCategories = append(legacyCategories, ExpenseCategory{OrganisationID: orgID, Name: name})
		}
		if len(legacyCategories) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&legacyCategories).Error; err != nil {
				return err
			}
		}
		categories := make([]ExpenseCategory, 0, len(defaults))
		for _, name := range defaults {
			key := strings.ToLower(name)
			if known[key] {
				continue
			}
			known[key] = true
			categories = append(categories, ExpenseCategory{OrganisationID: orgID, Name: name, IsDefault: true})
		}
		if len(categories) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&categories).Error; err != nil {
				return err
			}
		}
		return tx.Table("organisations").Where("id = ?", orgID).Update("expense_defaults_type", organisation.BusinessType).Error
	})
}

func (s *Service) ListCategories(ctx context.Context, orgID uuid.UUID) ([]ExpenseCategory, error) {
	if err := s.ensureDefaultCategories(ctx, orgID); err != nil {
		return nil, err
	}
	var categories []ExpenseCategory
	if err := s.repo.db.WithContext(ctx).Where("organisation_id = ?", orgID).Order("name asc").Find(&categories).Error; err != nil {
		return nil, err
	}
	for i := range categories {
		if err := s.repo.db.WithContext(ctx).Model(&Expense{}).
			Where("organisation_id = ? AND lower(category) = lower(?)", orgID, categories[i].Name).
			Count(&categories[i].ExpenseCount).Error; err != nil {
			return nil, err
		}
	}
	return categories, nil
}

func (s *Service) CreateCategory(ctx context.Context, orgID uuid.UUID, name string) (*ExpenseCategory, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierr.BadRequest("Category name is required")
	}
	var count int64
	if err := s.repo.db.WithContext(ctx).Model(&ExpenseCategory{}).
		Where("organisation_id = ? AND lower(name) = lower(?)", orgID, name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, apierr.Conflict("A category with this name already exists")
	}
	category := &ExpenseCategory{OrganisationID: orgID, Name: name}
	if err := s.repo.db.WithContext(ctx).Create(category).Error; err != nil {
		return nil, err
	}
	return category, nil
}

func (s *Service) UpdateCategory(ctx context.Context, orgID, id uuid.UUID, name string) (*ExpenseCategory, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierr.BadRequest("Category name is required")
	}
	var category ExpenseCategory
	if err := s.repo.db.WithContext(ctx).Where("id = ? AND organisation_id = ?", id, orgID).First(&category).Error; err != nil {
		return nil, apierr.NotFound("Category not found")
	}
	if strings.EqualFold(category.Name, name) {
		return &category, nil
	}
	var count int64
	if err := s.repo.db.WithContext(ctx).Model(&ExpenseCategory{}).
		Where("organisation_id = ? AND id <> ? AND lower(name) = lower(?)", orgID, id, name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, apierr.Conflict("A category with this name already exists")
	}
	oldName := category.Name
	if err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		category.Name = name
		category.IsDefault = false
		if err := tx.Save(&category).Error; err != nil {
			return err
		}
		return tx.Model(&Expense{}).
			Where("organisation_id = ? AND lower(category) = lower(?)", orgID, oldName).
			Update("category", name).Error
	}); err != nil {
		return nil, err
	}
	return &category, nil
}

func (s *Service) DeleteCategory(ctx context.Context, orgID, id uuid.UUID) error {
	var category ExpenseCategory
	if err := s.repo.db.WithContext(ctx).Where("id = ? AND organisation_id = ?", id, orgID).First(&category).Error; err != nil {
		return apierr.NotFound("Category not found")
	}
	var count int64
	if err := s.repo.db.WithContext(ctx).Model(&Expense{}).
		Where("organisation_id = ? AND lower(category) = lower(?)", orgID, category.Name).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return apierr.Conflict("This category is assigned to expenses; reassign them before deleting it")
	}
	return s.repo.db.WithContext(ctx).Delete(&category).Error
}

func (s *Service) validateCategory(ctx context.Context, orgID uuid.UUID, name string) error {
	var count int64
	if err := s.repo.db.WithContext(ctx).Model(&ExpenseCategory{}).
		Where("organisation_id = ? AND lower(name) = lower(?)", orgID, strings.TrimSpace(name)).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return apierr.BadRequest("Choose an expense category")
	}
	return nil
}
