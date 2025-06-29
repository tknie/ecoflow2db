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
	"os"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/tknie/flynn"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var dbTables []string
var dbRef *common.Reference
var dbPassword string

type storeElement struct {
	sn     string
	object any
}

var msgChan = make(chan *storeElement, 100)

// init initialized tablename information
func init() {
	tableName = os.Getenv("ECOFLOW_DB_TABLENAME")
}

// InitDatabase init database connections
func InitDatabase() {
	databaseUrl := os.Getenv("ECOFLOW_DB_URL")
	var err error
	dbRef, dbPassword, err = common.NewReference(databaseUrl)
	if err != nil {
		services.ServerMessage("Shuting down ... URL is incorrect or cannot be parsed: %v", err)
		log.Log.Fatalf("REST audit URL incorrect: " + databaseUrl)
	}
	if dbRef.User == "" {
		dbRef.User = os.Getenv("ECOFLOW_DB_USER")
	}
	if dbPassword == "" {
		dbPassword = os.Getenv("ECOFLOW_DB_PASS")
	}
	log.Log.Debugf("DB password: %s", dbPassword)
	_, err = flynn.Handler(dbRef, dbPassword)
	if err != nil {
		services.ServerMessage("Shuting down ... register database error: %v", err)
		log.Log.Fatalf("Register error log: %v", err)
	}
	readDatabaseMaps()
	go storeDatabase()
}

// readDatabaseMaps read database tables to check for
func readDatabaseMaps() {
	dbTables = flynn.Maps()
	log.Log.Debugf("Tables: %#v", dbTables)
}

// connnectDatabase connect connection to database for the corresponding storage
func connnectDatabase() common.RegDbID {
	log.Log.Debugf("Connected to database %s", dbRef)
	id, err := flynn.Handler(dbRef, dbPassword)
	if err != nil {
		services.ServerMessage("Shuting down, connect database error: %v", err)
		log.Log.Fatalf("Connect database error: %v", err)
	}
	return id
}

// storeDatabase final insert into database with device information
func storeDatabase() {
	storeid := connnectDatabase()
	for m := range msgChan {
		tn := m.checkTable(storeid)

		log.Log.Debugf("Insert structFields: %T into tn", m.object)
		fields := []string{"*"}
		insert := &common.Entries{DataStruct: m.object,
			Fields: fields}
		insert.Values = [][]any{{m.object}}
		_, err := storeid.Insert(tn, insert)
		if err != nil {
			services.ServerMessage("Error inserting record: %v", err)
			// Reconnecting ...
			storeid.Close()
			storeid = connnectDatabase()
			// log.Log.Fatal("Error inserting record: ", err)
		} else {
			getDbStatEntry(tn).counter++
		}

	}
}

// checkTable check if table is available and if not, create it
func (m *storeElement) checkTable(storeid common.RegDbID) string {
	tn := strings.ToLower(tableName + "_" + m.sn + "_" + getType(m.object))
	if !slices.Contains(dbTables, tn) {
		services.ServerMessage("Database %s needed to be created", tn)
		err := storeid.CreateTable(tn, m.object)
		if err != nil {
			log.Log.Fatal("Error creating database table: ", err)
		}
		readDatabaseMaps()
	}
	return tn
}

// checkTable check table and if not available, create table
func checkTable(storeid common.RegDbID, tn string, generateColumns func() []*common.Column) bool {
	if !slices.Contains(dbTables, strings.ToLower(tn)) {
		services.ServerMessage("Database %s needed to be created", tn)
		err := storeid.CreateTable(tn, generateColumns())
		if err != nil {
			services.ServerMessage("Shuting down ... error creating database for %s : %v", tn, err)
			log.Log.Fatal("Error creating database table: ", err)
		}
		readDatabaseMaps()
		return true
	}
	return false
}

// insertTable insert data into database
func insertTable(storeid common.RegDbID, tn string, data map[string]interface{}, generateColumns func(map[string]interface{}) ([]string, [][]any)) error {
	fields, values := generateColumns(data)
	log.Log.Debugf("Insert columnFields: %#v", fields)
	if len(fields) == 0 {
		debug.PrintStack()
	}
	insert := &common.Entries{
		Values: values,
		Fields: fields}
	_, err := storeid.Insert(tn, insert)
	if err != nil {
		services.ServerMessage("Error inserting record: %v\n", err)
		// log.Log.Fatal("Error inserting record: ", err)
	} else {
		getDbStatEntry(tn).counter++
	}
	return err
}

// insertTable insert data into database
func readBatch(readid common.RegDbID, tn string, selectCmd string, f func(search *common.Query, result *common.Result) error) error {
	query := common.Query{Search: selectCmd}
	err := readid.BatchSelectFct(&query, f)
	return err
}
