package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	GeneralParams GeneralParams
	MainDBParams  MainDBParams
	AuthDBParams  AuthDBParams
	UDPParams     UDPParams
	S3Params      S3Params
}

type GeneralParams struct {
	Env         string
	SecretKey   string
	HTTPaddress string
}

type MainDBParams struct {
	Username string
	Password string
	Name     string
	Port     int
	Host     string
	Timeout  int
}

type AuthDBParams struct {
	Host     string
	Username string
	Password string
}

type UDPParams struct {
	Address string
	Port    int
}

type S3Params struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
}

type ConfigManager struct {
	v      *viper.Viper
	config *Config
}

// NewConfigManager creates new config manager that handles
// all viper config options and loads a config from yaml
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

// Extracting data from yaml file and loading into Config
func (cm *ConfigManager) loadConfig() error {
	cm.config = &Config{
		GeneralParams: GeneralParams{
			Env:         cm.v.GetString("general_params.env"),
			SecretKey:   cm.v.GetString("general_params.secret_key"),
			HTTPaddress: cm.v.GetString("general_params.http_server_address"),
		},
		MainDBParams: MainDBParams{
			Username: cm.v.GetString("main_db_params.db_username"),
			Password: cm.v.GetString("main_db_params.db_password"),
			Name:     cm.v.GetString("main_db_params.db_name"),
			Port:     cm.v.GetInt("main_db_params.db_port"),
			Host:     cm.v.GetString("main_db_params.db_host"),
			Timeout:  cm.v.GetInt("main_db_params.db_timeout"),
		},
		AuthDBParams: AuthDBParams{
			Host:     cm.v.GetString("auth_db_params.db_host"),
			Username: cm.v.GetString("auth_db_params.db_username"),
			Password: cm.v.GetString("auth_db_params.db_password"),
		},
		UDPParams: UDPParams{
			Address: cm.v.GetString("udp_params.address"),
			Port:    cm.v.GetInt("udp_params.port"),
		},
		S3Params: S3Params{
			Endpoint:        cm.v.GetString("s3_params.endpoint"),
			AccessKeyID:     cm.v.GetString("s3_params.access_key_id"),
			SecretAccessKey: cm.v.GetString("s3_params.secret_access_key"),
			UseSSL:          cm.v.GetBool("s3_params.use_ssl"),
			BucketName:      cm.v.GetString("s3_params.bucket_name"),
		},
	}
	return nil
}

// Geting config instance
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// Compiling a string to connect to main db
func (db *MainDBParams) GetDSN() string {
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
	// Checking secret key
	if c.GeneralParams.SecretKey == "" {
		return fmt.Errorf("parameter secret_key is required")
	}

	// Checking http address
	if c.GeneralParams.HTTPaddress == "" {
		return fmt.Errorf("parameter http_server_address is requred")
	}

	// Checking out enviroment variable
	switch c.GeneralParams.Env {
	case "dev", "prod", "test":
	default:
		return fmt.Errorf("env parameter is invalid: %s. try dev/prod/test instead", c.GeneralParams.Env)
	}

	// Checking MainDbparams
	for name, mainDbConf := range map[string]MainDBParams{
		"MainDB": c.MainDBParams,
	} {
		if mainDbConf.Host == "" {
			return fmt.Errorf("%s: host is required", name)
		}
		if mainDbConf.Username == "" {
			return fmt.Errorf("%s: username is required", name)
		}
		if mainDbConf.Password == "" {
			return fmt.Errorf("%s: password is requred", name)
		}
		if mainDbConf.Port != 5432 {
			return fmt.Errorf("%s: port is invalid", name)
		}
	}

	// Checking AuthDbParams
	for name, authDbConf := range map[string]AuthDBParams{
		"AuthDB": c.AuthDBParams,
	} {
		if authDbConf.Host == "" {
			return fmt.Errorf("%s: host is required", name)
		}
		if authDbConf.Username == "" {
			return fmt.Errorf("%s: username is required", name)
		}
		if authDbConf.Password == "" {
			return fmt.Errorf("%s: password is required", name)
		}
	}

	// Checking UDP params
	if c.UDPParams.Address == "" {
		return fmt.Errorf("UDP address is required")
	}
	if c.UDPParams.Port <= 0 || c.UDPParams.Port > 65535 {
		return fmt.Errorf("UDP port must be between 1 and 65535")
	}

	// Checking S3 params
	if c.S3Params.Endpoint == "" {
		return fmt.Errorf("S3 endpoint is required")
	}
	if c.S3Params.AccessKeyID == "" {
		return fmt.Errorf("S3 access_key id is required")
	}
	if c.S3Params.SecretAccessKey == "" {
		return fmt.Errorf("S3 secret_access_key is required")
	}
	if c.S3Params.BucketName == "" {
		return fmt.Errorf("S3 bucket name is required")
	}

	return nil
}
