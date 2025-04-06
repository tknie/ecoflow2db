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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tess1o/go-ecoflow"
	"github.com/tknie/ecoflow2db"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"google.golang.org/protobuf/proto"
)

const layout = "2006-01-02T15:04:05"

const defaultMaxTries = 10

var tableName string
var ecoclient *ecoflow.MqttClient
var devices *ecoflow.DeviceListResponse
var storeid common.RegDbID

var (
	mu sync.Mutex
)

func init() {
	tableName = os.Getenv("ECOFLOW_DB_TABLENAME")
	ecoflow2db.StartLog("ecoflow2db.log")
}

func main() {
	create := false
	maxTries := defaultMaxTries

	flag.IntVar(&maxTries, "maxtries", defaultMaxTries, "The QoS to subscribe to messages at")
	flag.BoolVar(&create, "create", false, "Create new database")
	flag.Parse()

	user := os.Getenv("ECOFLOW_USER")
	password := os.Getenv("ECOFLOW_PASSWORD")

	accessKey := os.Getenv("ECOFLOW_ACCESS_KEY")
	secretKey := os.Getenv("ECOFLOW_SECRET_KEY")

	fmt.Printf("AccessKey: %v\n", accessKey)
	fmt.Printf("SecretKey: %v\n", secretKey)
	client := ecoflow.NewEcoflowClient(accessKey, secretKey)

	// url := os.Getenv("ECOFLOW_DB_URL")
	// dbRef, password, err := common.NewReference(url)
	// if err != nil {
	// 	log.Log.Fatalf("REST audit URL incorrect: " + url)
	// }
	// id, err := flynn.Handler(dbRef, password)
	// if err != nil {
	// 	log.Log.Fatalf("Register error log: %v", err)
	// }
	// storeid = id
	// fmt.Println("Connected to database", dbRef)

	// creating new client with options. Current supports two options:
	// 1. custom ecoflow base url (can be used with proxies, or if they change the url)
	// 2. custom http client

	// client = ecoflow.NewEcoflowClient(accessKey, secretKey,
	//	ecoflow.WithBaseUrl("https://ecoflow-api.example.com"),
	//	ecoflow.WithHttpClient(customHttpClient()),
	//)

	//get all linked ecoflow devices. Returns SN and online status
	list, err := client.GetDeviceList(context.Background())
	if err != nil {
		log.Log.Fatalf("Error getting device list: %v", err)
	}
	devices = list
	fmt.Printf("Devices:\n")
	for _, l := range list.Devices {
		fmt.Println("Serial number:", l.SN)
		if l.Online == 1 {
			parameters := []string{
				"bms_bmsInfo.accuChgCap",
				"bms_bmsInfo.accuChgEnergy",
				"bms_bmsInfo.accuDsgCap",
				"bms_bmsInfo.accuDsgEnergy",
				"bms_bmsInfo.bmsLanchDate",
				"bms_bmsInfo.bsmCycles",
				"bms_bmsInfo.deepDsgCnt",
				"bms_bmsInfo.highTempChgTime",
				"bms_bmsInfo.highTempTime",
				"bms_bmsInfo.lauchDateFlag",
				"bms_bmsInfo.lowTempChgTime",
				"bms_bmsInfo.lowTempTime",
				"bms_bmsInfo.num",
				"bms_bmsInfo.ohmRes",
				"bms_bmsInfo.powerCapability",
				"bms_bmsInfo.resetFlag",
				"bms_bmsInfo.roundTrip",
				"bms_bmsInfo.selfDsgRate",
				"bms_bmsInfo.sn",
				"bms_bmsInfo.soh",
				"bms_bmsStatus.actSoc",
				"bms_bmsStatus.allBmsFault",
				"bms_bmsStatus.allErrCode",
				"bms_bmsStatus.amp",
				"bms_bmsStatus.balanceState",
				"bms_bmsStatus.bmsFault",
				"bms_bmsStatus.bmsHeartVer",
				"bms_bmsStatus.bqSysStatReg",
				"bms_bmsStatus.caleSoh",
				"bms_bmsStatus.cellId",
				"bms_bmsStatus.cellTemp",
				"bms_bmsStatus.cellVol",
				"bms_bmsStatus.chgCap",
				"bms_bmsStatus.chgState",
				"bms_bmsStatus.cycSoh",
				"bms_bmsStatus.cycles",
				"bms_bmsStatus.designCap",
				"bms_bmsStatus.diffSoc",
				"bms_bmsStatus.dsgCap",
				"bms_bmsStatus.ecloudOcv",
				"bms_bmsStatus.errCode",
				"bms_bmsStatus.f32ShowSoc",
				"bms_bmsStatus.fullCap",
				"bms_bmsStatus.hwVersion",
				"bms_bmsStatus.inputWatts",
				"bms_bmsStatus.loaderVer",
				"bms_bmsStatus.maxCellTemp",
				"bms_bmsStatus.maxCellVol",
				"bms_bmsStatus.maxMosTemp",
				"bms_bmsStatus.maxVolDiff",
				"bms_bmsStatus.minCellTemp",
				"bms_bmsStatus.minCellVol",
				"bms_bmsStatus.minMosTemp",
				"bms_bmsStatus.mosState",
				"bms_bmsStatus.num",
				"bms_bmsStatus.openBmsIdx",
				"bms_bmsStatus.outputWatts",
				"bms_bmsStatus.packSn",
				"bms_bmsStatus.productDetail",
				"bms_bmsStatus.productType",
				"bms_bmsStatus.realSoh",
				"bms_bmsStatus.remainCap",
				"bms_bmsStatus.remainTime",
				"bms_bmsStatus.soc",
				"bms_bmsStatus.soh",
				"bms_bmsStatus.sysState",
				"bms_bmsStatus.sysVer",
				"bms_bmsStatus.tagChgAmp",
				"bms_bmsStatus.targetSoc",
				"bms_bmsStatus.temp",
				"bms_bmsStatus.type",
				"bms_bmsStatus.vol",
				"bms_emsStatus.bmsIsConnt",
				"bms_emsStatus.bmsModel",
				"bms_emsStatus.bmsWarState",
				"bms_emsStatus.chgAmp",
				"bms_emsStatus.chgCmd",
				"bms_emsStatus.chgDisCond",
				"bms_emsStatus.chgLinePlug",
				"bms_emsStatus.chgRemainTime",
				"bms_emsStatus.chgState",
				"bms_emsStatus.chgVol",
				"bms_emsStatus.dsgCmd",
				"bms_emsStatus.dsgDisCond",
				"bms_emsStatus.dsgRemainTime",
				"bms_emsStatus.emsIsNormalFlag",
				"bms_emsStatus.emsVer",
				"bms_emsStatus.f32LcdShowSoc",
				"bms_emsStatus.fanLevel",
				"bms_emsStatus.lcdShowSoc",
				"bms_emsStatus.maxAvailNum",
				"bms_emsStatus.maxChargeSoc",
				"bms_emsStatus.maxCloseOilEb",
				"bms_emsStatus.minDsgSoc",
				"bms_emsStatus.minOpenOilEb",
				"bms_emsStatus.openBmsIdx",
				"bms_emsStatus.openUpsFlag",
				"bms_emsStatus.paraVolMax",
				"bms_emsStatus.paraVolMin",
				"bms_emsStatus.sysChgDsgState",
				"bms_kitInfo.aviDataLen",
				"bms_kitInfo.kitNum",
				"bms_kitInfo.version",
				"bms_kitInfo.watts",
				"inv.FastChgWatts",
				"inv.SlowChgWatts",
				"inv.acDipSwitch",
				"inv.acInAmp",
				"inv.acInFreq",
				"inv.acInVol",
				"inv.cfgAcEnabled",
				"inv.cfgAcOutFreq",
				"inv.cfgAcOutVol",
				"inv.cfgAcWorkMode",
				"inv.cfgAcXboost",
				"inv.chargerType",
				"inv.chgPauseFlag",
				"inv.dcInAmp",
				"inv.dcInTemp",
				"inv.dcInVol",
				"inv.dischargeType",
				"inv.errCode",
				"inv.fanState",
				"inv.inputWatts",
				"inv.invOutAmp",
				"inv.invOutFreq",
				"inv.invOutVol",
				"inv.invType",
				"inv.outTemp",
				"inv.outputWatts",
				"inv.reserved",
				"inv.standbyMins",
				"inv.sysVer",
				"mppt.acCloseLoc",
				"mppt.acStandbyMins",
				"mppt.beepState",
				"mppt.carOutAmp",
				"mppt.carOutVol",
				"mppt.carOutWatts",
				"mppt.carStandbyMin",
				"mppt.carState",
				"mppt.carTemp",
				"mppt.cfgAcEnabled",
				"mppt.cfgAcOutFreq",
				"mppt.cfgAcOutVol",
				"mppt.cfgAcXboost",
				"mppt.cfgChgType",
				"mppt.cfgChgWatts",
				"mppt.chgPauseFlag",
				"mppt.chgState",
				"mppt.chgType",
				"mppt.dc24vState",
				"mppt.dc24vTemp",
				"mppt.dcChgCurrent",
				"mppt.dcCloseLoc",
				"mppt.dcdc12vAmp",
				"mppt.dcdc12vVol",
				"mppt.dcdc12vWatts",
				"mppt.dischargeType",
				"mppt.faultCode",
				"mppt.inAmp",
				"mppt.inVol",
				"mppt.inWatts",
				"mppt.mpptTemp",
				"mppt.outAmp",
				"mppt.outVol",
				"mppt.outWatts",
				"mppt.powStandbyMin",
				"mppt.res",
				"mppt.scrStandbyMin",
				"mppt.swVer",
				"mppt.x60ChgType",
				"pd.acAutoOnCfg",
				"pd.acAutoOutConfig",
				"pd.acAutoOutPause",
				"pd.acEnabled",
				"pd.beepMode",
				"pd.bmsInfoFull",
				"pd.bmsInfoIncre",
				"pd.bmsRunIncre",
				"pd.bpPowerSoc",
				"pd.brightLevel",
				"pd.carState",
				"pd.carTemp",
				"pd.carUsedTime",
				"pd.carWatts",
				"pd.chargerType",
				"pd.chgDsgState",
				"pd.chgPowerAC",
				"pd.chgPowerDC",
				"pd.chgSunPower",
				"pd.dcInUsedTime",
				"pd.dcOutState",
				"pd.dsgPowerAC",
				"pd.dsgPowerDC",
				"pd.errCode",
				"pd.ext3p8Port",
				"pd.ext4p8Port",
				"pd.extRj45Port",
				"pd.hysteresisAdd",
				"pd.icoBytes",
				"pd.inWatts",
				"pd.inputWatts",
				"pd.invUsedTime",
				"pd.lcdOffSec",
				"pd.minAcoutSoc",
				"pd.model",
				"pd.mpptUsedTime",
				"pd.outWatts",
				"pd.outputWatts",
				"pd.pdInfoFull",
				"pd.pdInfoIncre",
				"pd.pdRunIncre",
				"pd.pvChgPrioSet",
				"pd.qcUsb1Watts",
				"pd.qcUsb2Watts",
				"pd.relaySwitchCnt",
				"pd.remainTime",
				"pd.reserved",
				"pd.soc",
				"pd.standbyMin",
				"pd.sysVer",
				"pd.typec1Temp",
				"pd.typec1Watts",
				"pd.typec2Temp",
				"pd.typec2Watts",
				"pd.typecUsedTime",
				"pd.usb1Watts",
				"pd.usb2Watts",
				"pd.usbUsedTime",
				"pd.usbqcUsedTime",
				"pd.watchIsConfig",
				"pd.wattsInSum",
				"pd.wattsOutSum",
				"pd.wifiAutoRcvy",
				"pd.wifiRssi",
				"pd.wifiVer",
				"pd.wireWatts",
			}
			log.Log.Debugf("Parameters: %v", parameters)
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
			keys := make([]string, 0, len(resp))
			for k := range resp {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf(" %s=%v\n", k, resp[k])
			}
			fmt.Println("Resp: ", resp)
		}
	}

	fmt.Println("Initialize MQTT client")
	configuration := ecoflow.MqttClientConfiguration{
		Email:            user,
		Password:         password,
		OnConnect:        OnConnect,
		OnConnectionLost: OnConnectionLost,
		OnReconnect:      OnReconnect,
	}
	ecoclient, err = ecoflow.NewMqttClient(context.Background(), configuration)
	if err != nil {
		log.Log.Fatalf("Error new MQTT client: %v", err)
	}
	log.Log.Debugf("Strt mqtt Ecoflow connect")
	done := make(chan bool, 1)
	setupGracefulShutdown(done)
	fmt.Println("Connecting MQTT client")
	ecoclient.Connect()
	log.Log.Debugf("Wait for Ecoflow disconnect")
	fmt.Println("Waiting for MQTT data")
	<-done
}

