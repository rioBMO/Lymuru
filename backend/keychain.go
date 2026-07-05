package backend

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "lymuru"
	keyringAPIID   = "telegram_api_id"
	keyringAPIHash = "telegram_api_hash"
	keyringPhone   = "telegram_phone"
)

// SaveSidecarCredential stores a value in the OS keychain.
func SaveSidecarCredential(key, value string) error {
	if err := keyring.Set(keyringService, key, value); err != nil {
		return fmt.Errorf("keychain save %s: %w", key, err)
	}
	return nil
}

// LoadSidecarCredential retrieves a value from the OS keychain.
func LoadSidecarCredential(key string) (string, error) {
	v, err := keyring.Get(keyringService, key)
	if err != nil {
		return "", fmt.Errorf("keychain load %s: %w", key, err)
	}
	return v, nil
}

// DeleteSidecarCredential removes a value from the OS keychain.
func DeleteSidecarCredential(key string) error {
	if err := keyring.Delete(keyringService, key); err != nil {
		return fmt.Errorf("keychain delete %s: %w", key, err)
	}
	return nil
}

// SetSidecarCredentials stores all Telegram credentials at once.
func SetSidecarCredentials(apiID, apiHash, phone string) error {
	if err := SaveSidecarCredential(keyringAPIID, apiID); err != nil {
		return err
	}
	if err := SaveSidecarCredential(keyringAPIHash, apiHash); err != nil {
		return err
	}
	if err := SaveSidecarCredential(keyringPhone, phone); err != nil {
		return err
	}
	return nil
}

// GetSidecarCredentials loads all Telegram credentials from the keychain.
// Returns empty strings for any missing values.
func GetSidecarCredentials() (apiID, apiHash, phone string) {
	apiID, _ = LoadSidecarCredential(keyringAPIID)
	apiHash, _ = LoadSidecarCredential(keyringAPIHash)
	phone, _ = LoadSidecarCredential(keyringPhone)
	return
}
