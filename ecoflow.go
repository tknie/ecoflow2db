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
	"context"
	"fmt"
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

func prepareEcoflow() {

	accessKey := os.ExpandEnv(adapter.EcoflowConfig.AccessKey)
	secretKey := os.ExpandEnv(adapter.EcoflowConfig.SecretKey)

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	client = ecoflow.NewClient(accessKey, secretKey)
	client.RefreshDeviceList()
	serialNumberConverter = os.Getenv("ECOFLOW_DEVICE_SN")
}

// InitEcoflow init ecoflow MQTT
func InitEcoflow() {
	prepareEcoflow()
	user := adapter.EcoflowConfig.User
	password := adapter.EcoflowConfig.Password
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
	prepareEcoflow()
	client.SetEnvironmentPowerConsumption(serialNumberConverter, value)
}

func SetCarACOn(sn string, turnOn bool) {
	prepareEcoflow()

	resp, err := client.SetCarACOn(sn, turnOn)

	fmt.Println(err, resp)
}

func ListDevices() {
	prepareEcoflow()
	devs, err := client.GetDeviceList(context.Background())
	if err != nil {
		fmt.Println("List device error:", err)
		return
	}
	for i, d := range devs.Devices {
		fmt.Println(i, d.Online, d.SN)
		m, err := client.GetDeviceAllParameters(context.Background(), d.SN)
		if err != nil {
			fmt.Println("Get info of device error:", err)
			return
		}
		fmt.Println(m["bms_emsStatus.bmsModel"], m["pd.model"])
		fmt.Println(m)
	}
}
