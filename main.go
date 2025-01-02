package main

import (
	"bytes"
	apicalls "custom_lib/apicalls"
	"custom_lib/cache_replacement"
	common "custom_lib/common"
	"custom_lib/timecheck"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//remember to add an outgoing delay based on static delay (distance) to simulate outgoing lag

func init() {
	common.MyName = "seouledge3"
	common.MyIP = "150.109.247.150"
	common.MyType = "edge"
	common.CurrentBusyLevel = 100
	common.UsedCacheMemory = 0
	common.CurrentBusy = false
	common.GroupEdge = append(common.GroupEdge, "none")
	common.EnableOffloading = true

	//add edge/cloud nodes with static latency. Static latency here defines the latency between two nodes if both nodes were not busy. Aka pseudo-physical distance
	common.ConnectedNodes["Seoul1"] = common.NodeStruct{"Seoul1", "cloud", "43.131.234.197", 200, false}
	common.ConnectedNodes["seouledge1"] = common.NodeStruct{"seouledge1", "edge", "43.131.232.253", 200, false}
	common.ConnectedNodes["seouledge2"] = common.NodeStruct{"seouledge2", "edge", "43.155.131.94", 200, false}
}

func main() {
	go server()                   //start up server
	go UpdateNodeLatencies(false) //keep updating node latencies
	time.Sleep(2 * time.Second)

	var choice int
	var filename string
	var filetype string
	var filesize int
	fmt.Printf("Current Node name - %s\n Current Node IP - %s\n Current Node type - %s\n", common.MyName, common.MyIP, common.MyType)
	for {
		fmt.Printf("As the cloud, what would you like to do?\n 1. Create new data\n 2. Print current information\n 3. Fetch data\n 6. Change busy status \n 7. Change current busy level\n 8. Update Latencies\n 9.Close the server\n")
		fmt.Printf("your input: ")
		fmt.Scanf("%d", &choice)
		switch choice {
		case 99: //dummy data, can be deleted later
			for i := 1; i < 20; i++ {
				spread := 0
				filename := strconv.Itoa(i)
				new_entry := common.DataInformation{filename, "mp4", common.MyName, false}
				new_data := common.DataSpecification{filename, "mp4", 200, common.MyName, 0}

				//spread to cloud first if its an edge
				if common.MyType != "cloud" {
					spread = common.UpdateCloud(new_entry) //includes outgoing delay. Returns 1 when success
				}

				//update data location in local server if it was successfully updated in cloud (in case of edge)
				if common.MyType == "cloud" || spread == 1 {
					common.DataLocation = append(common.DataLocation, new_entry) //update data location
					common.OwnData = append(common.OwnData, new_data)            //upate own data list
					fmt.Printf("new data added!:")
					fmt.Printf("%+v\n", new_entry)
				} else {
					fmt.Printf("failed to upload data\n")
				}
			}
		case 1:
			spread := 0
			fmt.Printf("You have chosen to add new data!\n")
			fmt.Printf("Input name of file: ")
			fmt.Scanf("%s", &filename)
			fmt.Printf("Input type of file: ")
			fmt.Scanf("%s", &filetype)
			fmt.Printf("Input size of file (MB): ")
			fmt.Scanf("%d", &filesize)
			new_entry := common.DataInformation{filename, filetype, common.MyName, false}
			new_data := common.DataSpecification{filename, filetype, filesize, common.MyName, 0}

			//spread to cloud first if its an edge
			if common.MyType != "cloud" {
				spread = common.UpdateCloud(new_entry) //includes outgoing delay. Returns 1 when success
			}

			//update data location in local server if it was successfully updated in cloud (in case of edge)
			if common.MyType == "cloud" || spread == 1 {
				common.DataLocation = append(common.DataLocation, new_entry) //update data location
				common.OwnData = append(common.OwnData, new_data)            //upate own data list
				fmt.Printf("new data added!:")
				fmt.Printf("%+v\n", new_entry)
				common.UpdateEdge(new_entry) //update edges about new entry
			} else {
				fmt.Printf("failed to upload data\n")
			}
		case 2:
			fmt.Printf("These are the current data locations stored in this node:\n")
			fmt.Printf("%+v\n", common.DataLocation)
			fmt.Printf("\n")
			fmt.Printf("These are the current data that EXISTS in this node: \n")
			fmt.Printf("%+v\n", common.OwnData)
			fmt.Printf("These are the current data that are CACHED in this node: \n")
			fmt.Printf("%+v\n", common.OwnCache)
			fmt.Printf("current busy: %t\n", common.CurrentBusy)
			fmt.Printf("Other nodes status:\n")
			fmt.Printf("%+v\n", common.ConnectedNodes)
		case 3:
			fmt.Printf("enter data you want to fetch: ")
			var request_file string
			fmt.Scanf("%s", &request_file)
			request_file_array := strings.Split(request_file, ".")
			if len(request_file_array) != 2 {
				request_file_array = append(request_file_array, "mp4")
			}
			possible_nodes, local, remote, cached := findData(request_file_array[0], request_file_array[1])
			if local { //data exists in local so no need to go remote
				fmt.Printf("data exists in this current node. No need to fetch!\n")
				if cached {
					fmt.Printf("but this is cached data!\n")
					addPopularityCache(request_file_array[0], request_file_array[1]) //add popularity counters
				}
			} else if remote { //data exists remotely, so go check
				fmt.Printf("data exists but its not in this node, going to fetch...\n")
				fetchData(possible_nodes, request_file_array)
			} else { //check the main cloud
				if common.MyType == "cloud" { //if this is the cloud, it doesnt exist
					fmt.Printf("data does not exist anywhere\n")
				} else { //if this node is an edge
					fmt.Printf("updating edge's metadata")
					time.Sleep(time.Duration(common.StaticNodeLatency["Seoul1"].Latency) * time.Millisecond) //add delay
					requestURL := "http://" + common.CloudIP + ":" + common.Port + "/getmetadata"
					response, err := http.Get(requestURL)
					if err != nil {
						fmt.Println("error: ", err)
					}
					defer response.Body.Close()
					contentType := response.Header.Get("Content-Type")
					body, _ := ioutil.ReadAll(response.Body)

					if strings.HasPrefix(contentType, "application/json") {
						type Metadata struct {
							Metadata []common.DataInformation `json:"metadata"`
						}

						var newMetadata Metadata
						err := json.Unmarshal(body, &newMetadata)
						if err != nil {
							fmt.Println("error: ", err)
							break
						}
						common.DataLocation = newMetadata.Metadata                                            //update with global metadata
						possible_nodes, _, remote, _ = findData(request_file_array[0], request_file_array[1]) //find data again
						if remote {                                                                           //no need to check for local
							fetchData(possible_nodes, request_file_array)
						} else {
							fmt.Printf("data does not exist anywhere\n")
						}
					} else if strings.HasPrefix(contentType, "text/plain") { //display api call error message
						fmt.Println(string(body))
					}
				}
			}
		case 6:
			common.CurrentBusy = !common.CurrentBusy
			fmt.Printf("This current node busy: %t\n", common.CurrentBusy)
			SpreadBusy()
		case 7:
			fmt.Printf("enter new busy timer in ms (current is %d):", common.CurrentBusyLevel)
			fmt.Scanf("%d", &common.CurrentBusyLevel)
			apicalls.DefaultSleep = time.Duration(common.CurrentBusyLevel) * time.Millisecond
		case 8:
			fmt.Printf("updating latencies...\n")
			UpdateNodeLatencies(true)
		case 9:
			fmt.Printf("Closing servers!\n")
			return
		}
	}
}

func server() {
	router := gin.Default()
	router.GET("/ping", apicalls.NodePing)
	router.POST("/updatedata", apicalls.UpdateData)
	router.GET("/getdata", apicalls.FetchData)
	router.GET("/getmetadata", apicalls.FetchMetadata)
	router.GET("/getcachedata", apicalls.FetchCachedata)
	router.GET("/getfree", apicalls.GetFreeCacheMemory)
	router.POST("/cachedata", apicalls.CacheData)
	router.POST("/replacedata", apicalls.ReplaceData)
	router.POST("/deletedata", apicalls.DeleteData)
	router.GET("/updatebusy", apicalls.UpdateBusy)

	router.Run("0.0.0.0:8080")
}

// function to find data locally first
func findData(filename string, filetype string) ([]string, bool, bool, bool) {
	var local bool = false
	var remote bool = false
	var cached bool = false
	var owning_nodes []string

	for i := 0; i < len(common.OwnData); i++ { //find if you own the data
		if common.OwnData[i].Filename == filename && common.OwnData[i].Type == filetype {
			local = true
			cached = false
			remote = false
			return owning_nodes, local, remote, cached
		}
	}

	for i := 0; i < len(common.OwnCache); i++ { //find data in cache
		if common.OwnCache[i].Filename == filename && common.OwnCache[i].Type == filetype {
			local = true
			cached = true
			remote = false
			return owning_nodes, local, remote, cached
		}
	}

	for i := 0; i < len(common.DataLocation); i++ { //see if data exists
		if common.DataLocation[i].Filename == filename && common.DataLocation[i].Type == filetype {
			remote = true
			owning_nodes = append(owning_nodes, common.DataLocation[i].Node)
		}
	}
	return owning_nodes, local, remote, cached
}

func getClosestNode(owned_nodes []string) string {
	var closest_node string = owned_nodes[0]
	for i := 1; i < len(owned_nodes); i++ {
		if common.ConnectedNodes[owned_nodes[i]].Latency < common.ConnectedNodes[closest_node].Latency {
			closest_node = owned_nodes[i]
		}
	}
	return closest_node
}

func UpdateNodeLatencies(manual bool) {
	for {
		apicalls.DefaultSleep = time.Duration(common.CurrentBusyLevel) * time.Millisecond

		for key, value := range common.ConnectedNodes {
			latency := timecheck.GetLatency(value.Ip)

			if thisNode, ok := common.ConnectedNodes[key]; ok {
				thisNode.Latency = latency
				if latency >= common.ConnectedNodes["Seoul1"].Latency && common.ConnectedNodes[key].Type != "cloud" {
					thisNode.Busy = true
				} else {
					thisNode.Busy = false
				}
				common.ConnectedNodes[key] = thisNode
			}
		}
		//fmt.Printf("Latencies Updated!\n")
		//fmt.Printf("%+v\n", common.ConnectedNodes)
		if manual {
			return
		}
		time.Sleep(60 * time.Second)
	}
}

func addPopularityCache(filename string, filetype string) {
	for i := 0; i < len(common.OwnCache); i++ {
		if common.OwnCache[i].Filename == filename && common.OwnCache[i].Type == filetype {
			common.OwnCache[i].Popularity++
		}
	}
}

func fetchData(possible_nodes []string, request_file_array []string) {

	closest_node := getClosestNode(possible_nodes)
	type response_struct struct {
		Filename string `json:"filename"`
		Type     string `json:"type"`
		Node     string `json:"node"`
		Size     int    `json:"size"`
	}
	var response_data response_struct
	requestURL := "http://" + common.StaticNodeLatency[closest_node].Ip + ":" + common.Port + "/getdata?requester=" + common.MyName + "&filename=" + request_file_array[0] + "&filetype=" + request_file_array[1]
	time.Sleep(time.Duration(common.StaticNodeLatency[closest_node].Latency) * time.Millisecond) //outgoing delay
	response, err := http.Get(requestURL)
	if err != nil {
		fmt.Println("error: ", err)
	}
	defer response.Body.Close()
	contentType := response.Header.Get("Content-Type")
	body, _ := ioutil.ReadAll(response.Body)

	if strings.HasPrefix(contentType, "application/json") {
		json.Unmarshal(body, &response_data)
		canadd, target := cache_replacement.AddPolicy(response_data.Filename, response_data.Type, response_data.Size, response_data.Node)
		switch canadd {
		case 0:
			fmt.Printf("no need to cache\n")
		case 1: //successfully cached
			fmt.Printf("%s.%s was fetched from %s and cached into this node\n", response_data.Filename, response_data.Type, response_data.Node)
			new_entry := common.DataInformation{response_data.Filename, response_data.Type, common.MyName, true}
			common.DataLocation = append(common.DataLocation, new_entry) //update data locations
			new_data := common.DataSpecification{response_data.Filename, response_data.Type, response_data.Size, common.MyName, 1}
			common.OwnCache = append(common.OwnCache, new_data)
			common.UsedCacheMemory += response_data.Size
			if common.MyType != "cloud" { //update main cloud's metadata to include cached information
				common.UpdateCloud(new_entry)
			} else {
				common.UpdateEdge(new_entry)
			}
		case 3: //all caches are full and needs replacement
			fmt.Printf("cannot cache freely, need replacement policy\n")
			least_popular_data, least_popular_location := cache_replacement.ReplacementPolicy(response_data.Filename, response_data.Type, response_data.Size, response_data.Node)
			fmt.Printf("\nleast_popular_data: %s, least_popular_location: %s\n", least_popular_data, least_popular_location)
			if least_popular_location == common.MyName { //data to be replaced is in this node
				fmt.Printf("Data to be replaced cached locally\n")
				for i := 0; i < len(common.OwnCache); i++ { //modify cache first
					if common.OwnCache[i].Filename == least_popular_data {
						common.UsedCacheMemory -= common.OwnCache[i].Size                                                                      //restore size
						new_data := common.DataSpecification{response_data.Filename, response_data.Type, response_data.Size, common.MyName, 1} //add new data
						common.OwnCache = append(common.OwnCache, new_data)
						common.OwnCache[i] = common.OwnCache[len(common.OwnCache)-1] //remove data
						common.OwnCache = common.OwnCache[:len(common.OwnCache)-1]
						common.UsedCacheMemory += response_data.Size
						break
					}
				}
				for i := 0; i < len(common.DataLocation); i++ { //modify metadata
					if common.DataLocation[i].Filename == least_popular_data && common.DataLocation[i].Node == least_popular_location {
						common.DataLocation[i] = common.DataLocation[len(common.DataLocation)-1] //delete from local metadata
						common.DataLocation = common.DataLocation[:len(common.DataLocation)-1]
						if common.MyType != "cloud" {
							common.DeleteCloud(least_popular_data, least_popular_location) //delete entry from cloud
							common.DeleteEdge(least_popular_data, least_popular_location)  //delete from nearby nodes
							new_entry := common.DataInformation{response_data.Filename, response_data.Type, common.MyName, true}
							common.UpdateCloud(new_entry) //update cloud with new cache data
							common.UpdateEdge(new_entry)
						}
						break
					}
				}
			} else { //data to be replaced is in another node
				fmt.Printf("data is in another node\n")
				remoteReplaceData(least_popular_data, least_popular_location, response_data.Filename, response_data.Type, response_data.Size)
			}
		case 2:
			fmt.Printf("file too big to be cached\n")
		case 4: //add an entry that leads to the other node
			fmt.Printf("cache added to another node\n")
			new_entry := common.DataInformation{response_data.Filename, response_data.Type, target, true}
			common.DataLocation = append(common.DataLocation, new_entry)
			common.UpdateCloud(new_entry)
			common.UpdateEdge(new_entry)
		}
	} else if strings.HasPrefix(contentType, "text/plain") {
		fmt.Println(string(body))
	}
}

func remoteReplaceData(least_popular_data string, least_popular_location string, new_data string, new_data_type string, new_data_size int) int {
	//create json payload
	payload := map[string]interface{}{
		"replacefilename": least_popular_data,
		"filename":        new_data,
		"type":            new_data_type,
		"size":            new_data_size,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}

	//prepare http request
	url := "http://" + common.CloudIP + ":" + common.Port + "/replacedata"
	body := bytes.NewBuffer(jsonPayload)

	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("offload cache ok\n")
		return 1
	} else {
		fmt.Printf("offload cache failed\n")
		return 0
	}
}

func SpreadBusy() { //use it to spread the current status of THIS node
	var currentStatus string
	if common.CurrentBusy {
		currentStatus = "true"
	} else {
		currentStatus = "false"
	}
	for _, v := range common.ConnectedNodes {
		if v.Type != "cloud" {
			requestURL := "http://" + v.Ip + ":" + common.Port + "/updatebusy?requester=" + common.MyName + "&status=" + currentStatus
			response, err := http.Get(requestURL)
			if err != nil {
				fmt.Println("error: ", err)
			}
			defer response.Body.Close()
		}
	}
}
