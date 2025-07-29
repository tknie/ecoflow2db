/*
* Copyright 2025 Thorsten A. Knieling
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
	"os"

	"github.com/tknie/ecoflow"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var mqttid common.RegDbID
var MqttDisable = false
var client *ecoflow.Client

var serialNumberConverter string

// InitEcoflow init ecoflow MQTT
func InitEcoflow() {
	user := adapter.EcoflowConfig.User
	password := adapter.EcoflowConfig.Password

	accessKey := os.ExpandEnv(adapter.EcoflowConfig.AccessKey)
	secretKey := os.ExpandEnv(adapter.EcoflowConfig.SecretKey)

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	client = ecoflow.NewClient(accessKey, secretKey)
	client.RefreshDeviceList()
	// Start statistics output
	go httpParameterStore(client)
	startStatLoop()

	done := make(chan bool, 1)
	setupGracefulShutdown(done)
	if !MqttDisable {
		InitMqtt(user, password)
	}
	<-done
}

// InitMqtt initialize MQTT listener
func InitMqtt(user, password string) {
	ecoflow.InitMqtt(user, password)

	mqttid = connnectDatabase()
	log.Log.Debugf("Connecting MQTT Ecoflow connect")
	services.ServerMessage("Connecting MQTT client")
	ecoflow.InitMqtt(user, password)
	log.Log.Debugf("Wait for Ecoflow disconnect")
	services.ServerMessage("Waiting for MQTT data")

}

func SetEnvironmentPowerConsumption(value float64) {
	client.SetEnvironmentPowerConsumption(serialNumberConverter, value)
}
