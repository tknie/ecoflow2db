/*
* Copyright 2025-2026 Thorsten A. Knieling
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
 */

package ecoflow2db

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/stretchr/testify/assert/yaml"
	"github.com/tknie/log"
)

type adapterConfig struct {
	DefaultConfig  *defaultConfig  `yaml:"default"`
	DatabaseConfig *databaseConfig `yaml:"database"`
	EcoflowConfig  *ecoflowConfig  `yaml:"ecoflow"`
}

type defaultConfig struct {
	BaseRequest   int64  `yaml:"baseWatt"`
	MaxRequest    int64  `yaml:"maximumWatt"`
	LowerBatLimit int64  `yaml:"lowerBatLimit"`
	UpperBatLimit int64  `yaml:"upperBatLimit"`
	Debug         string `yaml:"debug"`
}

type databaseConfig struct {
	Target      string `yaml:"target"`
	TableName   string `yaml:"tableName"`
	Table       string `yaml:"ecoflowTable"`
	EnergyTable string `yaml:"energyTable"`
}

type ecoflowConfig struct {
	User           string   `yaml:"user"`
	Password       string   `yaml:"password"`
	AccessKey      string   `yaml:"accessKey"`
	SecretKey      string   `yaml:"secretKey"`
	MicroConverter []string `yaml:"microConverter"`
}

const defaultBaseRequest = 170
const defaultMaxRequest = 250

var adapter = &adapterConfig{
	DefaultConfig:  &defaultConfig{BaseRequest: defaultBaseRequest},
	DatabaseConfig: &databaseConfig{},
	EcoflowConfig:  &ecoflowConfig{},
}

var FlowLoopSeconds = DefaultSeconds
var header = ""

// ReadConfig read config file
func readConfig(file string) ([]byte, error) {
	configFile, err := os.Open(file)
	if err != nil {
		log.Log.Debugf("Open file error: %#v", err)
		return nil, fmt.Errorf("open file err of %s: %v", file, err)
	}
	defer configFile.Close()

	fi, _ := configFile.Stat()
	log.Log.Debugf("File size=%d", fi.Size())
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, configFile)
	if err != nil {
		log.Log.Debugf("Read file error: %#v", err)
		return nil, fmt.Errorf("read file err of %s: %v", file, err)
	}
	return buffer.Bytes(), nil
}

func LoadConfig(file string) {
	if file != "" {
		fileEnvResolved := os.ExpandEnv(file)

		data, err := readConfig(fileEnvResolved)
		if err != nil {
			log.Log.Fatalf("Error loading config: %s", file)
		}
		err = yaml.Unmarshal(data, adapter)
		if err != nil {
			fmt.Println("Error loading config file:", err)
			log.Log.Fatalf("Error unmarshal config %s: %v", file, err)
		}
		if adapter.DefaultConfig.BaseRequest == 0 {
			adapter.DefaultConfig.BaseRequest = defaultBaseRequest
		}
		if adapter.DefaultConfig.MaxRequest == 0 {
			adapter.DefaultConfig.MaxRequest = defaultMaxRequest
		}
	}
	if adapter.DatabaseConfig.TableName == "" {
		adapter.DatabaseConfig.TableName = os.Getenv("ECOFLOW_DB_TABLENAME")
	}
	if adapter.DatabaseConfig.Table == "" {
		adapter.DatabaseConfig.Table = os.Getenv("ECOFLOW_DB_TABLENAME")
	}
}
