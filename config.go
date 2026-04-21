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
	"github.com/tknie/services"
)

type adapterConfig struct {
	DefaultConfig  *defaultConfig  `yaml:"default"`
	DatabaseConfig *databaseConfig `yaml:"database"`
	Mqtt           *mqttConfig     `yaml:"mqtt"`
	EcoflowConfig  *ecoflowConfig  `yaml:"ecoflow"`
}

type defaultConfig struct {
	DynamicRequest          bool   `yaml:"dynamicRequest"`
	RealtimeRequest         bool   `yaml:"realtimeRequest"`
	WaitAfterRequestSeconds int64  `yaml:"waitAfterRequestSeconds"`
	BaseRequest             int64  `yaml:"baseWatt"`
	UpperBatLimit           int64  `yaml:"upperBatLimit"`
	IntermediateSize        int64  `yaml:"intermediateSize"`
	Debug                   string `yaml:"debug"`
}

type mqttConfig struct {
	Server              string   `yaml:"server"`
	Username            string   `yaml:"username"`
	Password            string   `yaml:"password"`
	LoopIntervalSeconds int      `yaml:"loopIntervalSeconds"`
	Qos                 int      `yaml:"qos"`
	Clientid            string   `yaml:"clientID"`
	MaxTries            int      `yaml:"maxTries"`
	Topics              []*Topic `yaml:"topics"`
}

type databaseConfig struct {
	Target      string `yaml:"target"`
	TableName   string `yaml:"tableName"`
	Table       string `yaml:"ecoflowTable"`
	EnergyTable string `yaml:"energyTable"`
}

type ecoflowConfig struct {
	CheckBatteryLimits      bool     `yaml:"checkBatteryLimits"`
	CheckBatteryLimitsTests bool     `yaml:"checkBatteryLimitsTests"`
	User                    string   `yaml:"user"`
	Password                string   `yaml:"password"`
	AccessKey               string   `yaml:"accessKey"`
	SecretKey               string   `yaml:"secretKey"`
	MicroConverter          []string `yaml:"microConverter"`
	Battery                 []string `yaml:"battery"`
}

const defaultBaseRequest = 100
const defaultMaxRequest = 250

var adapter = &adapterConfig{
	DefaultConfig:  &defaultConfig{BaseRequest: defaultBaseRequest},
	DatabaseConfig: &databaseConfig{},
	EcoflowConfig:  &ecoflowConfig{},
}

var FlowLoopSeconds = DefaultSeconds

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
	services.InitWatcher(file, file, watchConfig)
	evaluateConfig(file)
}

func evaluateConfig(file string) {
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
		if adapter.DefaultConfig.UpperBatLimit == 0 {
			adapter.DefaultConfig.UpperBatLimit = defaultMaxRequest
		}
	}
	if adapter.DatabaseConfig.TableName == "" {
		adapter.DatabaseConfig.TableName = os.Getenv("ECOFLOW_DB_TABLENAME")
	}
	if adapter.DatabaseConfig.Table == "" {
		adapter.DatabaseConfig.Table = os.Getenv("ECOFLOW_DB_TABLENAME")
	}
	switch {
	case adapter.DefaultConfig.DynamicRequest && adapter.DefaultConfig.RealtimeRequest:
		adapter.DefaultConfig.DynamicRequest = false
		services.ServerMessage("Attention: Dynamic is switched off and relatime request is enabled, power request will be updated")
	case adapter.DefaultConfig.DynamicRequest:
		services.ServerMessage("Dynamic request is enabled, power request will be updated if needed")
	case adapter.DefaultConfig.RealtimeRequest:
		services.ServerMessage("Realtime request is enabled, power request will be updated if needed")
	default:
		services.ServerMessage("Dynamic request is disabled, power request will not be updated")
	}

}

func watchConfig(s string, a any) error {
	log.Log.Infof("Configuration file %s/%s changed, reload it", s, a.(string))
	evaluateConfig(a.(string))
	return nil
}
