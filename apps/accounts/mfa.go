package accounts

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"image/png"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/security"
	"golang.org/x/crypto/bcrypt"
)

const totpPeriod = 30 * time.Second

// BeginAdminTOTPSetup creates a temporary enrollment secret. The encrypted
// pending secret is persisted; plaintext is returned only for the scan/manual
// setup step and is never included in user models or later status responses.
func (s *AuthService) BeginAdminTOTPSetup(ctx context.Context, userID uuid.UUID, password string) (map[string]string, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, apierr.NotFound("Admin account not found")
	}
	if user.Role != "admin" {
		return nil, apierr.Forbidden("Platform admin access required")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, apierr.Unauthorized("Password is incorrect")
	}
	if user.MFAEnabled {
		return nil, apierr.Conflict("Authenticator MFA is already enabled")
	}
	if s.cache == nil {
		return nil, apierr.New(503, "unavailable", "Authenticator setup is temporarily unavailable")
	}
	_ = s.cache.Del(ctx, "mfa:setup:attempts:"+userID.String())
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, err
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	ciphertext, err := s.encryptMFASecret(secret)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(10 * time.Minute)
	user.MFAPendingSecretEncrypted = ciphertext
	user.MFAPendingExpiresAt = &deadline
	if err := s.repo.UpdateUser(user); err != nil {
		return nil, err
	}
	account := EmailString(user.Email)
	if account == "" {
		account = user.Phone
	}
	label := url.PathEscape("Ziada POS:" + account)
	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", "Ziada POS")
	query.Set("algorithm", "SHA1")
	query.Set("digits", "6")
	query.Set("period", "30")
	otpURL := "otpauth://totp/" + label + "?" + query.Encode()
	qr, err := qrcode.New(otpURL, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	qr.DisableBorder = false
	img := qr.Image(256)
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, img); err != nil {
		return nil, err
	}
	return map[string]string{
		"secret":           secret,
		"otpauth_url":      otpURL,
		"qr_code_data_url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes.Bytes()),
		"expires_at":       deadline.UTC().Format(time.RFC3339),
	}, nil
}

// VerifyAdminTOTPSetup enables MFA only after the authenticator proves it has
// imported the pending secret.
func (s *AuthService) VerifyAdminTOTPSetup(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil || user.Role != "admin" {
		return apierr.NotFound("Admin account not found")
	}
	if user.MFAEnabled || user.MFAPendingSecretEncrypted == "" || user.MFAPendingExpiresAt == nil {
		return apierr.BadRequest("Start authenticator setup before verifying")
	}
	if err := s.countMFAAttempt(ctx, "mfa:setup:attempts:"+userID.String(), 5, 10*time.Minute); err != nil {
		return err
	}
	if time.Now().After(*user.MFAPendingExpiresAt) {
		user.MFAPendingSecretEncrypted = ""
		user.MFAPendingExpiresAt = nil
		_ = s.repo.UpdateUser(user)
		return apierr.BadRequest("Setup expired. Generate a new QR code")
	}
	secret, err := s.decryptMFASecret(user.MFAPendingSecretEncrypted)
	if err != nil {
		return err
	}
	if !validTOTP(secret, code, time.Now()) {
		return apierr.Unauthorized("Invalid authenticator code")
	}
	user.MFASecretEncrypted = user.MFAPendingSecretEncrypted
	user.MFAPendingSecretEncrypted = ""
	user.MFAPendingExpiresAt = nil
	user.MFAEnabled = true
	if err := s.repo.UpdateUser(user); err != nil {
		return err
	}
	_ = s.cache.Del(ctx, "mfa:setup:attempts:"+userID.String())
	return nil
}

func (s *AuthService) AdminTOTPStatus(ctx context.Context, userID uuid.UUID) (map[string]bool, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, apierr.NotFound("Admin account not found")
	}
	if user.Role != "admin" {
		return nil, apierr.Forbidden("Platform admin access required")
	}
	return map[string]bool{"enabled": user.MFAEnabled}, nil
}

func (s *AuthService) DisableAdminTOTP(ctx context.Context, userID uuid.UUID, password, code string) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil || user.Role != "admin" {
		return apierr.NotFound("Admin account not found")
	}
	if !user.MFAEnabled {
		return apierr.BadRequest("Authenticator MFA is not enabled")
	}
	if err := s.countMFAAttempt(ctx, "mfa:disable:attempts:"+userID.String(), 5, 15*time.Minute); err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return apierr.Unauthorized("Password is incorrect")
	}
	secret, err := s.decryptMFASecret(user.MFASecretEncrypted)
	if err != nil {
		return err
	}
	if !validTOTP(secret, code, time.Now()) {
		return apierr.Unauthorized("Invalid authenticator code")
	}
	user.MFAEnabled = false
	user.MFASecretEncrypted = ""
	user.MFAPendingSecretEncrypted = ""
	user.MFAPendingExpiresAt = nil
	if err := s.repo.UpdateUser(user); err != nil {
		return err
	}
	_ = s.cache.Del(ctx, "mfa:disable:attempts:"+userID.String())
	return nil
}

