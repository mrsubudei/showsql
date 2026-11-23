package showsql

import (
	"bufio"
	"os"
	"text/template"

	"github.com/pkg/errors"
)

const templateMainFile = `
package main

import (
	{{- range .Imports }}
    	{{.}}
	{{- end }}
)

func main(){
	impl := {{.StructName}}{}

	{{- range $i, $argType := .Arguments }}
		{{- if eq $argType "context.Context" }}
			arg{{$i}} := context.Background()
		{{- else }}
    		var tmp{{$i}} {{$argType}}
			populated{{$i}} := ShowsqlPopulate(tmp{{$i}})
			arg{{$i}}, _ := populated{{$i}}.({{$argType}})
		{{- end }}
	{{- end }}

	impl.{{.CallMethodName}}(
	{{- range $i, $argType := .Arguments }}
    	arg{{$i}},
	{{- end }}
	)	
}
`

type GenerateData struct {
	StructName     string
	CallMethodName string
	Arguments      []string
	Imports        []string
}

func GenerateMainFile(data GenerateData, destDir string) error {
	tmpl, err := template.New("generated").Parse(templateMainFile)
	if err != nil {
		return errors.Wrap(err, "template.New.Parse")
	}

	outFile, err := os.Create(destDir)
	if err != nil {
		return errors.Wrap(err, "os.Create")
	}
	defer outFile.Close()

	err = tmpl.Execute(outFile, data)

	return errors.Wrap(err, "tmpl.Execute")
}

func GeneratePrintingFile(destDir string) error {
	f, err := os.OpenFile(destDir, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "os.OpenFile")
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	_, err = writer.WriteString(templatePrintingFile)
	if err != nil {
		return errors.Wrap(err, "writer.WriteString")
	}
	err = writer.Flush()

	return errors.Wrap(err, "writer.Flush")
}

const templatePrintingFile = `
package main

import (
	"fmt"
	"strings"
	"regexp"
)

func ShowsqlPrint(sql string, args []interface{}, err error) {
    fmt.Println()
	if err != nil {
		fmt.Println("err: ", err)
		fmt.Println()

		return
    }

	removed := showsqlRemoveExtraSpaces(sql)
	prettied := showsqlMakePretty(removed)

	fmt.Println("sql: ", prettied)
	fmt.Println()

	fmt.Println("args: ", args)
	fmt.Println()
}

func showsqlRemoveExtraSpaces(str string) string {
	str = strings.ReplaceAll(str, "\t", " ")
	str = strings.ReplaceAll(str, "\n", " ")

	spaceRegex := regexp.MustCompile("\\s+")
	str = spaceRegex.ReplaceAllString(str, " ")

	return strings.TrimSpace(str)
}

func showsqlMakePretty(str string) string {
	keyWordsMap := map[string]struct{}{
		"SELECT":    {},
		"INSERT":    {},
		"DELETE":    {},
		"FROM":      {},
		"WHERE":     {},
		"GROUP":     {},
		"HAVING":    {},
		"ORDER":     {},
		"LIMIT":     {},
		"OFFSET":    {},
		"RETURNING": {},
		"BEGIN":     {},
		"COMMIT":    {},
		"ROLLBACK":  {},
		"FOR":       {},
		"OVER":      {},
		"WITH":      {},
		"SET":       {},
		"USING":     {},
		"JOIN":      {},
		"LEFT":      {},
		"RIGHT":     {},
		"INNER":     {},
		"FULL":      {},
		"CROSS":     {},
		"NATURAL":   {},
		"CASE":      {},
		"WHEN":      {},
		"THEN":      {},
		"ELSE":      {},
		"END":       {},
		"UNION":     {},
		"INTERSECT": {},
		"EXCEPT":    {},
		"DO":        {},
		"SHOW":      {},
		"CREATE":    {},
		"ALTER":     {},
		"DROP":      {},
		"TRUNCATE":  {},
	}

	var sb strings.Builder
	sl := strings.Split(str, " ")

	nestingLevel := 0
	for _, val := range sl {
		if _, ok := keyWordsMap[strings.ToUpper(val)]; ok {
			sb.WriteString("\n")
			sb.WriteString(addTabs(nestingLevel))
		}
		if val == ")" || val == ")," {
			sb.WriteString("\n")
			nestingLevel--
		}

		sb.WriteString(val + " ")

		if val == "(" {
			nestingLevel++
		}

	}

	return sb.String()
}

func addTabs(nestingLevel int) string {
	result := ""

	for range nestingLevel {
		result += "\t"
	}

	return result
}
`

func GeneratePopulatingFile(flags Flags, destDir string) error {
	tmpl, err := template.New("generated").Parse(templatePopulatingFile)
	if err != nil {
		return errors.Wrap(err, "template.New.Parse")
	}

	outFile, err := os.Create(destDir)
	if err != nil {
		return errors.Wrap(err, "os.Create")
	}
	defer outFile.Close()

	err = tmpl.Execute(outFile, flags)

	return errors.Wrap(err, "tmpl.Execute")
}

