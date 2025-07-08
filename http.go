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
	"math"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/tess1o/go-ecoflow"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

const layout = "2006-01-02 15:04:05.000"
const shortLayout = "2006-01-02 15:04"

const DefaultSeconds = 60

var LoopSeconds = DefaultSeconds
var httpDone = make(chan bool, 1)

// GetDeviceAllParameters get all device parameters for a specific device
// Use HTTP request to get the parameter information
func GetDeviceAllParameters(client *ecoflow.Client, deviceSn string) error {
	requestParams := make(map[string]interface{})
	requestParams["sn"] = deviceSn
	accessKey := os.Getenv("ECOFLOW_ACCESS_KEY")
	secretKey := os.Getenv("ECOFLOW_SECRET_KEY")

	request := ecoflow.NewHttpRequest(&http.Client{}, "GET", "https://api.ecoflow.com/iot-open/sign/device/quota/all", requestParams, accessKey, secretKey)
	response, err := request.Execute(context.Background())

	if err != nil {
		return err
	}
	log.Log.Debugf("Response: %s", string(response))
	return nil
}

// httpParameterStore main thread reading information with HTTP request
// and store them in the database
func httpParameterStore(client *ecoflow.Client) {
	id := connnectDatabase()

	for _, l := range devices.Devices {
		// get all parameters for device
		services.ServerMessage("Get Parameter for : %s", l.SN)
		resp, err := client.GetDeviceAllParameters(context.Background(), l.SN)
		if err != nil {
			services.ServerMessage("Error getting device parameter sn=%s: %v", l.SN, err)
			log.Log.Errorf("Error getting device parameter sn=%s: %v", l.SN, err)
			continue
		}

		// Check, create and write into table
		checkTable(id, "device_quota", func() []*common.Column {
			keys := make([]string, 0, len(resp))
			for k := range resp {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			columns := make([]*common.Column, 0)
			// prefix := ""
			for _, k := range keys {
				v := resp[k]
				// prefix = strings.Split(k, ".")[0]
				// name := "eco_" + strings.ReplaceAll(k[len(prefix)+1:], ".", "_")
				name := "eco_" + strings.ReplaceAll(k, ".", "_")
				log.Log.Debugf("Add column %s=%v %T -> %s\n", k, v, v, name)
				column := createValueColumn(name, v)
				columns = append(columns, column)
			}
			return columns
		})
	}

	// Loop reading and writing data into table
	counter := uint64(0)
	needRefresh := false
	for {
		counter++
		select {
		case <-httpDone:
			services.ServerMessage("Ecoflow API loop is stopped")

			return
		case <-time.After(time.Second * time.Duration(LoopSeconds)):
			if counter%350 == 0 {
				services.ServerMessage("Received HTTP requests: %04d", counter)
			}

			for _, l := range devices.Devices {
				tn := "device_quota"
				stat := getStatEntry(l.SN)
				resp, err := client.GetDeviceAllParameters(context.Background(), l.SN)
				if err != nil {
					log.Log.Errorf("Error getting device list %s: %v", l.SN, err)
					services.ServerMessage("Error getting device list %s: %v", l.SN, err)
				} else {
					if _, ok := resp["serial_number"]; !ok {
						resp["serial_number"] = l.SN
					}
					if _, ok := resp["timestamp"]; !ok {
						resp["timestamp"] = time.Now()
					}
					checkTableColumns(id, tn, resp)
					err = insertTable(id, tn, resp, insertHttpData)
					if err != nil && strings.Contains(err.Error(), "conn closed") {
						id.Close()
						id = connnectDatabase()
					}
					stat.httpCounter++
					if l.Online != 1 {
						services.ServerMessage(l.SN + " device is offline")
						needRefresh = true
					}
				}
			}
		}
		log.Log.Infof("Triggered %d. HTTP query at %s", counter, time.Now().Format(layout))
		if needRefresh {
			refreshDeviceList(client)
		}
	}
}

// createValueColumn create value columns dependent of the information
// received by HTTP request
func createValueColumn(name string, v interface{}) *common.Column {
	if strings.ToLower(name) == "timestamp" {
		return &common.Column{Name: name, DataType: common.CurrentTimestamp, Length: 8}
	}
	switch val := v.(type) {
	case string:
		return &common.Column{Name: name, DataType: common.Alpha, Length: 255}
	case time.Time:
		return &common.Column{Name: name, DataType: common.CurrentTimestamp, Length: 8}
	case float64:
		if val == math.Trunc(val) && val < math.MaxInt64 {
			return &common.Column{Name: name, DataType: common.BigInteger, Length: 0}
		} else {
			return &common.Column{Name: name, DataType: common.Decimal, Length: 8}
		}
	case []interface{}, map[string]interface{}:
		b, err := json.Marshal(val)
		if err != nil {
			services.ServerMessage("Error marshal: %#v", val)
			return nil
		}
		s := string(b)
		l := uint16(1024)
		if len(s) > 1024 {
			l += uint16(1024) + uint16(len(s))
		}
		return &common.Column{Name: name, DataType: common.Alpha, Length: l}
	default:
		services.ServerMessage("Unknown type %s=%T", name, v)
	}
	log.Log.Errorf("Unknown type %s=%T\n", name, v)
	return nil
}

// checkTableColumns check if new parameters are in current request to adapt table
func checkTableColumns(id common.RegDbID, tn string, data map[string]interface{}) {
	col, err := id.GetTableColumn(tn)
	if err != nil {
		services.ServerMessage("Get table column %v", err)
		return
	}
	log.Log.Debugf("Validate to defined columns %#v", col)
	columns := make([]*common.Column, 0)
	for k, v := range data {
		name := "eco_" + strings.ReplaceAll(strings.ToLower(k), ".", "_")
		if !slices.Contains(col, name) {
			log.Log.Debugf("Column not in table %s", name)
			c := createValueColumn(name, v)
			columns = append(columns, c)
		}
	}
	if len(columns) > 0 {
		log.Log.Debugf("Add %d. columns to table %T", len(columns), columns)
		err = id.AdaptTable(tn, columns)
		log.Log.Debugf("Added %d. columns to table: %v", len(columns), err)
	}
}

// insertHttpData prepare database data to be inserted into the database
func insertHttpData(data map[string]interface{}) ([]string, [][]any) {
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
		// prefix = strings.Split(k, ".")[0]
		// name := "eco_" + strings.ReplaceAll(k[len(prefix)+1:], ".", "_")
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
			} else {
				s := string(b)
				columns = append(columns, s)
			}
		default:
			services.ServerMessage("Unknown HTTP JSON type %s=%T", k, v)
			log.Log.Errorf("Unknown type %s=%T\n", k, v)
		}
	}
	return fields, [][]any{columns}
}

// endHttp end Database store of HTTP data
func endHttp() {
	httpDone <- true
}
