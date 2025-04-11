/*
* Copyright 2023 Thorsten A. Knieling
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
	"flag"

	"github.com/tknie/ecoflow2db"
)

const layout = "2006-01-02T15:04:05"

const defaultMaxTries = 10

func init() {
	ecoflow2db.StartLog("ecoflow2db.log")
}

func main() {
	create := false
	maxTries := defaultMaxTries

	flag.IntVar(&maxTries, "maxtries", defaultMaxTries, "The QoS to subscribe to messages at")
	flag.BoolVar(&create, "create", false, "Create new database")
	flag.Parse()

	ecoflow2db.InitDatabase()
	// creating new client with options. Current supports two options:
	// 1. custom ecoflow base url (can be used with proxies, or if they change the url)
	// 2. custom http client

	// client = ecoflow.NewEcoflowClient(accessKey, secretKey,
	//	ecoflow.WithBaseUrl("https://ecoflow-api.example.com"),
	//	ecoflow.WithHttpClient(customHttpClient()),
	//)
	ecoflow2db.InitEcoflow()

}
