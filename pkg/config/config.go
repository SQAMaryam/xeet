package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey            string `mapstructure:"api_key"`
	APISecret         string `mapstructure:"api_secret"`
	AccessToken       string `mapstructure:"access_token"`
	AccessTokenSecret string `mapstructure:"access_token_secret"`
	BearerToken       string `mapstructure:"bearer_token"`
	UserID            string `mapstructure:"user_id"`
	Username          string `mapstructure:"username"`
}

type ConfigManager struct {
	configPath string
	encKey     []byte
}

func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".xeet.yaml")

	// Generate or load encryption key
	keyPath := filepath.Join(homeDir, ".xeet.key")
	encKey, err := loadOrGenerateKey(keyPath)
	if err != nil {
		return nil, err
	}

	return &ConfigManager{
		configPath: configPath,
		encKey:     encKey,
	}, nil
}

func (cm *ConfigManager) Load() (*Config, error) {
	viper.SetConfigFile(cm.configPath)

	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Decrypt sensitive fields
	if config.APISecret != "" {
		decrypted, err := cm.decrypt(config.APISecret)
		if err != nil {
			return nil, err
		}
		config.APISecret = decrypted
	}

	if config.AccessTokenSecret != "" {
		decrypted, err := cm.decrypt(config.AccessTokenSecret)
		if err != nil {
			return nil, err
		}
		config.AccessTokenSecret = decrypted
	}

	return &config, nil
}

func (cm *ConfigManager) Save(config *Config) error {
	// Encrypt sensitive fields before saving
	configCopy := *config

	if configCopy.APISecret != "" {
		encrypted, err := cm.encrypt(configCopy.APISecret)
		if err != nil {
			return err
		}
		configCopy.APISecret = encrypted
	}

	if configCopy.AccessTokenSecret != "" {
		encrypted, err := cm.encrypt(configCopy.AccessTokenSecret)
		if err != nil {
			return err
		}
		configCopy.AccessTokenSecret = encrypted
	}

	viper.SetConfigFile(cm.configPath)
	viper.Set("api_key", configCopy.APIKey)
	viper.Set("api_secret", configCopy.APISecret)
	viper.Set("access_token", configCopy.AccessToken)
	viper.Set("access_token_secret", configCopy.AccessTokenSecret)
	viper.Set("bearer_token", configCopy.BearerToken)
	viper.Set("user_id", configCopy.UserID)
	viper.Set("username", configCopy.Username)

	return viper.WriteConfig()
}

func (cm *ConfigManager) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(cm.encKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (cm *ConfigManager) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(cm.encKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("invalid ciphertext")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func loadOrGenerateKey(keyPath string) ([]byte, error) {
	if _, err := os.Stat(keyPath); err == nil {
		return os.ReadFile(keyPath)
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, err
	}

	return key, nil
}
