package main

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/mux"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"time"
)

type Version struct {
	ID    string `json:"id"`
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Patch int    `json:"patch"`
}

func GetMongoDB() (*mgo.Database, error) {
	host := "mongodb://r3inbowari.top:27017"
	dbName := "pear"
	session, err := mgo.Dial(host)
	if err != nil {
		return nil, err
	}
	db := session.DB(dbName)
	return db, nil
}

var db *mgo.Database

var hash_tem = make(map[string]float64)
var hash_hum = make(map[string]float64)
var hash_pre = make(map[string]float64)
var hash_alt = make(map[string]float64)
var bme_ts = make(map[string]int)

var hash_co2 = make(map[string]float64)
var hash_tvoc = make(map[string]float64)
var ccs_ts = make(map[string]int)

var hash_clear = make(map[string]float64)
var hash_red = make(map[string]float64)
var hash_green = make(map[string]float64)
var hash_blue = make(map[string]float64)
var apds_ts = make(map[string]int)

func main() {
	db, _ = GetMongoDB()

	opts := mqtt.NewClientOptions().AddBroker("tcp://r3inbowari.top:1883").SetClientID("sample_go")
	opts.SetUsername("r3inb")
	opts.SetPassword("159463")
	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := c.Subscribe("meshNetwork/from/rootNode/bme", 0, bmeCallback); token.Wait() && token.Error() != nil {
	}
	if token := c.Subscribe("meshNetwork/from/rootNode/apds", 0, apdsCallback); token.Wait() && token.Error() != nil {
	}
	if token := c.Subscribe("meshNetwork/from/rootNode/logon", 0, logonCallback); token.Wait() && token.Error() != nil {
	}
	if token := c.Subscribe("meshNetwork/from/rootNode/ccs", 0, ccsCallback); token.Wait() && token.Error() != nil {
	}
	if token := c.Subscribe("meshNetwork/from/rootNode/checkupdate", 0, updateCallback); token.Wait() && token.Error() != nil {
	}

	r := mux.NewRouter()
	r.HandleFunc("/check_update", HandleVersion)
	r.HandleFunc("/bme/{id}", HandleBME)
	r.HandleFunc("/ccs/{id}", HandleCCS)
	r.HandleFunc("/apds/{id}", HandleAPDS)
	log.Fatal(http.ListenAndServe(":2999", r))

	time.Sleep(time.Hour)

}

type Result struct {
	ID string `json:"id"`
}

type MessageBody struct {
	ID        bson.ObjectId `bson:"_id"`
	DeviceID  string        `json:"id"`
	MID       string        `json:"mid"` // node id
	Ts        int           `json:"ts"`
	Operation int           `json:"operation"`
	Data      Data          `json:"data"`
}

type Data struct {
	Type    int       `json:"type"`
	Measure []float64 `json:"measure"`
}

func bmeCallback(client mqtt.Client, msg mqtt.Message) {
	var result MessageBody
	_ = json.Unmarshal(msg.Payload(), &result)
	_ = result.Save("bme")
	hash_tem[result.DeviceID] = result.Data.Measure[0]
	hash_hum[result.DeviceID] = result.Data.Measure[1]
	hash_pre[result.DeviceID] = result.Data.Measure[2]
	hash_alt[result.DeviceID] = result.Data.Measure[3]
	bme_ts[result.DeviceID] = result.Ts
	fmt.Println("[INFO] Topic ->", msg.Topic(), "Inserted BME Data ->", result.ID)
}

func (mb *MessageBody) Save(c string) error {
	mb.ID = bson.NewObjectId()
	return db.C(c).Insert(mb)
}

func apdsCallback(client mqtt.Client, msg mqtt.Message) {
	var result MessageBody
	_ = json.Unmarshal(msg.Payload(), &result)
	_ = result.Save("apds")
	hash_clear[result.DeviceID] = result.Data.Measure[0]
	hash_red[result.DeviceID] = result.Data.Measure[1]
	hash_green[result.DeviceID] = result.Data.Measure[2]
	hash_blue[result.DeviceID] = result.Data.Measure[3]
	apds_ts[result.DeviceID] = result.Ts
	fmt.Println("[INFO] Topic ->", msg.Topic(), "Inserted APDS Data ->", result.ID)
}