func setupGracefulShutdown(done chan bool) {
	// Create a channel to listen for OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to handle shutdown
	go func() {
		<-signalChan
		log.Log.Infof("Received shutdown signal")

		done <- true
	}()
}

func MessageHandler(_ mqtt.Client, msg mqtt.Message) {
	mu.Lock()
	defer mu.Unlock()

	serialNumber := getSnFromTopic(msg.Topic())

	fmt.Println("Received for ", serialNumber, "at", time.Now())

	log.Log.Debugf("received message on topic %s; body (retain: %t): %s\n", msg.Topic(),
		msg.Retained(), ecoflow2db.FormatByteBuffer("MQTT Body", msg.Payload()))
	payload := msg.Payload()

	data := make(map[string]interface{})
	err := json.Unmarshal(payload, &data)
	if err == nil {
		fmt.Println("Payload : ", string(payload))
		fmt.Println("Received: ", data)
		fmt.Printf("-> CmdId   %f\n", data["cmdId"].(float64))
		fmt.Printf("-> CmdFunc %f\n", data["cmdFunc"].(float64))
		fmt.Printf("-> Version %s\n", data["version"].(string))
		fmt.Printf("ID           : %f\n", data["id"].(float64))
		if v, ok := data["mppt.dcChgCurrent"]; ok {
			fmt.Println("DCChgCurrent :", v.(float64))
		}
		if v, ok := data["mppt.cfgChgWatts"]; ok {
			fmt.Printf("ChgWatts     : %f\n", v.(float64))
		}

		if v, ok := data["mppt.outVol"]; ok {
			fmt.Printf("OutVol       : %f\n", v.(float64))
		}
		fmt.Println("Timestamp    :", time.Unix(int64(data["timestamp"].(float64)), 0))
		return
	}

	start := 0
	end := len(payload)
	index := bytes.Index(payload, []byte(serialNumber))
	if index != -1 {
		end = index + len(serialNumber)
	}
	log.Log.Debugf("Serial index 1: %d/%d %d:%d", index, len(payload), start, end)
	displayPayload(payload[start:end])
	start = end
	if len(payload) > index+len(serialNumber) {
		index = bytes.Index(payload[end:], []byte(serialNumber))
		if index != -1 {
			end = end + index + len(serialNumber)
		} else {
			end = len(payload)
		}
		log.Log.Debugf("Serial index 2: %d", index)
		displayPayload(payload[start:end])
	}

}

