package apicalls

import (
	"custom_lib/common"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// NodePing, server-wise, will work to check ping
// it has a built-in delay to simulate delay in network

var DefaultSleep = time.Duration(common.CurrentBusyLevel * int(time.Millisecond))

func NodePing(c *gin.Context) {
	time.Sleep(DefaultSleep)

	c.Data(http.StatusOK, "text/plain", []byte("Connection OK"))
}

// this function will be used mostly by the cloud to update data created by edges
func UpdateData(c *gin.Context) {
	time.Sleep(DefaultSleep)

	var received_data common.DataInformation
	if err := c.BindJSON(&received_data); err != nil {
		return
	}

	common.DataLocation = append(common.DataLocation, received_data)
	c.Status(http.StatusOK)
}

func FetchData(c *gin.Context) {
	time.Sleep(DefaultSleep)
	requester := c.Query("requester")
	filename := c.Query("filename")
	filetype := c.Query("filetype")

	for i := 0; i < len(common.OwnData); i++ { //loop through own data first
		if common.OwnData[i].Filename == filename && common.OwnData[i].Type == filetype {
			payload := map[string]interface{}{
				"filename": common.OwnData[i].Filename,
				"type":     common.OwnData[i].Type,
				"node":     common.MyName,
				"size":     common.OwnData[i].Size,
			}
			c.JSON(http.StatusOK, payload)
			fmt.Printf("\n offloaded for %s\n", requester)
			return
		}
	}

	for i := 0; i < len(common.OwnCache); i++ { //loop through cache
		if common.OwnCache[i].Filename == filename && common.OwnCache[i].Type == filetype {
			payload := map[string]interface{}{
				"filename": common.OwnData[i].Filename,
				"type":     common.OwnData[i].Type,
				"node":     common.MyName,
				"size":     common.OwnData[i].Size,
			}
			c.JSON(http.StatusOK, payload)
			fmt.Printf("\n offloaded for %s\n", requester)
			common.OwnCache[i].Popularity++
			return
		}
	}

	c.Data(http.StatusNotFound, "text/plain", []byte("the data you requested does not exist")) //data does not exist
}

func FetchMetadata(c *gin.Context) {
	time.Sleep(DefaultSleep)
	c.JSON(http.StatusOK, gin.H{"metadata": common.DataLocation}) //return metadata

}

func FetchCachedata(c *gin.Context) {
	time.Sleep(DefaultSleep)
	c.JSON(http.StatusOK, gin.H{"cachedata": common.OwnCache}) //return metadata
}

func GetFreeCacheMemory(c *gin.Context) {
	time.Sleep(DefaultSleep)
	free_memory := common.TotalCacheMemory - common.UsedCacheMemory
	c.JSON(http.StatusOK, gin.H{"free": free_memory}) //return leftover memory
}

func CacheData(c *gin.Context) {
	time.Sleep(DefaultSleep)
	var received_data common.DataSpecification
	if err := c.BindJSON(&received_data); err != nil {
		return
	}
	received_data.Node = common.MyName
	received_data.Popularity = 1
	common.OwnCache = append(common.OwnCache, received_data) //get data and cache in lieu of another node
	common.UsedCacheMemory += received_data.Size
	c.Status(http.StatusOK)
}

func ReplaceData(c *gin.Context) {
	time.Sleep(DefaultSleep)
	type response_struct struct {
		Replacefilename string `json:"replacefilename"`
		Filename        string `json:"filename"`
		Type            string `json:"type"`
		Size            int    `json:"size"`
	}
	var received_data response_struct
	if err := c.BindJSON(&received_data); err != nil {
		return
	}
	for i := 0; i < len(common.OwnCache); i++ { //replace in cache
		if common.OwnCache[i].Filename == received_data.Replacefilename {
			common.UsedCacheMemory -= common.OwnCache[i].Size                //restore size
			common.DeleteCloud(received_data.Replacefilename, common.MyName) //delete from cloud metadata
			new_cache := common.DataSpecification{received_data.Filename, received_data.Type, received_data.Size, common.MyName, 1}
			common.OwnCache = append(common.OwnCache, new_cache)         //update own cache
			common.UsedCacheMemory += received_data.Size                 //update used memory slots
			common.OwnCache[i] = common.OwnCache[len(common.OwnCache)-1] //delete cached file
			common.OwnCache = common.OwnCache[:len(common.OwnCache)-1]
			break
		}
	}
	for i := 0; i < len(common.DataLocation); i++ {
		if common.DataLocation[i].Filename == received_data.Replacefilename {
			common.DataLocation[i] = common.DataLocation[len(common.DataLocation)-1] //remove from metadata
			common.DataLocation = common.DataLocation[:len(common.DataLocation)-1]
			new_entry := common.DataInformation{received_data.Filename, received_data.Type, common.MyName, true}
			common.DataLocation = append(common.DataLocation, new_entry)
			common.UpdateCloud(new_entry) //update cloud metadata
			break
		}
	}
	c.Status(http.StatusOK)
}

func DeleteData(c *gin.Context) {
	time.Sleep(DefaultSleep)
	type response_struct struct {
		Filename string `json:"filename"`
		Node     string `json:"node"`
	}
	var received_data response_struct
	if err := c.BindJSON(&received_data); err != nil {
		return
	}
	for i := 0; i < len(common.DataLocation); i++ {
		if common.DataLocation[i].Filename == received_data.Filename && common.DataLocation[i].Node == received_data.Node {
			common.DataLocation[i] = common.DataLocation[len(common.DataLocation)-1]
			common.DataLocation = common.DataLocation[:len(common.DataLocation)-1]
			c.Status(http.StatusOK)
			break
		}
	}
	c.Status(http.StatusNoContent)
}

func UpdateBusy(c *gin.Context) {
	var boolstatus bool
	node := c.Query("requester")
	status := c.Query("status")
	boolstatus, _ = strconv.ParseBool(status)

	if thisNode, ok := common.ConnectedNodes[node]; ok {
		thisNode.Busy = boolstatus
		common.ConnectedNodes[node] = thisNode
	}
	c.Status(http.StatusOK)
}
