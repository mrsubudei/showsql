package main

import (
	"fmt"
	"log"

	"github.com/mrsubudei/showsql"
	"github.com/spf13/pflag"
)

var (
	helpInfo = `Usage: showsql [OPTION]...

-h     --help           show help information
-p     --path=string    specify path to required file, works only with --method
-m     --method=string  specify required method, works only with --path
-k     --keep           keep generated files
-i     --integer        populate all integer variables with 0 value
-s     --string         populate all string variables with "" value
-b     --boolean        populate all boolean variables with false value
-t     --time           populate all time variables with zero value
-n     --number=int     specify the number of elements to populate slices, maps, arrays 
-f     --file=string    specify config file for mutation query names
`
)

func main() {
	var help bool
	pflag.BoolVarP(&help, "help", "h", false, "show help information")

	var currentFilePath string
	pflag.StringVarP(&currentFilePath, "path", "p", "", "specify file path")

	var methodName string
	pflag.StringVarP(&methodName, "method", "m", "", "specify method name")

	var keepGenerated bool
	pflag.BoolVarP(&keepGenerated, "keep", "k", false, "keep generated files")

	var allIntsZero bool
	pflag.BoolVarP(&allIntsZero, "integer", "i", false, "populate all integer variables with zero value")

	var allStrsEmpty bool
	pflag.BoolVarP(&allStrsEmpty, "string", "s", false, "populate all string variables with empty value")

	var allBoolsFalse bool
	pflag.BoolVarP(&allBoolsFalse, "bool", "b", false, "populate all boolean variables with false value")

	var allTimeEmpty bool
	pflag.BoolVarP(&allTimeEmpty, "time", "t", false, "populate all time variables with zero value")

	var elemsCount int
	pflag.IntVarP(&elemsCount, "number", "n", 1, "number of elements to populate slices, maps, arrays ")

	var queriesFile string
	pflag.StringVarP(&queriesFile, "file", "f", "", "specify config file for mutation query names")

	pflag.Parse()

	if help {
		fmt.Println(helpInfo)

		return
	}

	if !showsql.CheckGoModExists() {
		log.Fatal("go.mod file not found")
	}

	if (currentFilePath == "" && methodName != "") || (currentFilePath != "" && methodName == "") {
		log.Fatal("--path and --method should both be filled or empty")
	}

	if elemsCount > 10 {
		log.Fatal("--number should not exceed 10")
	}

	flags := showsql.Flags{
		KeepGenerated: keepGenerated,
		AllIntsZero:   allIntsZero,
		AllStrsEmpty:  allStrsEmpty,
		AllboolsFalse: allBoolsFalse,
		AllTimeEmpty:  allTimeEmpty,
		ElemsCount:    elemsCount,
		QueriesFile:   queriesFile,
	}

	if currentFilePath != "" && methodName != "" {
		if err := showsql.Handle(currentFilePath, methodName, flags); err != nil {
			log.Fatal(err)
		}

		return
	}

	errNav := showsql.Navigate(flags)
	if errNav != nil {
		log.Fatal(errNav)
	}
}
