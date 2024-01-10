package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	cuckoo "github.com/glim2485/cuckoofilter"
)

func main() {
	new_filter := cuckoo.NewFilter(100)
	fmt.Printf("testing")
	fmt.Println(new_filter)
	go server()

}

func server() {
	router := gin.Default()
	router.Run("0.0.0.0:8080")
}
