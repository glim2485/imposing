package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var MyName string
var MyIP string
var MyType string
var CurrentBusy bool
var CloudIP string = "43.131.234.197"
var Port string = "8080"
var TotalCacheMemory int = 1024
var UsedCacheMemory int
var EnableOffloading bool

type NodeStruct struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Ip      string `json:"ip"`
	Latency int    `json:"latency"`
	Busy    bool   `json:"busy"`
}

// this will be used dynamically to introduce latency
var CurrentBusyLevel int

type DataInformation struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Node     string `json:"node"`
	Cache    bool   `json:"cache"`
}

type DataSpecification struct {
	Filename   string `json:"filename"`
	Type       string `json:"type"`
	Size       int    `json:"size"`
	Node       string `json:"node"`
	Popularity int    `json:"popularity"`
}

var ConnectedNodes = make(map[string]NodeStruct)
var DataLocation []DataInformation
var StaticNodeLatency = make(map[string]NodeStruct)
var OwnCache []DataSpecification
var GroupCache []DataSpecification //cache from neighbor data excluding own
var OwnData []DataSpecification
var GroupEdge []string

func UpdateCloud(new_entry DataInformation) int {
	delay := time.Duration(StaticNodeLatency["Seoul1"].Latency) * time.Millisecond
	time.Sleep(delay)
	//create json payload
	payload := map[string]interface{}{
		"filename": new_entry.Filename,
		"type":     new_entry.Type,
		"node":     new_entry.Node,
		"cache":    new_entry.Cache,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}

	//prepare http request
	url := "http://" + CloudIP + ":" + Port + "/updatedata"
	body := bytes.NewBuffer(jsonPayload)

	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("updating cloud successs\n")
		return 1
	} else {
		fmt.Printf("updating cloud failed\n")
		return 0
	}
}

func UpdateEdge(new_entry DataInformation) {
	//create json payload
	payload := map[string]interface{}{
		"filename": new_entry.Filename,
		"type":     new_entry.Type,
		"node":     new_entry.Node,
		"cache":    new_entry.Cache,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error:", err)
	}

	//prepare http request
	body := bytes.NewBuffer(jsonPayload)

	for _, k := range GroupEdge {
		url := "http://" + ConnectedNodes[k].Ip + ":" + Port + "/updatedata"
		resp, err := http.Post(url, "application/json", body)
		if err != nil {
			fmt.Println("Error:", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("updating %s successs\n", k)
		} else {
			fmt.Printf("updating %s failed\n", k)
		}
	}
}

func DeleteCloud(filename string, node string) int {
	url := "http://" + CloudIP + ":" + Port + "/deletedata"
	payload := map[string]interface{}{
		"filename": filename,
		"node":     node,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}
	body := bytes.NewBuffer(jsonPayload)
	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("deleting from cloud successs\n")
		return 1
	} else {
		fmt.Printf("deleting from cloud failed\n")
		return 0
	}
}

func DeleteEdge(filename string, node string) {
	payload := map[string]interface{}{
		"filename": filename,
		"node":     node,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error:", err)
	}
	body := bytes.NewBuffer(jsonPayload)

	for _, k := range GroupEdge {
		url := "http://" + ConnectedNodes[k].Ip + ":" + Port + "/deletedata"
		resp, err := http.Post(url, "application/json", body)
		if err != nil {
			fmt.Println("Error:", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("deleting from edge successs\n")
		} else {
			fmt.Printf("deleting from edge failed\n")
		}
	}
}
