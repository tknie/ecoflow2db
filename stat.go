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

import sync "sync"

type statMqtt struct {
	mu          sync.Mutex
	mqttCounter uint64
	httpCounter uint64
}

var mapStatMqtt = make(map[string]*statMqtt)

type statDatabase struct {
	counter uint64
}

var mapStatDatabase = make(map[string]*statDatabase)

func getStatEntry(serialNumber string) *statMqtt {
	if s, ok := mapStatMqtt[serialNumber]; ok {
		return s
	} else {
		stat := &statMqtt{}
		mapStatMqtt[serialNumber] = stat
		return stat
	}

}

func getDbStatEntry(tn string) *statDatabase {
	if s, ok := mapStatDatabase[tn]; ok {
		return s
	} else {
		stat := &statDatabase{}
		mapStatDatabase[tn] = stat
		return stat
	}
}
