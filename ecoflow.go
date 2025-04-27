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
	"bytes"
	"context"
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
	//get all linked ecoflow devices. Returns SN and online status
	list, err := client.GetDeviceList(context.Background())
	if err != nil {
		services.ServerMessage("Shutdown ... error getting device list: %v", err)
		log.Log.Fatalf("Error getting device list: %v", err)
	}
	devices = list

	// Start statistics output
	triggerParameterStore(client)
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				var buffer bytes.Buffer
				buffer.WriteString("Statistics:\n")
				for k, v := range mapStatMqtt {
					buffer.WriteString(fmt.Sprintf("  %s got http=%03d mqtt=%03d messages\n", k, v.httpCounter, v.mqttCounter))
				}
				for k, v := range mapStatDatabase {
					buffer.WriteString(fmt.Sprintf("  %s inserted %03d records\n", k, v.counter))
				}
				services.ServerMessage(buffer.String())
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	done := make(chan bool, 1)
	setupGracefulShutdown(done)
	if !MqttDisable {
		InitMqtt(user, password)
	}
	<-done
}
