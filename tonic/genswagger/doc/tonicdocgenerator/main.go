package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/loopfz/gadgeto/tonic/genswagger/doc"
)

/*
	Run this script to generate swagger schema + sdks
*/

func main() {

	godoc := doc.GenerateDoc()
	b, err := json.MarshalIndent(godoc, "", "    ")
	if err != nil {
		panic(err)
	}
	godocStr := strings.Replace(string(b), "`", "'", -1)
	fmt.Println(godocStr)

}
