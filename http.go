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
)

const layout = "2006-01-02 15:04:05.000"

var LoopSeconds = 60
var httpDone = make(chan bool, 1)

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

func triggerParameterStore(client *ecoflow.Client) {
	go httpParameterStore(client)
}

func httpParameterStore(client *ecoflow.Client) {
	id := connnectDatabase()

	for _, l := range devices.Devices {
		if l.Online == 1 {
			// GetDeviceAllParameters(client, l.SN)
			// log.Log.Debugf("Parameters: %v", parameters)
			// get param1 and param2 for device
			// resp, err := client.GetDeviceParameters(context.Background(), l.SN, []string{"param1", "param2"})
			// if err != nil {
			// 	log.Log.Fatalf("Error getting device list: %v", err)
			// }
			// get all parameters for device
			fmt.Printf("Get Parameter for : %s\n", l.SN)
			resp, err := client.GetDeviceAllParameters(context.Background(), l.SN)
			if err != nil {
				log.Log.Fatalf("Error getting device list: %v", err)
			}

			checkTable(id, "device_"+l.SN+"_quota", func() []*common.Column {
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
	}

	counter := uint64(0)
	for {
		counter++
		select {
		case <-httpDone:
			return
		case <-time.After(time.Second * time.Duration(LoopSeconds)):

			for _, l := range devices.Devices {
				if l.Online == 1 {
					tn := "device_" + l.SN + "_quota"
					stat := getStatEntry(l.SN)
					resp, err := client.GetDeviceAllParameters(context.Background(), l.SN)
					if err != nil {
						log.Log.Fatalf("Error getting device list: %v", err)
					}
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
				}
			}
		}
		log.Log.Infof("Triggered %d. HTTP query at %s", counter, time.Now().Format(layout))
	}
}

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
			fmt.Printf("Error marshal: %#v", val)
			return nil
		}
		s := string(b)
		l := uint16(1024)
		if len(s) > 1024 {
			l += uint16(1024) + uint16(len(s))
		}
		return &common.Column{Name: name, DataType: common.Alpha, Length: l}
	default:
		fmt.Printf("Unknown type %s=%T\n", name, v)
	}
	log.Log.Fatalf("Unknown type %s=%T\n", name, v)
	return nil
}

func checkTableColumns(id common.RegDbID, tn string, data map[string]interface{}) {
	col, err := id.GetTableColumn(tn)
	if err != nil {
		fmt.Println("Get table column", err)
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
				fmt.Printf("Error marshal: %#v", val)
			} else {
				s := string(b)
				columns = append(columns, s)
			}
		default:
			fmt.Printf("Unknown type %s=%T\n", k, v)
			log.Log.Fatalf("Unknown type %s=%T\n", k, v)
		}
	}
	return fields, [][]any{columns}
}

func endHttp() {
	httpDone <- true
}
