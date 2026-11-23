package showsql

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/eiannone/keyboard"
	"github.com/pkg/errors"
)

func Navigate(flags Flags) error {
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
				selected--
				if selected == -1 {
					selected = len(dirs) - 1
				}
			case keyboard.KeyArrowDown:
				selected++
				if selected >= len(dirs) {
					selected = 0
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

				isChosen, err := chooseMethod(methods, currentPath, flags)
				if err != nil {
					return errors.Wrap(err, "chooseMethod")
				}

				if !isChosen {
					sl := strings.Split(currentPath, "/")
					currentPath = strings.Join(sl[:len(sl)-1], "/")

					break
				}

				return nil
			}
		}
	}
}

func chooseMethod(methods []string, fileName string, flags Flags) (bool, error) {
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
			return false, errors.Wrap(err, "keyboard.GetKey")
		}

		switch key {
		case keyboard.KeyArrowUp:
			selected--
			if selected == -1 {
				selected = len(methods) - 1
			}
		case keyboard.KeyArrowDown:
			selected++
			if selected >= len(methods) {
				selected = 0
			}
		case keyboard.KeyEnter:
			if selected == 0 {
				return false, nil
			}

			clearScreen()

			method := methods[selected]
			fmt.Printf("-p=%s -m=%s\n\n", fileName, method)

			err = Handle(fileName, method, flags)

			return true, errors.Wrap(err, "Handle")
		case keyboard.KeyEsc:
			return true, nil
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

	result := []string{".."}
	methodsMap := make(map[string]struct{}, len(signatureDataMap))

	for key := range signatureDataMap {
		sl := strings.Split(key, "_")

		if len(sl) > 1 && len(sl[1]) > 0 && unicode.IsUpper(rune(sl[1][0])) {
			if _, ok := methodsMap[sl[1]]; ok {
				continue
			}

			methodsMap[sl[1]] = struct{}{}
			result = append(result, sl[1])
		}
	}

	sort.Strings(result)

	return result, nil
}