func (s *AuthService) countMFAAttempt(ctx context.Context, key string, max int64, ttl time.Duration) error {
	if s.cache == nil {
		return apierr.New(503, "unavailable", "Authenticator verification is temporarily unavailable")
	}
	n, err := s.cache.Incr(ctx, key)
	if err != nil {
		return apierr.New(503, "unavailable", "Authenticator verification is temporarily unavailable")
	}
	if n == 1 {
		_ = s.cache.Expire(ctx, key, ttl)
	}
	if n > max {
		return apierr.Locked("Too many authenticator attempts. Try again later")
	}
	return nil
}

// CompleteAdminMFALogin exchanges a password-verified MFA challenge for a
// normal session. Redis rate limits attempts and makes each challenge one-use.
func (s *AuthService) CompleteAdminMFALogin(ctx context.Context, challenge, code string, ip string) (*security.TokenPair, *User, error) {
	claims, err := s.tokens.ParseMFAChallenge(challenge)
	if err != nil || claims.Role != "admin" || claims.ID == "" {
		return nil, nil, apierr.Unauthorized("Invalid or expired login challenge")
	}
	if s.cache == nil {
		return nil, nil, apierr.New(503, "unavailable", "Authenticator verification is temporarily unavailable")
	}
	attemptKey := "mfa:login:attempts:" + claims.ID
	attempts, err := s.cache.Incr(ctx, attemptKey)
	if err != nil {
		return nil, nil, apierr.New(503, "unavailable", "Authenticator verification is temporarily unavailable")
	}
	if attempts == 1 {
		_ = s.cache.Expire(ctx, attemptKey, 5*time.Minute)
	}
	if attempts > 5 {
		return nil, nil, apierr.Locked("Too many authenticator attempts. Sign in again")
	}
	user, err := s.repo.GetUserByID(claims.UserID)
	if err != nil || !user.IsActive || user.Role != "admin" || !user.MFAEnabled {
		return nil, nil, apierr.Unauthorized("Invalid or expired login challenge")
	}
	secret, err := s.decryptMFASecret(user.MFASecretEncrypted)
	if err != nil {
		return nil, nil, err
	}
	if !validTOTP(secret, code, time.Now()) {
		return nil, nil, apierr.Unauthorized("Invalid authenticator code")
	}
	usedKey := "mfa:login:used:" + claims.ID
	if ok, err := s.cache.Client().SetNX(ctx, usedKey, "1", 5*time.Minute).Result(); err != nil || !ok {
		return nil, nil, apierr.Unauthorized("Login challenge has already been used")
	}
	now := time.Now()
	user.LastLogin = &now
	user.LastLoginIP = ip
	if err := s.repo.UpdateUser(user); err != nil {
		return nil, nil, err
	}
	pair, err := s.issueMFATokens(user)
	if err != nil {
		return nil, nil, err
	}
	s.bus.Publish("user.logged_in", map[string]interface{}{"user_id": user.ID, "ip": ip})
	_ = s.cache.Del(ctx, attemptKey)
	return pair, user, nil
}

func (s *AuthService) issueMFATokens(user *User) (*security.TokenPair, error) {
	var orgID, storeID uuid.UUID
	if user.OrganisationID != nil {
		orgID = *user.OrganisationID
	}
	if user.StoreID != nil {
		storeID = *user.StoreID
	}
	return s.tokens.IssueMFA(user.ID, user.Role, orgID, storeID)
}

func (s *AuthService) encryptMFASecret(plain string) (string, error) {
	key := sha256.Sum256([]byte(s.cfg.MFAEncryptionKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(plain), []byte("ziada-admin-totp"))
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (s *AuthService) decryptMFASecret(encoded string) (string, error) {
	key := sha256.Sum256([]byte(s.cfg.MFAEncryptionKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	sealed, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(sealed) < aead.NonceSize() {
		return "", errors.New("could not read authenticator secret; contact an administrator")
	}
	nonce, ciphertext := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ciphertext, []byte("ziada-admin-totp"))
	if err != nil {
		return "", errors.New("could not decrypt authenticator secret; check MFA_ENCRYPTION_KEY")
	}
	return string(plain), nil
}

func validTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}
	counter := now.Unix() / int64(totpPeriod.Seconds())
	for drift := int64(-1); drift <= 1; drift++ {
		var msg [8]byte
		binary.BigEndian.PutUint64(msg[:], uint64(counter+drift))
		mac := hmac.New(sha1.New, key)
		_, _ = mac.Write(msg[:])
		sum := mac.Sum(nil)
		offset := sum[len(sum)-1] & 0x0f
		value := (binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff) % 1_000_000
		expected := fmt.Sprintf("%06d", value)
		if subtle.ConstantTimeCompare([]byte(code), []byte(expected)) == 1 {
			return true
		}
	}
	return false
}
