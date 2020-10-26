package main

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/mux"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

var c mqtt.Client

func main() {
	http.Handle("/esp8266/", http.StripPrefix("/esp8266/", http.FileServer(http.Dir("files"))))
	go http.ListenAndServe(":1998", nil)

	db, _ = GetMongoDB()

	opts := mqtt.NewClientOptions().AddBroker("tcp://r3inbowari.top:1883").SetClientID("sample_go")
	opts.SetUsername("r3inb")
	opts.SetPassword("159463")
	c = mqtt.NewClient(opts)
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
	r.HandleFunc("/tw/apds/{id}", HandleAPDS12)
	r.HandleFunc("/tw/bme/{id}", HandleBME12)
	r.HandleFunc("/tw/ccs/{id}", HandleCCS12)
	r.HandleFunc("/online", HandleLoginList)

	r.HandleFunc("/coupler/{id}", HandleCoupler)
	r.HandleFunc("/info/update", HandleGetVersion)

	r.HandleFunc("/update/{id}", HandleCheckUpdate)

	r.HandleFunc("/upload/{typename}", FileUpload)
	// r.HandleFunc("/esp8266/{filename}", FileDownload)
	r.Handle("/esp8266/", http.FileServer(http.Dir("./files")))
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
	Member    int           `json:"member"`
}

type Data struct {
	Type    int       `json:"type"`
	Measure []float64 `json:"measure"`
}

func FileDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// var file os.File
	name := vars["filename"]
	ff, err := os.Open("./files/" + name)
	if err != nil {
		FailedResult(w, "file operation failed", 1, http.StatusInternalServerError, 897)
		return
	}

	w.Header().Add("Content-type", "application/octet-stream")
	w.Header().Add("content-disposition", "attachment; filename=\""+url.QueryEscape(name)+"\"")

	_, err = io.Copy(w, ff)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "Bad request")
		return
	}
}

