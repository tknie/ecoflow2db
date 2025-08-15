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

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	ht "github.com/ogen-go/ogen/http"

	"github.com/go-faster/jx"
	"github.com/tknie/clu/api"
	"github.com/tknie/clu/plugins"
	"github.com/tknie/ecoflow"
	"github.com/tknie/log"
)

type greeting string

type cmdStatus struct {
	status string
	cmd    *exec.Cmd
	err    error
}

const (
	checkMediaNr byte = iota
)

func init() {
}

// Types type of plugin working with
func (g greeting) Types() []plugins.PluginTypes {
	return []plugins.PluginTypes{plugins.ExtendPlugin}
}

// Name name of the plugin
func (g greeting) Name() string {
	return "Ecoflow"
}

// Version version of the number
func (g greeting) Version() string {
	return "1.0"
}

// Stop stop plugin
func (g greeting) Stop() {
}

func (g greeting) EntryPoint() []string {
	return []string{"ecoflow"}
}

func parseQuery(req *http.Request) (service string, values url.Values) {
	service = req.URL.RawFragment
	val := req.URL.Query()
	return service, val
}

func (g greeting) CallExtendGet(path string, req *http.Request) (r api.CallExtendRes, _ error) {
	callPath := strings.ReplaceAll(path, g.EntryPoint()[0]+"/", "")
	fmt.Println("Ecoflow plugin GET call received:" + path)
	fmt.Println("Callpath plugin              : <" + callPath + ">")
	service, valMap := parseQuery(req)

	d := make(api.ResponseRaw)
	switch strings.ToLower(callPath) {
	case "health":
		for k, v := range valMap {
			fmt.Println(k, ":", v)
		}
		status := "ok"
		d["Health"] = jx.Raw([]byte("\"" + status + "\""))
	default:
		fmt.Println("Unknown service: " + callPath + " -> " + service + " call status")
		generateStatus(d)
	}

	return &d, nil

}

func prepareEcoflow() *ecoflow.Client {

	accessKey := os.ExpandEnv("ECOFLOW_ACCESSKEY")
	secretKey := os.ExpandEnv("ECOFLOW_SECRETKEY")

	log.Log.Debugf("AccessKey: %v", accessKey)
	log.Log.Debugf("SecretKey: %v", secretKey)
	client := ecoflow.NewClient(accessKey, secretKey)
	client.RefreshDeviceList()
	return client
}

func (g greeting) CallExtendPut(path string, req *http.Request) (r api.TriggerExtendRes, _ error) {
	callPath := strings.ReplaceAll(path, g.EntryPoint()[0]+"/", "")
	fmt.Println("Extend plugin PUT call received:" + path)
	fmt.Println("Callpath plugin              : <" + callPath + ">")
	service, valMap := parseQuery(req)
	for k, v := range valMap {
		fmt.Println(k, ":", v)
	}

	d := make(api.ResponseRaw)
	switch strings.ToLower(callPath) {
	case "power":
		fmt.Println("Update power ...", valMap["energy"][0])
		sn := valMap["serialNumber"][0]
		power, err := strconv.ParseFloat(valMap["energy"][0], 64)
		if err != nil {
			return nil, err
		}
		config := prepareEcoflow()
		config.SetEnvironmentPowerConsumption(sn, power)
	default:
		fmt.Println("Unknown service: " + callPath + " -> " + service + " call status")
		generateStatus(d)
	}

	return &d, nil
}

func (g greeting) CallExtendPost(path string, req *http.Request) (r api.CallPostExtendRes, _ error) {
	callPath := strings.ReplaceAll(path, g.EntryPoint()[0]+"/", "")
	fmt.Println("Extend plugin PUT call received:" + path)
	fmt.Println("Callpath plugin              : <" + callPath + ">")
	_, valMap := parseQuery(req)
	for k, v := range valMap {
		fmt.Println(k, ":", v)
	}

	return r, ht.ErrNotImplemented
}

func generateStatus(d api.ResponseRaw) {
	t := "Ecoflow"
	s := "Tool"
	raw := jx.Raw([]byte("\"" + t + "\""))
	d[s] = raw
	d["Status"] = jx.Raw([]byte("\"OK\""))
}

// exported

// Loader loader for initialize plugin
var Loader greeting

// EntryPoint entry point for main structure
var EntryPoint greeting
