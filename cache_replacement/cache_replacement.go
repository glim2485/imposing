package cache_replacement

import (
	"bytes"
	"custom_lib/common"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func AddPolicy(filename string, filetype string, size int, node string) (int, string) { // 0 = do not cache, 1 = cache, 2 = cannot cache, 3=cache elsewhere. Node here refers to where data was FETCHED FROM
	if size > common.TotalCacheMemory { //file is just too big to be cached
		return 2, "none"
	} else if common.TotalCacheMemory >= common.UsedCacheMemory+size { //there is enough memory to cache
		return 1, "none"
	} else if checkNearbyNodes(node) == 1 { //not enough memory but it was cached from a nearby node
		return 0, "none"
	} else { //there is not enough memory nor was the data from a nearby node
		var offloadNode string
		if common.EnableOffloading {
			offloadNode = offloadOtherNode(size)
		} else {
			offloadNode = "none"
		}
		if offloadNode != "none" { //offload the cache to another node
			fmt.Printf("\n can offload to node: %s\n", offloadNode)
			requestURL := "http://" + common.ConnectedNodes[offloadNode].Ip + ":" + common.Port + "/cachedata"
			payload := map[string]interface{}{
				"filename": filename,
				"type":     filetype,
				"size":     size,
				"node":     common.MyName,
			}
			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				fmt.Println("Error:", err)
				return 0, "none"
			}
			body := bytes.NewBuffer(jsonPayload)
			resp, err := http.Post(requestURL, "application/json", body)
			if err != nil {
				fmt.Println("Error:", err)
				return 0, "none"
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("successfully offloaded\n")
				return 4, common.ConnectedNodes[offloadNode].Name
			}
		}
		return 3, "none"
	}
}

func ReplacementPolicy(filename string, filetype string, size int, node string) (string, string) { //happens when all nearby edges have their cache full
	common.GroupCache = []common.DataSpecification{}
	type OtherCache struct {
		Cachedata []common.DataSpecification `json:"cachedata"`
	}
	if common.EnableOffloading {
		for _, v := range common.ConnectedNodes {
			if v.Type != "cloud" { //ignore cloud
				if !v.Busy {
					requestURL := "http://" + v.Ip + ":" + common.Port + "/getcachedata"
					response, err := http.Get(requestURL)
					if err != nil {
						fmt.Println("error: ", err)
					}
					defer response.Body.Close()
					body, _ := ioutil.ReadAll(response.Body)
					var otherCache OtherCache
					err = json.Unmarshal(body, &otherCache)
					if err != nil {
						fmt.Println("error: ", err)
						break
					}
					common.GroupCache = append(common.GroupCache, otherCache.Cachedata...)
				}
			}
		}
	}
	group_cache := append(common.OwnCache, common.GroupCache...) //get entire group cache
	dataCount := make(map[string][][]string)

	for _, v := range group_cache { //merge each cache entry's location and popularity
		info := []string{v.Node, strconv.Itoa(v.Popularity)}
		dataCount[v.Filename] = append(dataCount[v.Filename], info)
	}

	repeatedEntries := make([]string, 0)
	repeatedEntriesData := make(map[string][][]string)

	for name, data := range dataCount {
		if len(data) > 1 {
			repeatedEntries = append(repeatedEntries, name)
			repeatedEntriesData[name] = data
		}
	}

	var least_popular int = 99999
	var least_popular_location = "none"
	var least_popular_data = "none"

	if len(repeatedEntries) == 0 { //there are no repeated entries across the entire group. Replace the least popular one
		for name, data := range dataCount {
			current_popularity, _ := strconv.Atoi(data[0][1])
			if current_popularity < least_popular {
				least_popular = current_popularity
				least_popular_data = name
				least_popular_location = data[0][0]

			}
		}
	} else { //there are repeated entries. Replace the least popular and repeated entry
		for i := 0; i < len(repeatedEntries); i++ {
			for _, data := range repeatedEntriesData[repeatedEntries[i]] {
				current_popularity, _ := strconv.Atoi(data[1])
				if current_popularity < least_popular {
					least_popular = current_popularity
					least_popular_location = data[0]
					least_popular_data = repeatedEntries[i]
				}
			}
		}
	}
	return least_popular_data, least_popular_location
}

func checkNearbyNodes(node string) int {
	for i := 0; i < len(common.GroupEdge); i++ {
		if !common.ConnectedNodes[common.GroupEdge[i]].Busy && common.GroupEdge[i] == node { //find non-busy available node
			return 1
		}
	}
	return 0
}

func offloadOtherNode(size int) string { //api call nearby nodes to get their leftover cache size in order of nearest to furthest
	for i := 0; i < len(common.GroupEdge); i++ {
		if !common.ConnectedNodes[common.GroupEdge[i]].Busy { //find storage of non-busy nodes
			requestURL := "http://" + common.ConnectedNodes[common.GroupEdge[i]].Ip + ":" + common.Port + "/getfree"
			response, err := http.Get(requestURL)
			if err != nil {
				fmt.Println("error: ", err)
			}
			defer response.Body.Close()
			contentType := response.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(response.Body)
			if strings.HasPrefix(contentType, "application/json") {
				type free_struct struct {
					Free int `json:"free"`
				}
				var leftover free_struct
				err := json.Unmarshal(body, &leftover)
				if err != nil {
					fmt.Println("error: ", err)
					break
				}
				if size <= leftover.Free {
					return common.GroupEdge[i]
				}

			}
		}
	}
	return "none"
}
