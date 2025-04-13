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
	"flag"
	"fmt"

	"github.com/tknie/ecoflow2db"
)

func init() {
	ecoflow2db.StartLog("ecoflow2db.log")
}

func main() {
	create := false

	flag.IntVar(&ecoflow2db.LoopSeconds, "t", 1, "The seconds between REST API queries")
	flag.BoolVar(&create, "create", false, "Create new database")
	flag.Parse()

	fmt.Printf("Start ecoflow2db application v%s (build at %v)\n", ecoflow2db.Version, ecoflow2db.BuildDate)
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
