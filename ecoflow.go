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
	"os"

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
	triggerParameterStore(client)
	startStatLoop()

	done := make(chan bool, 1)
	setupGracefulShutdown(done)
	if !MqttDisable {
		InitMqtt(user, password)
	}
	<-done
}

func refreshDeviceList(client *ecoflow.Client) {
	//get all linked ecoflow devices. Returns SN and online status
	list, err := client.GetDeviceList(context.Background())
	if err != nil {
		services.ServerMessage("Shutdown ... error getting device list: %v", err)
		log.Log.Fatalf("Error getting device list: %v", err)
	}
	devices = list
}

func SetEnvironmentPowerConsumption(value int) {
	accessKey := os.Getenv("ECOFLOW_ACCESS_KEY")
	secretKey := os.Getenv("ECOFLOW_SECRET_KEY")

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	client := ecoflow.NewEcoflowClient(accessKey, secretKey)

	request := make(map[string]interface{})
	client.SetDeviceParameter(context.Background(), request)
}