func FileUpload(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	major, err := strconv.Atoi(r.Form.Get("major"))
	if err != nil {
		FailedResult(w, "error handle", 1, http.StatusBadRequest, 898)
		return
	}
	minor, err := strconv.Atoi(r.Form.Get("minor"))
	if err != nil {
		FailedResult(w, "error handle", 1, http.StatusBadRequest, 898)
		return
	}
	patch, err := strconv.Atoi(r.Form.Get("patch"))
	if err != nil {
		FailedResult(w, "error handle", 1, http.StatusBadRequest, 898)
		return
	}

	uploadFile, handle, err := r.FormFile("file")
	if handle == nil {
		FailedResult(w, "error handle", 1, http.StatusBadRequest, 899)
		return
	}

	vars := mux.Vars(r)
	s := vars["typename"]
	err = os.Mkdir("./files/", 0777)
	saveFile, err := os.OpenFile("./files/"+s+"_v5_"+r.Form.Get("major")+"."+r.Form.Get("minor")+"."+r.Form.Get("patch")+".bin", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		FailedResult(w, "file operation failed", 1, http.StatusInternalServerError, 500)
		return
	}
	_, _ = io.Copy(saveFile, uploadFile)
	defer uploadFile.Close()
	defer saveFile.Close()
	a := GetConfig()
	a.IotVersion = &IV{
		A: major,
		B: minor,
		C: patch,
	}
	_ = a.SetConfig()
	SucceedResult(w, "upload succeed", 1, http.StatusOK, 0)
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

type LoginBody struct {
	ID           bson.ObjectId `bson:"_id"`
	DeviceID     string        `json:"id"`
	Code         int           `json:"code"`
	Operation    int           `json:"operation"`
	Data         string        `json:"data"`
	SSID         string        `json:"ssid"` // node id
	MeshPrefix   string        `json:"prefix"`
	WifiMode     int           `json:"mode"`
	MqttBroker   string        `json:"broker"`
	MqttUsername string        `json:"username"` // node id
	Major        int           `json:"major"`
	Minor        int           `json:"minor"`
	Patch        int           `json:"patch"`
	Ts           int           `json:"ts"`
}

func (mb *LoginBody) Save(c string) error {
	mb.ID = bson.NewObjectId()
	return db.C(c).Insert(mb)
}

func logonCallback(client mqtt.Client, msg mqtt.Message) {
	var result LoginBody
	fmt.Printf("MSG: %s\n", msg.Payload())
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
	token := client.Publish("meshNetwork/to/rootNode/checkupdate", 1, false, ret)
	token.WaitTimeout(time.Second)
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

func HandleCoupler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = r.ParseForm()
	ID := vars["id"]
	member, err := strconv.Atoi(r.Form.Get("pos"))
	if err != nil {
		FailedResult(w, "error handle", 1, http.StatusBadRequest, 898)
		return
	}
	action, err := strconv.Atoi(r.Form.Get("action"))
	if err != nil {
		FailedResult(w, "error handle", 1, http.StatusBadRequest, 898)
		return
	}
	var sensorBody MessageBody
	sensorBody.DeviceID = ID
	sensorBody.Operation = action
	sensorBody.Member = member
	ret, _ := json.Marshal(sensorBody)
	println(string(ret))
	token := c.Publish("meshNetwork/to/rootNode/coupler", 1, false, ret)
	token.WaitTimeout(time.Second)
	SucceedResult(w, "oper succeed", 1, http.StatusOK, 0)
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

func HandleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	id := v["id"]
	if id == "" {
		id = "null"
	}
	version := Version{
		Major: GetConfig().IotVersion.A,
		Minor: GetConfig().IotVersion.B,
		Patch: GetConfig().IotVersion.C,
		ID:    id,
	}
	ret, _ := json.Marshal(version)
	token := c.Publish("meshNetwork/to/rootNode/checkupdate", 1, false, ret)
	token.WaitTimeout(time.Second)
	_, _ = w.Write(ret)
}

func HandleAPDS12(w http.ResponseWriter, r *http.Request) {
	var res [][]float64
	vars := mux.Vars(r)
	ID := vars["id"]
	tn := time.Now()
	for i := 0; i < 12; i++ {
		// 1-2
		ms := new(MessageBody)
		t1 := time.Date(tn.Year(), tn.Month(), tn.Day(), tn.Hour(), 0, 0, 0, time.Local).Unix() - (int64)((i+1)*3600)
		t2 := time.Date(tn.Year(), tn.Month(), tn.Day(), tn.Hour(), 0, 0, 0, time.Local).Unix() - (int64)((i)*3600)
		db.C("apds").Find(bson.M{"deviceid": ID, "ts": bson.M{"$gt": t1, "$lt": t2}}).One(&ms)
		res = append(res, ms.Data.Measure)
	}
	ret, _ := json.Marshal(res)
	_, _ = w.Write(ret)
}

func HandleCCS12(w http.ResponseWriter, r *http.Request) {
	var res [][]float64
	vars := mux.Vars(r)
	ID := vars["id"]
	tn := time.Now()
	for i := 0; i < 12; i++ {
		// 1-2
		ms := new(MessageBody)
		t1 := time.Date(tn.Year(), tn.Month(), tn.Day(), tn.Hour(), 0, 0, 0, time.Local).Unix() - (int64)((i+1)*3600)
		t2 := time.Date(tn.Year(), tn.Month(), tn.Day(), tn.Hour(), 0, 0, 0, time.Local).Unix() - (int64)((i)*3600)
		db.C("ccs").Find(bson.M{"deviceid": ID, "ts": bson.M{"$gt": t1, "$lt": t2}}).One(&ms)
		res = append(res, ms.Data.Measure)
	}
	ret, _ := json.Marshal(res)
	_, _ = w.Write(ret)
}

func HandleBME12(w http.ResponseWriter, r *http.Request) {
	var res [][]float64
	vars := mux.Vars(r)
	ID := vars["id"]
	tn := time.Now()
	for i := 0; i < 12; i++ {
		// 1-2
		ms := new(MessageBody)
		t1 := time.Date(tn.Year(), tn.Month(), tn.Day(), tn.Hour(), 0, 0, 0, time.Local).Unix() - (int64)((i+1)*3600)
		t2 := time.Date(tn.Year(), tn.Month(), tn.Day(), tn.Hour(), 0, 0, 0, time.Local).Unix() - (int64)((i)*3600)
		db.C("bme").Find(bson.M{"deviceid": ID, "ts": bson.M{"$gt": t1, "$lt": t2}}).One(&ms)
		res = append(res, ms.Data.Measure)
	}
	ret, _ := json.Marshal(res)
	_, _ = w.Write(ret)
}

func HandleLoginList(w http.ResponseWriter, r *http.Request) {
	var l []LoginBody
	tn := time.Now()
	t1 := tn.Unix()
	t2 := time.Date(tn.Year(), tn.Month(), tn.Day(), tn.Hour(), tn.Minute(), tn.Second(), 0, time.Local).Unix() - (int64)(80)
	db.C("logon").Find(bson.M{"ts": bson.M{"$gt": t2, "$lt": t1}}).All(&l)
	ret, _ := json.Marshal(l)
	_, _ = w.Write(ret)
}

func HandleGetVersion(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	id := v["id"]
	if id == "" {
		id = "null"
	}
	version := Version{
		Major: GetConfig().IotVersion.A,
		Minor: GetConfig().IotVersion.B,
		Patch: GetConfig().IotVersion.C,
		ID:    id,
	}
	ret, _ := json.Marshal(version)
	_, _ = w.Write(ret)
}
