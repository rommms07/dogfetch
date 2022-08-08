package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/rommms07/dogfetch"
)

var idParam = flag.String("id", "", "Get breed by id.")
var nameParam = flag.String("name", "", "Get breed by name. (ex: ./cmd -name \"Golden Retriever\")")
var allFlag = flag.Bool("all", false, "Get all dog breeds.")

func main() {
	flag.Parse()

	var res any

	if len(*idParam) != 0 {
		res = dogfetch.GetById(*idParam)
	} else if len(*nameParam) != 0 {
		res = dogfetch.GetByName(*nameParam)
	} else if *allFlag {
		res = dogfetch.GetAll()
	}

	P, err := json.Marshal(res)
	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Println(string(P))
}
