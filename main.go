package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"zk/lib"
)

//StartNewSocket ...
func StartNewSocket(station int, host string, port int, replyChan chan lib.Attendance) {
	zk, err := lib.MustConnect(host, port)
	defer func() {
		if v := recover(); v != nil {
			fmt.Println(v)
		}
		zk.Disconnect()
	}()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connect success", host, port)
	zk.SetTime()
	sn, err := zk.GetSerialNumber()
	if err != nil {
		panic(err)
	}
	zk.SerialNumber = strings.Split(string(sn.PayloadData), "=")[1] //cắt chuỗi
	err = zk.EnableRealtime()
	if err != nil {
		panic(err)
	}
	for {
		event, err := zk.RecieveEvent()
		if err != nil {
			panic(err)
		}
		//fmt.Println(lib.Struct2String(event))
		if event.SessionCode == lib.EF_ATTLOG {
			data := event.PayloadData
			att, err := lib.ParseAttendance(host, data)
			att.SerialNumber = zk.SerialNumber
			att.StationID = station
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println(lib.Struct2String(*att))
			replyChan <- *att //Gán

		}
	}
}

func processEvent(attChan chan lib.Attendance) {
	for {
		select {
		case reply := <-attChan:
			{
				dataform := &Attendance{
					StationID:    reply.StationID,
					SerialNumber: reply.SerialNumber[0 : len(reply.SerialNumber)-1],
					MachineIP:    reply.MachineIP,
					UserID:       reply.UserID,
					VerifyType:   reply.VerifyType,
					Status:       reply.Status,
					AttTime:      reply.AttTime,
				}
				b, err := json.Marshal(dataform) //Struct to Json: Json -> map[string]{} json.Unmarshal
				if err != nil {
					fmt.Println(">>>>>>>>>>>>>>")
					fmt.Println(err)
					return
				}
				fmt.Println(string(b))
				client := &http.Client{}
				req, err := http.NewRequest("POST", "http://172.16.120.187:80/checkers", bytes.NewBuffer(b))
				req.Close = true
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					log.Fatal(err)
				}
				defer resp.Body.Close()

			}
		}
	}

}

type Attendance struct {
	StationID    int    `json:"station"`
	SerialNumber string `json:"serial_number"`
	MachineIP    string `json:"machine_ip"`
	UserID       string `json:"user_id"`
	VerifyType   int    `json:"verify_type"`
	Status       int    `json:"status"`
	AttTime      string `json:"att_time"`
}

func main() {

	replyChan := make(chan lib.Attendance)

	go StartNewSocket(71, "172.16.70.200", 4370, replyChan)
	go StartNewSocket(72, "172.16.70.177", 4370, replyChan)
	go StartNewSocket(5, "172.16.50.175", 4370, replyChan)
	go StartNewSocket(4, "172.16.80.234", 4370, replyChan)
	go StartNewSocket(6, "172.16.80.210", 4370, replyChan)
	go StartNewSocket(3, "172.16.60.110", 4370, replyChan)
	go StartNewSocket(2, "172.16.50.171", 4370, replyChan)
	go StartNewSocket(8, "172.16.30.45", 4370, replyChan)
	go processEvent(replyChan)

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Println("exit")

	fmt.Println("connect ok")

}
