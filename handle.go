package showsql

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

type Flags struct {
	KeepGenerated bool
	AllIntsZero   bool
	AllStrsEmpty  bool
	AllboolsFalse bool
	AllTimeEmpty  bool
	ElemsCount    int
}

func Handle(
	fileName, callMethodName string,
	flags Flags,
) error {
	if !strings.HasSuffix(fileName, ".go") {
		return errors.New("this is not .go file")
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, fileName, nil, parser.AllErrors)
	if err != nil {
		return errors.Wrap(err, "parser.ParseFile")
	}

	allImports := getImports(file)
	squirrelImport := getSquirrelImport(allImports)

	callDataMap := GetCallData(file, fset, squirrelImport)
	signatureDataMap := ExtractSignature(file)

	destDir := "cmd/showsql_tmp"
	deleteTmpFolder(destDir)

	srcDir := cutFileName(fileName)
	if err := CopyAndModifyFiles(srcDir, destDir, fileName, callDataMap, signatureDataMap); err != nil {
		return errors.Wrap(err, "CopyAndModifyFiles")
	}

	if !flags.KeepGenerated {
		defer deleteTmpFolder(destDir)
	}

	structName := getStructName(signatureDataMap)
	signatureData, hasSignatureData := signatureDataMap[structName+"_"+callMethodName]
	if !hasSignatureData {
		return fmt.Errorf("there is no %s method", callMethodName)
	}

	generateData := GenerateData{
		StructName:     structName,
		CallMethodName: callMethodName,
		Arguments:      signatureData.Arguments,
		Imports:        allImports,
	}

	if err := GeneratePopulatingFile(flags, destDir+"/showsql_populate.go"); err != nil {
		return errors.Wrap(err, "GeneratePopulatingFile")
	}

	if err := GeneratePrintingFile(destDir + "/showsql_print.go"); err != nil {
		return errors.Wrap(err, "GeneratePrintingFile")
	}

	mainFilePath := destDir + "/main.go"
	if err := GenerateMainFile(generateData, mainFilePath); err != nil {
		return errors.Wrap(err, "GenerateMainFile")
	}

	formattedData, err := imports.Process(mainFilePath, nil, nil)
	if err != nil {
		return errors.Wrap(err, "imports.Process")
	}

	err = os.WriteFile(mainFilePath, formattedData, 0644)
	if err != nil {
		return errors.Wrap(err, "os.WriteFile")
	}

	err = executeMain(destDir)

	return errors.Wrap(err, "executeMain")
}

func executeMain(destDir string) error {
	cmd := exec.Command("go", "run", "./"+destDir)
	output, err := cmd.CombinedOutput()
	fmt.Print(string(output))

	return errors.Wrap(err, "cmd.CombinedOutput")
}

func deleteTmpFolder(destDir string) error {
	cmd := exec.Command("rm", "-rf", "./"+destDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Print(string(output))

		return errors.Wrap(err, "cmd.CombinedOutput")
	}

	return nil
}

func cutFileName(fileName string) string {
	sl := strings.Split(fileName, "/")

	return string(strings.Join(sl[:len(sl)-1], "/"))
}

func getStructName(signatureDataMap map[string]SignatureData) string {
	keysMap := make(map[string]int, len(signatureDataMap))

	for _, data := range signatureDataMap {
		keysMap[data.StructName]++
	}

	maxCount := 0
	parentFunc := ""

	for structName, count := range keysMap {
		if count > maxCount {
			maxCount = count
			parentFunc = structName
		}
	}

	sl := strings.Split(parentFunc, "_")

	return sl[0]
}

func CheckGoModExists() bool {
	_, err := os.Stat("go.mod")

	return err == nil || !os.IsNotExist(err)
}
