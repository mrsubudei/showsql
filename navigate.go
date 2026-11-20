package showsql

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/eiannone/keyboard"
	"github.com/pkg/errors"
)

func Navigate() error {
	currentPath := "./"

	for {
		dirs, err := listDirectoriesAndFiles(currentPath)
		if err != nil {
			return errors.Wrap(err, "listDirectoriesAndFiles")
		}

		selected := 0

		for {
			clearScreen()

			for i, dir := range dirs {
				if i == selected {
					fmt.Printf("-> %s\n", dir)
				} else {
					fmt.Printf("   %s\n", dir)
				}
			}

			err = keyboard.Open()
			if err != nil {
				return errors.Wrap(err, "keyboard.Open")
			}

			defer func() {
				closeErr := keyboard.Close()
				if closeErr != nil {
					log.Fatal(err)
				}
			}()

			_, key, err := keyboard.GetKey()
			if err != nil {
				return errors.Wrap(err, "keyboard.GetKey")
			}

			switch key {
			case keyboard.KeyArrowUp:
				if selected > 0 {
					selected--
				}
			case keyboard.KeyArrowDown:
				if selected < len(dirs)-1 {
					selected++
				}
			case keyboard.KeyEnter:
				currentPath = filepath.Join(currentPath, dirs[selected])

			case keyboard.KeyEsc:
				return nil
			}

			if key == keyboard.KeyEnter {
				if !strings.HasSuffix(currentPath, ".go") {
					break
				}

				methods, err := getAllMethods(currentPath)
				if err != nil {
					return errors.Wrap(err, "getAllMethods")
				}

				err = chooseMethod(methods, currentPath)

				return errors.Wrap(err, "chooseMethod")
			}
		}
	}
}

func chooseMethod(methods []string, fileName string) error {
	selected := 0

	for {
		clearScreen()

		for i, dir := range methods {
			if i == selected {
				fmt.Printf("-> %s\n", dir)
			} else {
				fmt.Printf("   %s\n", dir)
			}
		}

		_, key, err := keyboard.GetKey()
		if err != nil {
			return errors.Wrap(err, "keyboard.GetKey")
		}

		switch key {
		case keyboard.KeyArrowUp:
			if selected > 0 {
				selected--
			}
		case keyboard.KeyArrowDown:
			if selected < len(methods)-1 {
				selected++
			}
		case keyboard.KeyEnter:
			clearScreen()

			method := methods[selected]
			fmt.Printf("-p=%s -m=%s\n\n", fileName, method)

			err = Handle(fileName, method, true)

			return errors.Wrap(err, "Handle")
		case keyboard.KeyEsc:
			return nil
		}
	}
}

func listDirectoriesAndFiles(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, errors.Wrap(err, "os.ReadDir")
	}

	dirs := []string{".."}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || (!entry.IsDir() && !strings.HasSuffix(entry.Name(), ".go")) {
			continue
		}

		dirs = append(dirs, entry.Name())
	}

	return dirs, nil
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func getAllMethods(fileName string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, fileName, nil, parser.AllErrors)
	if err != nil {
		return nil, errors.Wrap(err, "parser.ParseFile")
	}

	signatureDataMap := ExtractSignature(file)

	result := []string{}

	for key := range signatureDataMap {
		sl := strings.Split(key, "_")

		if len(sl) > 1 && len(sl[1]) > 0 && unicode.IsUpper(rune(sl[1][0])) {
			result = append(result, sl[1])
		}
	}

	return result, nil
}
