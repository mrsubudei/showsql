package main

import (
	"log"

	service "github.com/mrsubudei/showsql"
	"github.com/spf13/pflag"
)

func main() {
	var currentFilePath string
	pflag.StringVarP(&currentFilePath, "path", "p", "", "specify file path")

	var methodName string
	pflag.StringVarP(&methodName, "method", "m", "", "specify method name")

	var keepGenerated bool
	pflag.BoolVarP(&keepGenerated, "keep", "k", false, "keep generated files")

	pflag.Parse()

	if err := service.Handle(currentFilePath, methodName, keepGenerated); err != nil {
		log.Fatal(err)
	}
}
