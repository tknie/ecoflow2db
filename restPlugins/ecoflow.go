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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/tknie/clu/api"
	"github.com/tknie/ecoflow"
	"github.com/tknie/log"
)

func prepareEcoflow() *ecoflow.Client {

	accessToken := os.ExpandEnv("${ECOFLOW_ACCESS_KEY}")
	secretToken := os.ExpandEnv("${ECOFLOW_SECRET_KEY}")

	log.Log.Debugf("AccessKey: %v", accessToken)
	log.Log.Debugf("SecretKey: %v", secretToken)
	client := ecoflow.NewClient(accessToken, secretToken)
	client.RefreshDeviceList()
	return client
}

func getDeviceImportant(req *http.Request, specific string) (r api.CallExtendRes, _ error) {
	val := req.URL.Query()
	sn := ""
	if len(val) > 0 {
		sn = val["sn"][0]
	}
	ctx := context.Background()
	if sn == "" {
		return nil, fmt.Errorf("No serial number given")
	}
	client := prepareEcoflow()
	dsn, err := client.GetDeviceInfo(ctx, sn, specific)
	if err != nil {
		return nil, err
	}
	removeList := make([]string, 0)
	for k := range dsn {
		if strings.Contains(k, "task") {
			removeList = append(removeList, k)
		}
	}
	for _, k := range removeList {
		delete(dsn, k)
	}
	jsonMap := make(map[string]interface{})
	d := make(api.ResponseRaw)
	v := struct {
		Serial string
		Info   map[string]interface{}
	}{Serial: sn, Info: dsn}
	jsonMap["Device"] = v
	data, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, err
	}
	d.UnmarshalJSON(data)
	return &d, nil
}

func getDeviceInfo(req *http.Request) (r api.CallExtendRes, _ error) {
	val := req.URL.Query()
	sn := ""
	if len(val) > 0 {
		sn = val["sn"][0]
	}
	ctx := context.Background()
	list := make([]interface{}, 0)
	jsonMap := make(map[string]interface{})
	d := make(api.ResponseRaw)
	if sn != "" {
		client := prepareEcoflow()
		dsn, err := client.GetDeviceAllParameters(ctx, sn)
		if err != nil {
			return nil, err
		}
		v := struct {
			Serial string
			Info   map[string]interface{}
		}{Serial: sn, Info: dsn}
		list = append(list, v)
	} else {
		client := prepareEcoflow()
		devList, err := client.GetDeviceList(ctx)
		if err != nil {
			return nil, err
		}
		for index, device := range devList.Devices {
			fmt.Println(index, device.SN)
			d, err := client.GetDeviceAllParameters(ctx, device.SN)
			if err != nil {
				return nil, err
			}
			v := struct {
				Serial string
				Info   map[string]interface{}
			}{Serial: device.SN, Info: d}
			list = append(list, v)
		}
	}
	jsonMap["Devices"] = list
	data, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, err
	}
	d.UnmarshalJSON(data)
	return &d, nil
}