func ccsCallback(client mqtt.Client, msg mqtt.Message) {
	var result MessageBody
	_ = json.Unmarshal(msg.Payload(), &result)
	_ = result.Save("ccs")
	hash_co2[result.DeviceID] = result.Data.Measure[0]
	hash_tvoc[result.DeviceID] = result.Data.Measure[1]
	ccs_ts[result.DeviceID] = result.Ts
	fmt.Println("[INFO] Topic ->", msg.Topic(), "Inserted CCS Data ->", result.ID)
}

func logonCallback(client mqtt.Client, msg mqtt.Message) {
	var result MessageBody
	_ = json.Unmarshal(msg.Payload(), &result)
	_ = result.Save("logon")
	fmt.Println("[INFO] Topic ->", msg.Topic(), "Inserted BME Data ->", result.ID)
}

func updateCallback(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
	var result Result
	_ = json.Unmarshal(msg.Payload(), &result)

	version := Version{
		Major: GetConfig().IotVersion.A,
		Minor: GetConfig().IotVersion.B,
		Patch: GetConfig().IotVersion.C,
		ID:    result.ID,
	}
	ret, _ := json.Marshal(version)
	client.Publish("meshNetwork/to/rootNode/checkupdate", 1, false, ret)
}

type SensorBody struct {
	ID   string    `json:"id"`
	TS   int       `json:"ts"`
	Data []float64 `json:"data"`
}

func HandleBME(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var sensorBody SensorBody
	sensorBody.ID = vars["id"]
	sensorBody.TS = bme_ts[vars["id"]]
	sensorBody.Data = append(sensorBody.Data, hash_hum[vars["id"]])
	sensorBody.Data = append(sensorBody.Data, hash_tem[vars["id"]])
	sensorBody.Data = append(sensorBody.Data, hash_alt[vars["id"]])
	sensorBody.Data = append(sensorBody.Data, hash_pre[vars["id"]])
	fmt.Printf("[INFO] Get BME value: %+v\n", vars["id"])
	ret, _ := json.Marshal(sensorBody)
	_, _ = w.Write(ret)
}

func HandleCCS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var sensorBody SensorBody
	sensorBody.ID = vars["id"]
	sensorBody.TS = ccs_ts[vars["id"]]
	sensorBody.Data = append(sensorBody.Data, hash_co2[vars["id"]])
	sensorBody.Data = append(sensorBody.Data, hash_tvoc[vars["id"]])
	fmt.Printf("[INFO] Get CCS value: %+v\n", vars["id"])
	ret, _ := json.Marshal(sensorBody)
	_, _ = w.Write(ret)
}

func HandleAPDS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var sensorBody SensorBody
	sensorBody.ID = vars["id"]
	sensorBody.TS = apds_ts[vars["id"]]
	sensorBody.Data = append(sensorBody.Data, hash_clear[vars["id"]])
	sensorBody.Data = append(sensorBody.Data, hash_red[vars["id"]])
	sensorBody.Data = append(sensorBody.Data, hash_green[vars["id"]])
	sensorBody.Data = append(sensorBody.Data, hash_blue[vars["id"]])
	fmt.Printf("[INFO] Get APDS value: %+v\n", vars["id"])
	ret, _ := json.Marshal(sensorBody)
	_, _ = w.Write(ret)
}

func HandleVersion(w http.ResponseWriter, r *http.Request) {
	version := Version{
		Major: GetConfig().IotVersion.A,
		Minor: GetConfig().IotVersion.B,
		Patch: GetConfig().IotVersion.C,
	}
	ret, _ := json.Marshal(version)
	_, _ = w.Write(ret)
}
