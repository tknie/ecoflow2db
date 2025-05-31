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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/tess1o/go-ecoflow"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var quit = make(chan struct{})

// InitEcoflow init ecoflow MQTT
func InitEcoflow() {
	user := os.Getenv("ECOFLOW_USER")
	password := os.Getenv("ECOFLOW_PASSWORD")

	accessKey := os.Getenv("ECOFLOW_ACCESS_KEY")
	secretKey := os.Getenv("ECOFLOW_SECRET_KEY")

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	client := ecoflow.NewEcoflowClient(accessKey, secretKey)
	refreshDeviceList(client)
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

// refreshDeviceList refresh device list using HTTP device list request
func refreshDeviceList(client *ecoflow.Client) {
	//get all linked ecoflow devices. Returns SN and online status
	list, err := client.GetDeviceList(context.Background())
	if err != nil {
		services.ServerMessage("Error getting device list: %v", err)
	} else {
		devices = list
	}
}

// SetEnvironmentPowerConsumption set new environment consumption value
func SetEnvironmentPowerConsumption(value float64) {
	accessKey := os.Getenv("ECOFLOW_ACCESS_KEY")
	secretKey := os.Getenv("ECOFLOW_SECRET_KEY")
	if value > 6000 || value < 0 {
		services.ServerMessage("Value %f out of range in 0:1000", value)
		return
	}
	sn := os.Getenv("ECOFLOW_DEVICE_SN")

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	client := ecoflow.NewEcoflowClient(accessKey, secretKey)

	params := make(map[string]interface{})
	params["permanentWatts"] = value
	cmdReq := ecoflow.CmdSetRequest{
		Id:      fmt.Sprint(time.Now().UnixMilli()),
		CmdCode: "WN511_SET_PERMANENT_WATTS_PACK",
		Sn:      sn,
		Params:  params,
	}

	jsonData, err := json.Marshal(cmdReq)
	if err != nil {
		services.ServerMessage("Error marshal data: %v", err)
		return
	}

	var req map[string]interface{}

	err = json.Unmarshal(jsonData, &req)
	if err != nil {
		services.ServerMessage("Error unmarshal data: %v", err)
		return
	}
	cmd, err := client.SetDeviceParameter(context.Background(), req)

	if err != nil {
		services.ServerMessage("Error set device parameter: %v", err)
	} else {
		services.ServerMessage("Set device parameter to %f: %s", value, cmd.Message)
	}
}