func displayHeader(msg *ecoflow2db.Header) {
	fmt.Printf("-> Header  %s\n", msg)
	fmt.Printf("-> SM      %s\n", msg.GetDeviceSn())
	fmt.Printf("-> Version %d\n", msg.GetVersion())
	fmt.Printf("-> PayloadVersion %d\n", msg.GetPayloadVer())
	fmt.Printf("-> SRC     %d\n", msg.GetSrc())
	fmt.Printf("-> Dest    %d\n", msg.GetDest())
	fmt.Printf("-> Datalen %d\n", msg.GetDataLen())
	fmt.Printf("-> CmdId   %d\n", msg.GetCmdId())
	fmt.Printf("-> CmdFunc %d\n", msg.GetCmdFunc())
	fmt.Printf("-> DSRC    %d\n", msg.GetDSrc())
	fmt.Printf("-> DDest   %d\n", msg.GetDDest())
	fmt.Printf("-> NeedAcl %d\n", msg.GetNeedAck())
}

func displayPayload(payload []byte) bool {
	fmt.Printf("================================================>===============\n")
	log.Log.Debugf("Base64: %s", base64.RawStdEncoding.EncodeToString(payload))
	log.Log.Debugf("Payload %s", ecoflow2db.FormatByteBuffer("MQTT Body", payload))

	platform := &ecoflow2db.SendHeaderMsg{}
	err := proto.Unmarshal(payload, platform)
	if err != nil {
		log.Log.Errorf("Unable to parse message message %v: %v", payload, err)
	} else {
		switch platform.Msg.GetCmdId() {
		case 1:

			ih := &ecoflow2db.InverterHeartbeat{}
			err := proto.Unmarshal(platform.Msg.Pdata, ih)
			if err != nil {
				log.Log.Errorf("Unable to parse pdata message: %v", err)
			} else {
				log.Log.Debugf("-> InverterHearbeat %s\n", ih)
				fields := []string{"*"}
				fmt.Printf("Fields: %#v\n", fields)

				// insert := &common.Entries{DataStruct: ih,
				// 	Fields: []string{"*"}}
				// insert.Values = [][]any{{ih}}
				// _, err = storeid.Insert(tableName, insert)
				// if err != nil {
				// 	log.Log.Fatal("Error inserting record: ", err)
				// }

				log.Log.Debugf("DynamicWatts   %v", ih.GetDynamicWatts())
				log.Log.Debugf("LowerLimit     %v", ih.GetLowerLimit())
				log.Log.Debugf("PermanentWatts %v", ih.GetPermanentWatts())
				log.Log.Debugf("UpperLimit     %v", ih.GetUpperLimit())
				log.Log.Debugf("InstallCountry %v", ih.GetInstallCountry())
				log.Log.Debugf("InvOnOff       %v", ih.GetInvOnOff())
				log.Log.Debugf("Pv10pVolt      %v", ih.GetPv1OpVolt())
				log.Log.Debugf("Pv1InputVolt   %v", ih.GetPv1InputVolt())
				log.Log.Debugf("Pv1InputWatts  %v", ih.GetPv1InputWatts())
				log.Log.Debugf("Pv20pVolt      %v", ih.GetPv2OpVolt())
				log.Log.Debugf("Pv2InputVolt   %v", ih.GetPv2InputVolt())
				log.Log.Debugf("Pv2InputWatts  %v", ih.GetPv2InputWatts())
				log.Log.Debugf("Timestamp      %v", ih.GetTimestamp())
				log.Log.Debugf("Time           %v", time.Unix(int64(ih.GetTimestamp()), 0))
			}
		default:
			displayHeader(platform.Msg)
			fmt.Println("Unknown Cmd ID", platform.Msg.GetCmdId())
			fmt.Printf("received message: %s\n", ecoflow2db.FormatByteBuffer("MSG Payload", platform.Msg.Pdata))
			fmt.Printf("Base64: %s\n", base64.RawStdEncoding.EncodeToString(payload))
			return false
		}
	}
	return true
}

var serial_number = os.Getenv("ECOFLOW_DEVICE_SN")

func OnConnect(client mqtt.Client) {
	fmt.Println("Register MQTT:", serial_number)
	for _, d := range devices.Devices {
		err := ecoclient.SubscribeForParameters(d.SN, MessageHandler)
		if err != nil {
			log.Log.Errorf("Unable to subscribe for parameters %s: %v", serial_number, err)
		} else {
			log.Log.Infof("Subscribed to receive parameters %s", serial_number)
		}
	}
}

func OnConnectionLost(_ mqtt.Client, err error) {
	log.Log.Debugf("Error connection lost: %v", err)
}
func OnReconnect(mqtt.Client, *mqtt.ClientOptions) {
	log.Log.Debugf("Reconnecting...")
}

func getSnFromTopic(topic string) string {
	topicStr := strings.Split(topic, "/")
	return topicStr[len(topicStr)-1]
}
