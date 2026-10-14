package accounts

import (
	"errors"
	"fmt"

	"github.com/troubleman96/ziadapos/internal/security"
	"gorm.io/gorm"
)

// MigrateIntegrationSecrets encrypts legacy provider keys in place before routes
// are registered. It never logs or returns credential contents. Existing values
// that cannot be encrypted or opened stop startup rather than silently remaining
// usable as plaintext or being discarded.
func MigrateIntegrationSecrets(db *gorm.DB, box *security.SecretBox) error {
	var organisations []Organisation
	if err := db.Select("id", "send_africa_api_key", "ngamia_api_key").Find(&organisations).Error; err != nil {
		return fmt.Errorf("load integration credential records: %w", err)
	}

	for i := range organisations {
		org := &organisations[i]
		updates := map[string]interface{}{}
		for _, field := range []struct {
			column string
			value  string
		}{{"send_africa_api_key", org.SendAfricaAPIKey}, {"ngamia_api_key", org.NgamiaAPIKey}} {
			if field.value == "" {
				continue
			}
			if box == nil {
				return errors.New("INTEGRATION_ENCRYPTION_KEY is required to migrate or use saved provider credentials")
			}
			if security.IsEncrypted(field.value) {
				if _, err := box.Decrypt(field.value); err != nil {
					return errors.New("saved provider credential cannot be opened; verify INTEGRATION_ENCRYPTION_KEY")
				}
				continue
			}
			ciphertext, err := box.Encrypt(field.value)
			if err != nil {
				return errors.New("saved provider credential could not be encrypted")
			}
			updates[field.column] = ciphertext
		}
		if len(updates) > 0 {
			if err := db.Model(&Organisation{}).Where("id = ?", org.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("encrypt legacy provider credentials for organisation %s: %w", org.ID, err)
			}
		}
	}
	return nil
}
