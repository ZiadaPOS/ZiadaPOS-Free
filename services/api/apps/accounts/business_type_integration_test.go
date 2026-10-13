package accounts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/config"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/integrations/sms"
	"github.com/troubleman96/ziadapos/internal/security"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func accountIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("ZIADA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ZIADA_TEST_DATABASE_URL to run the PostgreSQL account integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	schema := "accounts_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DROP SCHEMA " + schema + " CASCADE").Error })
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestOutletTypeIsOnlyStoredForPharmacyAccounts(t *testing.T) {
	db := accountIntegrationDB(t)
	if err := db.Exec("ALTER TABLE organisations ALTER COLUMN outlet_type SET DEFAULT 'pharmacy'").Error; err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("upgrade migration failed: %v", err)
	}
	var outletDefault string
	if err := db.Raw("SELECT column_default FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'organisations' AND column_name = 'outlet_type'").Scan(&outletDefault).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(outletDefault, "pharmacy") {
		t.Fatalf("legacy outlet_type default remains after migration: %q", outletDefault)
	}

	for _, businessType := range []string{"retail", "wholesale", "retail_wholesale", "pharmacy"} {
		org := Organisation{Name: businessType, Slug: businessType, BusinessType: businessType, IsActive: true}
		if businessType == "pharmacy" {
			org.OutletType = "addo"
		}
		if err := db.Create(&org).Error; err != nil {
			t.Fatalf("create %s organisation: %v", businessType, err)
		}
		var stored Organisation
		if err := db.First(&stored, "id = ?", org.ID).Error; err != nil {
			t.Fatal(err)
		}
		if businessType == "pharmacy" && stored.OutletType != "addo" {
			t.Fatalf("pharmacy outlet type = %q; want addo", stored.OutletType)
		}
		if businessType != "pharmacy" && stored.OutletType != "" {
			t.Fatalf("%s organisation inherited outlet type %q", businessType, stored.OutletType)
		}
	}
}

func TestPharmacyRegistrationStoresTypeAndCompletesPhoneVerification(t *testing.T) {
	db := accountIntegrationDB(t)
	codeSent := make(chan string, 1)
	smsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode SMS request: %v", err)
		}
		match := regexp.MustCompile(`[0-9]{6}`).FindString(payload.Message)
		if match == "" {
			t.Errorf("verification code missing from SMS message %q", payload.Message)
		}
		codeSent <- match
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"message_id":"integration"}}`))
	}))
	defer smsServer.Close()

	cfg := &config.Config{
		SendAfricaBaseURL: smsServer.URL, SendAfricaAPIKey: "integration-key",
		JWTAccessSecret:  "integration-access-secret-at-least-32-bytes",
		JWTRefreshSecret: "integration-refresh-secret-at-least-32-bytes",
		JWTAccessTTL:     15 * time.Minute, JWTRefreshTTL: 24 * time.Hour,
	}
	bus := events.New(1, 8)
	defer bus.Close()
	service := NewAuthService(NewRepo(db), nil, nil, sms.New(cfg), security.NewManager(cfg), bus, cfg)
	result, err := service.Register(context.Background(), &RegisterRequest{
		FirstName: "Asha", Phone: "0712345678", Password: "securepass", ConfirmPassword: "securepass",
		Shop:       ShopInput{Name: "Asha Pharmacy", BusinessType: "pharmacy", OutletType: "addo", Region: "dar-es-salaam"},
	})
	if err != nil {
		t.Fatalf("register pharmacy account: %v", err)
	}
	code := <-codeSent
	pair, err := service.VerifyOTP(context.Background(), "0712345678", otpPurposeRegistration, code)
	if err != nil {
		t.Fatalf("verify registration phone: %v", err)
	}
	if pair == nil || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("phone verification did not issue a session")
	}
	org, err := NewRepo(db).GetOrgByID(result.OrganisationID)
	if err != nil {
		t.Fatal(err)
	}
	if org.BusinessType != "pharmacy" || org.OutletType != "addo" {
		t.Fatalf("registered business = %q / %q; want pharmacy / addo", org.BusinessType, org.OutletType)
	}
	user, err := NewRepo(db).GetUserByPhone("0712345678")
	if err != nil || !user.IsPhoneVerified {
		t.Fatalf("registered user phone verification failed: user=%+v err=%v", user, err)
	}
}
