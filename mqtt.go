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
	"encoding/json"
	"fmt"
	"math"
	reflect "reflect"
	"sort"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tess1o/go-ecoflow"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var ecoclient *ecoflow.MqttClient

var devices *ecoflow.DeviceListResponse
var tableName string

var mqttid common.RegDbID

var MqttDisable = false

// InitMqtt initialize MQTT listener
func InitMqtt(user, password string) {
	services.ServerMessage("Initialize MQTT client")
	configuration := ecoflow.MqttClientConfiguration{
		Email:            user,
		Password:         password,
		OnConnect:        OnConnect,
		OnConnectionLost: OnConnectionLost,
		OnReconnect:      OnReconnect,
	}
	var err error
	ecoclient, err = ecoflow.NewMqttClient(context.Background(), configuration)
	if err != nil {
		services.ServerMessage("Shuting down ... error creating MQTT client: %v", err)
		log.Log.Fatalf("Error new MQTT client: %v", err)
	}
	mqttid = connnectDatabase()
	log.Log.Debugf("Connecting MQTT Ecoflow connect")
	services.ServerMessage("Connecting MQTT client")
	ecoclient.Connect()
	log.Log.Debugf("Wait for Ecoflow disconnect")
	services.ServerMessage("Waiting for MQTT data")

}

// MessageHandler message handle called if MQTT event entered
func MessageHandler(_ mqtt.Client, msg mqtt.Message) {
	serialNumber := getSnFromTopic(msg.Topic())
	stat := getStatEntry(serialNumber)
	stat.mu.Lock()
	defer stat.mu.Unlock()

	stat.mqttCounter++

	log.Log.Infof("Received message of device %s at %v\n", serialNumber, time.Now().Format(layout))

	log.Log.Debugf("received message on topic %s; body (retain: %t):\n%s", msg.Topic(),
		msg.Retained(), FormatByteBuffer("MQTT Body", msg.Payload()))
	payload := msg.Payload()

	data := make(map[string]interface{})
	err := json.Unmarshal(payload, &data)
	if err == nil {
		log.Log.Debugf("JSON: %v", string(payload))
		if log.IsDebugLevel() {
			cmdId := int(data["cmdId"].(float64))
			log.Log.Debugf("-> CmdId   %03d", cmdId)
			log.Log.Debugf("-> CmdFunc %f", data["cmdFunc"].(float64))
			log.Log.Debugf("-> Version %s", data["version"].(string))
			log.Log.Debugf("ID           : %f", data["id"].(float64))
		}
		if _, ok := data["params"]; ok {
			data = data["params"].(map[string]interface{})
		}
		if _, ok := data["serial_number"]; !ok {
			data["serial_number"] = serialNumber
		}
		if _, ok := data["timestamp"]; !ok {
			data["timestamp"] = time.Now()
		}
		tn := fmt.Sprintf("%s_mqtt", serialNumber)
		if !checkTable(mqttid, tn, func() []*common.Column {
			keys := make([]string, 0, len(data))
			for k := range data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			columns := make([]*common.Column, 0)
			// prefix := ""
			for _, k := range keys {
				v := data[k]
				name := "eco_" + strings.ReplaceAll(k, ".", "_")
				log.Log.Debugf("Add column %s=%v %T -> %s\n", k, v, v, name)
				column := createValueColumn(name, v)
				columns = append(columns, column)
			}
			return columns
		}) {
			checkTableColumns(mqttid, tn, data)
		}
		err = insertTable(mqttid, tn, data, insertMqttData)
		if err != nil && strings.Contains(err.Error(), "conn closed") {
			// Connection is closed reconnect
			mqttid.Close()
			mqttid = connnectDatabase()
		}
		return
	}

	start := 0
	end := len(payload)
	index := bytes.Index(payload, []byte(serialNumber))
	if index != -1 {
		end = index + len(serialNumber)
	}
	log.Log.Debugf("Serial index 1: %d/%d %d:%d", index, len(payload), start, end)
	displayPayload(serialNumber, payload[start:end])
	start = end
	if len(payload) > index+len(serialNumber) {
		index = bytes.Index(payload[end:], []byte(serialNumber))
		if index != -1 {
			end = end + index + len(serialNumber)
		} else {
			end = len(payload)
		}
		log.Log.Debugf("Serial index 2: %d", index)
		displayPayload(serialNumber, payload[start:end])
	}

}

func insertMqttData(data map[string]interface{}) ([]string, [][]any) {
	keys := make([]string, 0)
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	columns := make([]any, 0)
	// prefix := ""
	fields := make([]string, 0)
	for _, k := range keys {
		v := data[k]
		name := "eco_" + strings.ReplaceAll(k, ".", "_")
		fields = append(fields, name)
		log.Log.Debugf(" %s=%v %T -> %s\n", k, v, v, name)
		switch val := v.(type) {
		case string:
			columns = append(columns, val)
		case float64:
			if val == math.Trunc(val) {
				columns = append(columns, int64(val))
			} else {
				columns = append(columns, val)
			}
		case time.Time:
			columns = append(columns, val)
		case []interface{}, map[string]interface{}:
			b, err := json.Marshal(val)
			if err != nil {
				services.ServerMessage("Error marshal: %#v", val)
				columns = append(columns, nil)
			} else {
				s := string(b)
				columns = append(columns, s)
			}
		default:
			services.ServerMessage("Unknown type %s=%T\n", k, v)
			log.Log.Fatalf("Unknown type %s=%T\n", k, v)
		}
	}
	return fields, [][]any{columns}
}

func displayHeader(msg *Header) {
	if !log.IsDebugLevel() {
		return
	}
	log.Log.Debugf("-> Header  %s\n", msg)
	log.Log.Debugf("-> SM      %s\n", msg.GetDeviceSn())
	log.Log.Debugf("-> Version %d\n", msg.GetVersion())
	log.Log.Debugf("-> PayloadVersion %d\n", msg.GetPayloadVer())
	log.Log.Debugf("-> SRC     %d\n", msg.GetSrc())
	log.Log.Debugf("-> Dest    %d\n", msg.GetDest())
	log.Log.Debugf("-> Datalen %d\n", msg.GetDataLen())
	log.Log.Debugf("-> CmdId   %d\n", msg.GetCmdId())
	log.Log.Debugf("-> CmdFunc %d\n", msg.GetCmdFunc())
	log.Log.Debugf("-> DSRC    %d\n", msg.GetDSrc())
	log.Log.Debugf("-> DDest   %d\n", msg.GetDDest())
	log.Log.Debugf("-> NeedAcl %d\n", msg.GetNeedAck())
}

// OnConnect on connect open handler called if connetion is done
func OnConnect(client mqtt.Client) {
	for _, d := range devices.Devices {
		services.ServerMessage("Subscribe for MQTT entries of device %s", d.SN)
		err := ecoclient.SubscribeForParameters(d.SN, MessageHandler)
		if err != nil {
			log.Log.Errorf("Unable to subscribe for parameters %s: %v", d.SN, err)
		} else {
			log.Log.Infof("Subscribed to receive parameters %s", d.SN)
		}
	}
}

func getType(myvar interface{}) string {
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

// OnConnectionLost on connection lost happened
func OnConnectionLost(_ mqtt.Client, err error) {
	log.Log.Errorf("Error connection lost: %v", err)
}

// OnReconnect on connection reconnection
func OnReconnect(mqtt.Client, *mqtt.ClientOptions) {
	log.Log.Infof("Reconnecting...")
}

func getSnFromTopic(topic string) string {
	topicStr := strings.Split(topic, "/")
	return topicStr[len(topicStr)-1]
}
