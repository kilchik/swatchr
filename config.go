package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Params      ConfigParams
	DefaultPath string
}

func InitConfig(path string, c *Config) error {
	if _, err := toml.DecodeFile(path, &c.Params); err != nil {
		return err
	}
	if err := c.Validate(); err != nil {
		return err
	}

	return nil
}

type ConfigParams struct {
	Addr        string `toml:"addr"`
	Port        int    `toml:"port"`
	CatalogPath string `toml:"catalog_path"`
	StoragePath string `toml:"storage_path"`
	Quota       int64  `toml:"quota"`
}

func (c *Config) Validate() error {
	logPrefix := "parsing config: "
	if len(c.Params.Addr) == 0 {
		return fmt.Errorf(logPrefix + "listen_addr is not set")
	}
	if c.Params.Port == 0 {
		return fmt.Errorf(logPrefix + "port is not set")
	}
	if len(c.Params.CatalogPath) == 0 {
		logW.Println(logPrefix + "catalog_path is not set")
	}
	if len(c.Params.StoragePath) == 0 {
		return fmt.Errorf(logPrefix + "storage_path is not set")
	}
	if c.Params.Quota == 0 {
		return fmt.Errorf(logPrefix + "quota is not set")
	}
	return nil
}
