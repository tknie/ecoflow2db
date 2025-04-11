package ecoflow2db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/signal"
	reflect "reflect"
	"sort"
	"strings"
	sync "sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tess1o/go-ecoflow"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
)

var ecoclient *ecoflow.MqttClient

var devices *ecoflow.DeviceListResponse
var tableName string

var mqttid common.RegDbID

var (
	mu sync.Mutex
)

func InitMqtt(user, password string) {
	fmt.Println("Initialize MQTT client")
	configuration := ecoflow.MqttClientConfiguration{
		Email:            user,
		Password:         password,
		OnConnect:        OnConnect,
		OnConnectionLost: OnConnectionLost,
		OnReconnect:      OnReconnect,
	}
	var err error
	ecoclient, err = ecoflow.NewMqttClient(context.Background(), configuration)
	if err != nil {
		log.Log.Fatalf("Error new MQTT client: %v", err)
	}
	mqttid = openDatabase()
	log.Log.Debugf("Strt mqtt Ecoflow connect")
	fmt.Println("Connecting MQTT client")
	ecoclient.Connect()
	log.Log.Debugf("Wait for Ecoflow disconnect")
	fmt.Println("Waiting for MQTT data")
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

	fmt.Printf("Received message of %s at %v\n", serialNumber, time.Now())

	log.Log.Debugf("received message on topic %s; body (retain: %t): %s\n", msg.Topic(),
		msg.Retained(), FormatByteBuffer("MQTT Body", msg.Payload()))
	payload := msg.Payload()

	data := make(map[string]interface{})
	err := json.Unmarshal(payload, &data)
	if err == nil {
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
		tn := serialNumber + "_mqtt"
		checkTable(mqttid, tn, func() []*common.Column {
			keys := make([]string, 0, len(data))
			for k := range data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			columns := make([]*common.Column, 0)
			// prefix := ""
			for _, k := range keys {
				v := data[k]
				// prefix = strings.Split(k, ".")[0]
				// name := "eco_" + strings.ReplaceAll(k[len(prefix)+1:], ".", "_")
				name := "eco_" + strings.ReplaceAll(k, ".", "_")
				log.Log.Debugf("Add column %s=%v %T -> %s\n", k, v, v, name)
				switch val := v.(type) {
				case string:
					columns = append(columns, &common.Column{Name: name, DataType: common.Alpha, Length: 255})
				case float64:
					if val == math.Trunc(val) && val < math.MaxInt64 {
						columns = append(columns, &common.Column{Name: name, DataType: common.BigInteger, Length: 4})
					} else {
						columns = append(columns, &common.Column{Name: name, DataType: common.Decimal, Length: 8})
					}
				case []interface{}, map[string]interface{}:
					columns = append(columns, &common.Column{Name: name, DataType: common.Alpha, Length: 1024})
				default:
					fmt.Printf("Unknown type %s=%T\n", k, v)
					log.Log.Fatalf("Unknown type %s=%T\n", k, v)
				}
			}
			return columns
		})
		if v, ok := data["mppt.outVol"]; ok {
			fmt.Printf("OutVol       : %f\n", v.(float64))
		}
		fmt.Println("Timestamp    :", time.Unix(int64(data["timestamp"].(float64)), 0))
		fmt.Printf("Data: %#v\n", data)
		insertTable(mqttid, tn, func() ([]string, [][]any) {
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
				case []interface{}, map[string]interface{}:
					columns = append(columns, fmt.Sprintf("%#v", val))
				default:
					fmt.Printf("Unknown type %s=%T\n", k, v)
					log.Log.Fatalf("Unknown type %s=%T\n", k, v)
				}
			}
			return fields, [][]any{columns}
		})
		return
	}

	start := 0
	end := len(payload)
	index := bytes.Index(payload, []byte(serialNumber))
	if index != -1 {
		end = index + len(serialNumber)
	}
	log.Log.Debugf("Serial index 1: %d/%d %d:%d", index, len(payload), start, end)
	displayPayload(serialNumber, payload[start:end])
	start = end
	if len(payload) > index+len(serialNumber) {
		index = bytes.Index(payload[end:], []byte(serialNumber))
		if index != -1 {
			end = end + index + len(serialNumber)
		} else {
			end = len(payload)
		}
		log.Log.Debugf("Serial index 2: %d", index)
		displayPayload(serialNumber, payload[start:end])
	}

}

func displayHeader(msg *Header) {
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

func OnConnect(client mqtt.Client) {
	for _, d := range devices.Devices {
		fmt.Println("Register MQTT:", d.SN)
		err := ecoclient.SubscribeForParameters(d.SN, MessageHandler)
		if err != nil {
			log.Log.Errorf("Unable to subscribe for parameters %s: %v", d.SN, err)
		} else {
			log.Log.Infof("Subscribed to receive parameters %s", d.SN)
		}
	}
}

func getType(myvar interface{}) string {
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
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
