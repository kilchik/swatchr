package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type Config interface {
	Params() interface{}
	Validate() error
}

func InitConfig(path string, c Config) error {
	if _, err := toml.DecodeFile(path, c.Params()); err != nil {
		return err
	}
	if err := c.Validate(); err != nil {
		return err
	}

	return nil
}

type configParams struct {
	ListenAddr  string `toml:"listen_addr"`
	CatalogPath string `toml:"catalog_path"`
	StoragePath string `toml:"storage_path"`
	Quota       int64  `toml:"quota"`
}

type configImpl struct {
	params      configParams
	defaultPath string
}

func (c *configImpl) Params() interface{} {
	return &c.params
}

func (c *configImpl) Validate() error {
	logPrefix := "parsing config: "
	if len(c.params.ListenAddr) == 0 {
		return fmt.Errorf(logPrefix + "listen_addr is not set")
	}
	if len(c.params.CatalogPath) == 0 {
		logW.Println(logPrefix + "catalog_path is not set")
	}
	if len(c.params.StoragePath) == 0 {
		return fmt.Errorf(logPrefix + "storage_path is not set")
	}
	if c.params.Quota == 0 {
		return fmt.Errorf(logPrefix + "quota is not set")
	}
	return nil
}
