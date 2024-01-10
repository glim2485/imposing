package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	cuckoo "github.com/glim2485/cuckoofilter"
)

func main() {
	go server()
	new_filter := cuckoo.NewFilter(1000000)
	fmt.Println(new_filter)

}

func server() {
	router := gin.Default()
	router.Run("0.0.0.0:8080")
}
