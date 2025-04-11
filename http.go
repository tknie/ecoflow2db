package ecoflow2db

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tess1o/go-ecoflow"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
)

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
	id := openDatabase()
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
		}
	}

	for {
		select {
		case <-httpDone:
			return
		case <-time.After(time.Minute * 1):

			for _, l := range devices.Devices {
				if l.Online == 1 {
					tn := "device_" + l.SN + "_quota"
					resp, err := client.GetDeviceAllParameters(context.Background(), l.SN)
					if err != nil {
						log.Log.Fatalf("Error getting device list: %v", err)
					}
					insertTable(id, tn, func() ([]string, [][]any) {
						keys := make([]string, 0)
						for k := range resp {
							keys = append(keys, k)
						}
						sort.Strings(keys)
						columns := make([]any, 0)
						// prefix := ""
						fields := make([]string, 0)
						for _, k := range keys {
							v := resp[k]
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
					// fmt.Println("Resp: ", resp)
				}
			}
		}
		fmt.Println("time after", time.Now())
	}
}

func endHttp() {
	httpDone <- true
}
