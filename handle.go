package service

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

func Handle(
	fileName, callMethodName string,
	keepGeneratedFiles bool,
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
	srcDir := cutFileName(fileName)
	if err := CopyAndModifyFiles(srcDir, destDir, fileName, callDataMap, signatureDataMap); err != nil {
		return errors.Wrap(err, "CopyAndModifyFiles")
	}

	if !keepGeneratedFiles {
		defer deleteTmpFolder(destDir)
	}

	structName := getStructName(signatureDataMap)
	signatureData := signatureDataMap[structName+"_"+callMethodName]

	generateData := GenerateData{
		StructName:     structName,
		CallMethodName: callMethodName,
		Arguments:      signatureData.Arguments,
		Imports:        allImports,
	}

	if err := GeneratePopulatingFile(destDir + "/showsql_populate.go"); err != nil {
		return errors.Wrap(err, "GeneratePopulatingFile")
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
	parentFunc := ""
	for k := range signatureDataMap {
		parentFunc = k

		break
	}

	sl := strings.Split(parentFunc, "_")

	return sl[0]
}
