package showsql

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func CopyAndModifyFiles(
	srcDir, destDir, fileName string,
	callDataMap map[int]CallData,
	signatureMap map[string]SignatureData,
) error {
	err := os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "os.MkdirAll: failed to create destination directory")
	}

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "filepath.Walk")
		}

		if info.IsDir() && path != srcDir {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return errors.Wrap(err, "filepath.Rel")
		}

		destPath := filepath.Join(destDir, relPath)

		err = copyFile(path, destPath, fileName, callDataMap, signatureMap)

		return errors.Wrap(err, "copyFile")
	})

	return errors.Wrap(err, "filepath.Walk")
}

func copyFile(
	src, dest, fileName string,
	callDataMap map[int]CallData,
	signatureMap map[string]SignatureData,
) error {
	if !strings.HasSuffix(src, ".go") {
		return nil
	}

	file, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "os.Open")
	}
	defer file.Close()

	f, err := os.OpenFile(dest, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "os.OpenFile")
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	scanner := bufio.NewScanner(file)

	lineNumber := 0
	isPackage := true
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++

		if isPackage && strings.Contains(line, "package") && !strings.Contains(line, "/") {
			_, err := writer.WriteString("package main\n")
			if err != nil {
				return errors.Wrap(err, "writer.WriteString")
			}

			isPackage = false

			continue
		}

		callData, hasCallData := callDataMap[lineNumber]
		if src != fileName || !hasCallData {
			_, err := writer.WriteString(line + "\n")
			if err != nil {
				return errors.Wrap(err, "writer.WriteString")
			}

			continue
		}

		signatureData, hasSignatureData := signatureMap[callData.ParentMethodName]
		if !hasSignatureData {
			return fmt.Errorf("could not find signature data, parent method: %s", callData.ParentMethodName)
		}

		if callData.IsSqlizer {
			if err := addSqlizerPrintLines(writer, callData.SqlizerVarName); err != nil {
				return errors.Wrap(err, "addSqlizerPrintLines")
			}
		} else {
			if err := addSqlPrintLines(writer, callData.SqlVarName, callData.SqlArgsVarName); err != nil {
				return errors.Wrap(err, "addSqlPrintLines")
			}
		}

		if err := addEarlyReturnLines(writer, signatureData.Returns); err != nil {
			return errors.Wrap(err, "addEarlyReturnLines")
		}

		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return errors.Wrap(err, "writer.WriteString")
		}
	}

	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "writer.Flush")
	}

	err = scanner.Err()

	return errors.Wrap(err, "scanner.Err")
}

func addSqlizerPrintLines(writer *bufio.Writer, sqlizerVarName string) error {
	_, err := writer.WriteString(fmt.Sprintf(`showsqlSql, showsqlArgs, showsqlErr := %s.ToSql()`+"\n", sqlizerVarName))
	if err != nil {
		return errors.Wrap(err, "writer.WriteString")
	}

	_, err = writer.WriteString(`ShowsqlPrint(showsqlSql, showsqlArgs, showsqlErr)` + "\n")

	return errors.Wrap(err, "writer.WriteString")
}

func addSqlPrintLines(writer *bufio.Writer, sqlVarName, sqlArgsVarName string) error {
	_, err := writer.WriteString(fmt.Sprintf(`ShowsqlPrint(%s, %s, nil)`+"\n", sqlVarName, sqlArgsVarName))

	return errors.Wrap(err, "writer.WriteString")
}

func addEarlyReturnLines(writer *bufio.Writer, returns []string) error {
	args := make([]string, 0, len(returns))
	for i, val := range returns {
		_, err := writer.WriteString(fmt.Sprintf(`var showsqlArg%d %s`+"\n", i, val))
		if err != nil {
			return errors.Wrap(err, "writer.WriteString")
		}

		args = append(args, fmt.Sprintf("showsqlArg%d", i))
	}

	_, err := writer.WriteString(fmt.Sprintf(`return %s`+"\n", strings.Join(args, ", ")))

	return errors.Wrap(err, "writer.WriteString")
}
