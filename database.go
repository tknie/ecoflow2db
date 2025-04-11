package ecoflow2db

import (
	"fmt"
	"os"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/tknie/flynn"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
)

var dbTables []string
var dbRef *common.Reference
var dbPassword string

type storeElement struct {
	sn     string
	object any
}

var msgChan = make(chan *storeElement, 100)

func init() {
	tableName = os.Getenv("ECOFLOW_DB_TABLENAME")
}

func InitDatabase() {
	databaseUrl := os.Getenv("ECOFLOW_DB_URL")
	var err error
	dbRef, dbPassword, err = common.NewReference(databaseUrl)
	if err != nil {
		log.Log.Fatalf("REST audit URL incorrect: " + databaseUrl)
	}
	if dbPassword == "" {
		dbPassword = os.Getenv("ECOFLOW_DB_PASS")
	}
	_, err = flynn.Handler(dbRef, dbPassword)
	if err != nil {
		log.Log.Fatalf("Register error log: %v", err)
	}
	dbTables = flynn.Maps()
	log.Log.Infof("Tables: %#v", dbTables)
	go storeDatabase()
}

func openDatabase() common.RegDbID {
	log.Log.Debugf("Connected to database %s", dbRef)
	id, err := flynn.Handler(dbRef, dbPassword)
	if err != nil {
		log.Log.Fatalf("Register error log: %v", err)
	}
	return id
}

func storeDatabase() {
	storeid := openDatabase()
	for m := range msgChan {
		tn := m.checkTable(storeid)

		log.Log.Debugf("Insert structFields: %T", m.object)
		fields := []string{"*"}
		insert := &common.Entries{DataStruct: m.object,
			Fields: fields}
		insert.Values = [][]any{{m.object}}
		_, err := storeid.Insert(tn, insert)
		if err != nil {
			fmt.Println("Error inserting record:", err)
			// log.Log.Fatal("Error inserting record: ", err)
		}

	}
}

func (m *storeElement) checkTable(storeid common.RegDbID) string {
	tn := strings.ToLower(tableName + "_" + m.sn + "_" + getType(m.object))
	if !slices.Contains(dbTables, tn) {
		fmt.Printf("Database %s needed to be created", tn)
		err := storeid.CreateTable(tn, m.object)
		if err != nil {
			log.Log.Fatal("Error creating database table: ", err)
		}
	}
	return tn
}

func checkTable(storeid common.RegDbID, tn string, generateColumns func() []*common.Column) string {
	if !slices.Contains(dbTables, strings.ToLower(tn)) {
		fmt.Printf("Database %s needed to be created\n", tn)
		err := storeid.CreateTable(tn, generateColumns())
		if err != nil {
			fmt.Printf("Error creating database for %s : %v\n", tn, err)
			log.Log.Fatal("Error creating database table: ", err)
		}
	}
	return tn
}

func insertTable(storeid common.RegDbID, tn string, generateColumns func() ([]string, [][]any)) {

	fields, values := generateColumns()
	log.Log.Debugf("Insert columnFields: %#v", fields)
	if len(fields) == 0 {
		debug.PrintStack()
	}
	insert := &common.Entries{
		Values: values,
		Fields: fields}
	_, err := storeid.Insert(tn, insert)
	if err != nil {
		fmt.Printf("Error inserting record: %v\n", err)
		// log.Log.Fatal("Error inserting record: ", err)
	}

}
