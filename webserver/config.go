package main

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"private-sphinx-docs/libs"
	db "private-sphinx-docs/services/database"
)

type Config struct {
	App struct {
		Port      int    `mapstructure:"port"`
		DocFolder string `mapstructure:"doc_folder"`
		TLS       struct {
			CertFile string `mapstructure:"cert_file"`
			KeyFile  string `mapstructure:"key_file"`
		} `mapstructure:"tls"`
	} `mapstructure:"app"`

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DbName   string `mapstructure:"db_name"`
		SSLMode  string `mapstructure:"ssl_mode"`
		Migrate  bool   `mapstructure:"migrate"`
	} `mapstructure:"database"`
}

func (c *Config) DbOption() *db.DbOption {
	return &db.DbOption{
		Host:     c.Database.Host,
		Port:     c.Database.Port,
		User:     c.Database.User,
		Password: c.Database.Password,
		DbName:   c.Database.DbName,
		SSLMode:  c.Database.SSLMode,
	}
}

func (c *Config) HasCert() bool {
	tls := c.App.TLS

	exists := func(filepath string) bool {
		filepath = strings.TrimSpace(filepath)
		return libs.PathExists(filepath) && libs.PathType(filepath) == libs.File
	}

	return exists(tls.CertFile) && exists(tls.KeyFile)
}

func (c *Config) TLSFiles() (certFile, keyFile string) {
	tls := c.App.TLS
	return tls.CertFile, tls.KeyFile
}

func ReadConfig() (*Config, error) {
	if err := setConfigDirectory(); err != nil {
		return nil, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "could not read in configuration")
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal config")
	}

	return config, nil
}

func setConfigDirectory() error {
	// this is where the default config file is placed in the docker image
	viper.AddConfigPath("/app")

	// this is where the user specified (via env var) config file is place
	if fp := os.Getenv("PSD_CONFIG_FILE_PATH"); libs.PathExists(fp) {
		if libs.PathType(fp) == libs.File {
			// if user specified the config file directly, change file name to config.yaml
			// and set path to the parent directory
			dir, name := filepath.Split(fp)

			if name = strings.ToLower(name); name != "config.yml" {
				err := os.Rename(fp, filepath.Join(dir, "config.yml"))
				if err != nil {
					return errors.Wrapf(err, "unable to rename user given config file at %s to config.yml", fp)
				}
			}
			viper.AddConfigPath(dir)
		} else {
			viper.AddConfigPath(fp)
		}
	}

	// search for the config file where the application binary is
	workDir, err := os.Executable()
	if err != nil {
		return err
	}
	viper.AddConfigPath(filepath.Dir(workDir))

	// alternative path to store the config file. This is used in cases where the
	// user does not have full write permissions
	usr, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "could not get user directory")
	}
	viper.AddConfigPath(filepath.Join(usr.HomeDir, "psd"))

	// set root directory where we're writing the program
	_, file, _, _ := runtime.Caller(0)
	workDir = filepath.Dir(file)
	viper.AddConfigPath(workDir)

	return nil
}
