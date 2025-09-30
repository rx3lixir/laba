package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	GeneralParams   GeneralParams
	UserDBParams    DBParams
	MessageDBParams DBParams
	AuthDBParams    DBParams
}

type GeneralParams struct {
	Env       string
	SecretKey string
}

type DBParams struct {
	Host     string
	Username string
	Password string
	Name     string
	Port     int
	Timeout  int
}

type ConfigManager struct {
	v      *viper.Viper
	config *Config
}

func NewConfigManager(configPath string) (*ConfigManager, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	v.AutomaticEnv()
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cm := &ConfigManager{v: v}

	if err := cm.loadConfig(); err != nil {
		return nil, err
	}

	return cm, nil
}

func (cm *ConfigManager) loadConfig() error {
	cm.config = &Config{
		GeneralParams: GeneralParams{
			Env:       cm.v.GetString("general_params.env"),
			SecretKey: cm.v.GetString("general_params.secret_key"),
		},
		UserDBParams: DBParams{
			Host:     cm.v.GetString("user_db_params.db_host"),
			Username: cm.v.GetString("user_db_params.db_username"),
			Password: cm.v.GetString("user_db_params.db_password"),
			Name:     cm.v.GetString("user_db_params.db_name"),
			Port:     cm.v.GetInt("user_db_params.db_port"),
			Timeout:  cm.v.GetInt("user_db_params.db_timeout"),
		},
		MessageDBParams: DBParams{
			Host:     cm.v.GetString("messages_db_params.db_host"),
			Username: cm.v.GetString("messages_db_params.db_username"),
			Password: cm.v.GetString("messages_db_params.db_password"),
			Name:     cm.v.GetString("messages_db_params.db_name"),
			Port:     cm.v.GetInt("messages_db_params.db_port"),
			Timeout:  cm.v.GetInt("messages_db_params.db_timeout"),
		},
		AuthDBParams: DBParams{
			Host:     cm.v.GetString("auth_db_params.db_host"),
			Username: cm.v.GetString("auth_db_params.db_username"),
			Password: cm.v.GetString("auth_db_params.db_password"),
		},
	}
	return nil
}

func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

func (db *DBParams) GetDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?connect_timeout=%d&sslmode=disable",
		db.Username,
		db.Password,
		db.Host,
		db.Port,
		db.Name,
		db.Timeout,
	)
}

func (c *Config) Validate() error {
	if c.GeneralParams.SecretKey == "" {
		return fmt.Errorf("parameter secret_key is required")
	}

	for name, dbConfig := range map[string]DBParams{
		"UserDB":     c.UserDBParams,
		"MessagesDB": c.MessageDBParams,
		"AuthDB":     c.AuthDBParams,
	} {
		if dbConfig.Host == "" {
			return fmt.Errorf("%s: host is required", name)
		}
		if dbConfig.Username == "" {
			return fmt.Errorf("%s: username is required", name)
		}
	}

	return nil
}