const templatePopulatingFile = `
package main

import (
	"math"
	"reflect"
	"strconv"
	"time"
)

const (
	allIntsZero     = {{.AllIntsZero}}
	allStringsEmpty = {{.AllStrsEmpty}}
	allBoolsFalse   = {{.AllboolsFalse}}
	allTimeEmpty    = {{.AllTimeEmpty}}
)

const elemsCount = {{.ElemsCount}}

func isPrime(n int) bool {
	if n <= 1 {
		return false
	}

	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}

	return true
}

func GetNextInt() func() int {
	val := 5

	return func() int {
		if allIntsZero {
			return 0
		}

		val++

		for i := val; i < math.MaxInt32; i++ {
			if isPrime(i) {
				val = i

				break
			}
		}

		return val
	}
}

func GetNextString() func() string {
	val := "str"
	suffix := 0

	return func() string {
		if allStringsEmpty {
			return ""
		}

		suffix++

		return val + strconv.Itoa(suffix)
	}
}

func GetNextTime() func() time.Time {
	val := time.Date(2025, 1, 0, 15, 0, 0, 0, time.UTC)

	return func() time.Time {
		if allTimeEmpty {
			return time.Time{}
		}

		val = val.Add(time.Hour * 24)

		return val
	}
}

var (
	NextInt  = GetNextInt()
	NextStr  = GetNextString()
	NextTime = GetNextTime()
)

var nesting int

const maxNesting = 10000

func ShowsqlPopulate(input any) any {
	if input == nil {
		return nil
	}

	t := reflect.TypeOf(input)

	isPointer := false
	if t.Kind() == reflect.Pointer {
		isPointer = true
		t = t.Elem()
	}

	elem := reflect.New(t).Elem()

	switch t.Kind() {
	case
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

		populated := populatePrimitive(t)

		return getValuePointerType(t, populated, isPointer)
	case reflect.Struct:
		populated := populateCommonStruct(t)

		return getValuePointerType(t, populated, isPointer)
	case reflect.Slice:
		populated := populateSlice(elem)

		return getValuePointerType(t, populated, isPointer)
	case reflect.Array:
		populated := populateArray(elem)

		return getValuePointerType(t, populated, isPointer)
	case reflect.Map:
		populated := populateMap(elem)

		return getValuePointerType(t, populated, isPointer)
	}

	return nil
}

func populateSlice(input reflect.Value) reflect.Value {
	if nesting >= maxNesting {
		return input
	}
	nesting++

	elemType := input.Type().Elem()
	slice := reflect.MakeSlice(input.Type(), elemsCount, elemsCount)
	isPointer := false

	if elemType.Kind() == reflect.Ptr {
		isPointer = true
		elemType = elemType.Elem()
	}

	for i := 0; i < elemsCount; i++ {
		elem := reflect.New(elemType).Elem()
		var populated reflect.Value

		switch elem.Kind() {
		case
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

			populated = populatePrimitive(elem.Type())
		case reflect.Struct:
			populated = populateCommonStruct(elem.Type())
		case reflect.Slice:
			populated = populateSlice(elem)
		case reflect.Array:
			populated = populateArray(elem)
		case reflect.Map:
			populated = populateMap(elem)
		}

		elem.Set(populated.Convert(elem.Type()))

		if isPointer {
			ptr := reflect.New(elemType)
			ptr.Elem().Set(elem)
			elem = ptr
		}

		slice.Index(i).Set(elem)
	}

	return slice
}

func populateArray(input reflect.Value) reflect.Value {
	if nesting >= maxNesting {
		return input
	}
	nesting++

	elemType := input.Type().Elem()
	arrayType := reflect.ArrayOf(input.Len(), elemType)
	array := reflect.New(arrayType).Elem()
	isPointer := false

	if elemType.Kind() == reflect.Ptr {
		isPointer = true
		elemType = elemType.Elem()
	}

	for i := 0; i < input.Len(); i++ {
		elem := reflect.New(elemType).Elem()

		switch elem.Kind() {
		case
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

			elem.Set(populatePrimitive(elemType).Convert(elemType))
		case reflect.Struct:
			elem.Set(populateCommonStruct(elemType))
		case reflect.Slice:
			elem.Set(populateSlice(elem).Convert(elemType))
		case reflect.Array:
			elem.Set(populateArray(elem).Convert(elemType))
		case reflect.Map:
			elem.Set(populateMap(elem).Convert(elemType))
		}

		if isPointer {
			ptr := reflect.New(elemType)
			ptr.Elem().Set(elem)
			elem = ptr
		}

		array.Index(i).Set(elem)
	}

	return array
}

func populateMap(input reflect.Value) reflect.Value {
	if nesting >= maxNesting {
		return input
	}
	nesting++

	keyType := input.Type().Key()
	valType := input.Type().Elem()
	mapInstance := reflect.MakeMap(input.Type())

	isKeyPointer := false
	isValPointer := false

	if keyType.Kind() == reflect.Ptr {
		isKeyPointer = true
		keyType = keyType.Elem()
	}

	if valType.Kind() == reflect.Ptr {
		isValPointer = true
		valType = valType.Elem()
	}

	for i := 0; i < elemsCount; i++ {
		key := reflect.New(keyType).Elem()
		var populatedKey reflect.Value

		switch key.Kind() {
		case
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

			populatedKey = populatePrimitive(key.Type())
		case reflect.Struct:
			populatedKey = populateCommonStruct(key.Type())
		case reflect.Array:
			populatedKey = populateArray(key)
		}

		tmpKey := populatedKey
		populatedKey = getValuePointer(key.Type(), tmpKey, isKeyPointer)

		val := reflect.New(valType).Elem()
		var populatedVal reflect.Value

		switch val.Kind() {
		case
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

			populatedVal = populatePrimitive(val.Type())
		case reflect.Struct:
			populatedVal = populateCommonStruct(val.Type())
		case reflect.Slice:
			populatedVal = populateSlice(val)
		case reflect.Array:
			populatedVal = populateArray(val)
		case reflect.Map:
			populatedVal = populateMap(val)
		}

		tmpVal := populatedVal
		populatedVal = getValuePointer(val.Type(), tmpVal, isValPointer)

		mapInstance.SetMapIndex(populatedKey, populatedVal)
	}

	return mapInstance
}

func populateStruct(t reflect.Type) reflect.Value {
	newInstance := reflect.New(t).Elem()

	if nesting >= maxNesting {
		return newInstance
	}
	nesting++

	for i := 0; i < t.NumField(); i++ {
		fieldValue := newInstance.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		fieldType := fieldValue.Type()
		isPointer := false
		if fieldType.Kind() == reflect.Pointer {
			isPointer = true
			fieldType = fieldType.Elem()
		}

		field := reflect.New(fieldType).Elem()
		var populated reflect.Value

		switch field.Kind() {
		case
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

			populated = populatePrimitive(field.Type())
		case reflect.Struct:
			populated = populateCommonStruct(field.Type())
		case reflect.Slice:
			populated = populateSlice(field)
		case reflect.Array:
			populated = populateArray(field)
		case reflect.Map:
			populated = populateMap(field)
		}

		tmp := populated
		populated = getValuePointer(field.Type(), tmp, isPointer)
		fieldValue.Set(populated)
	}

	return newInstance
}

func populatePrimitive(t reflect.Type) reflect.Value {
	var value any
	switch t.Kind() {
	case reflect.Int:
		value = NextInt()
	case reflect.Int8:
		value = int8(NextInt())
	case reflect.Int16:
		value = int16(NextInt())
	case reflect.Int32:
		value = int32(NextInt())
	case reflect.Int64:
		value = int64(NextInt())
	case reflect.Uint:
		value = uint(NextInt())
	case reflect.Uint8:
		value = uint8(NextInt())
	case reflect.Uint16:
		value = uint16(NextInt())
	case reflect.Uint32:
		value = uint32(NextInt())
	case reflect.Uint64:
		value = uint64(NextInt())
	case reflect.Float32:
		value = float32(NextInt())
	case reflect.Float64:
		value = float64(NextInt())
	case reflect.String:
		value = NextStr()
	case reflect.Bool:
		if allBoolsFalse {
			value = false
		} else {
			value = true
		}
	case reflect.Chan:
		value = reflect.MakeChan(t, 0).Interface()
	case reflect.Func:
		fn := reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
			for i := 0; i < t.NumOut(); i++ {
				outType := t.Out(i)

				isPointer := false
				if outType.Kind() == reflect.Pointer {
					isPointer = true
					outType = outType.Elem()
				}

				outValue := reflect.New(outType).Elem()
				var populated reflect.Value

				switch outValue.Kind() {
				case
					reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
					reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
					reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

					populated = populatePrimitive(outValue.Type())
				case reflect.Struct:
					populated = populateCommonStruct(outValue.Type())
				case reflect.Slice:
					populated = populateSlice(outValue)
				case reflect.Array:
					populated = populateArray(outValue)
				case reflect.Map:
					populated = populateMap(outValue)
				default:
					results = append(results, reflect.Zero(t.Out(i)))

					continue
				}

				tmp := populated
				populated = getValuePointer(outType, tmp, isPointer)
				results = append(results, populated)
			}

			return
		})

		value = fn.Interface()
	}

	return reflect.ValueOf(value)
}

func populateCommonStruct(t reflect.Type) reflect.Value {
	if t == reflect.TypeOf(time.Time{}) {
		return reflect.ValueOf(NextTime())
	}

	return populateStruct(t)
}

func getValuePointerType(t reflect.Type, populated reflect.Value, isPointer bool) any {
	if isPointer {
		ptr := reflect.New(t)
		ptr.Elem().Set(populated)

		return ptr.Interface()
	}

	return populated.Convert(t).Interface()
}

func getValuePointer(t reflect.Type, populated reflect.Value, isPointer bool) reflect.Value {
	if isPointer {
		ptr := reflect.New(t)
		ptr.Elem().Set(populated)

		return ptr
	}

	return populated.Convert(t)
}
`
