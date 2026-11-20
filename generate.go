package service

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
			populated{{$i}} := Populate(tmp{{$i}})
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
)

func ShowsqlPrint(sql string, args []interface{}, err error) {
	fmt.Println()

    if err != nil {
		fmt.Println("err: ", err)
		fmt.Println()

		return
    }

	fmt.Println("sql: ", sql)

	fmt.Println()

	fmt.Println("args: ", args)

	fmt.Println()
}
`

func GeneratePopulatingFile(destDir string) error {
	f, err := os.OpenFile(destDir, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "os.OpenFile")
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	_, err = writer.WriteString(templatePopulatingFile)
	if err != nil {
		return errors.Wrap(err, "writer.WriteString")
	}
	err = writer.Flush()

	return errors.Wrap(err, "writer.Flush")
}

const templatePopulatingFile = `
package main

import (
	"math"
	"reflect"
	"strconv"
	"time"
)

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
		suffix++

		return val + strconv.Itoa(suffix)
	}
}

func GetNextTime() func() time.Time {
	val := time.Date(2025, 1, 0, 15, 0, 0, 0, time.UTC)

	return func() time.Time {
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

func Populate(input any) any {
	if input == nil {
		return nil
	}

	t := reflect.TypeOf(input)

	elem := reflect.New(t).Elem()

	isPointer := false
	if t.Kind() == reflect.Pointer {
		isPointer = true
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

		populated := populatePrimitive(t)
		if isPointer {
			ptr := reflect.New(t)
			ptr.Elem().Set(populated)

			return ptr.Convert(t).Interface()
		}

		return populated.Convert(t).Interface()
	case reflect.Struct:
		populated := populateStruct(t)

		if isPointer {
			ptr := reflect.New(t)
			ptr.Elem().Set(populated)

			return ptr.Convert(t).Interface()
		}

		return populated.Convert(t).Interface()
	case reflect.Slice:
		return populateSlice(elem).Interface()
	case reflect.Array:
		return populateArray(elem).Interface()
	case reflect.Map:
		return populateMap(elem).Interface()
	}

	return nil
}

func populateSlice(input reflect.Value) reflect.Value {
	if nesting >= maxNesting {
		return input
	}
	nesting++

	elemType := input.Type().Elem()
	slice := reflect.MakeSlice(input.Type(), 1, 1)
	isPointer := false

	if elemType.Kind() == reflect.Ptr {
		isPointer = true
		elemType = elemType.Elem()
	}

	elem := reflect.New(elemType).Elem()
	var populated reflect.Value

	switch elem.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

		populated = populatePrimitive(elem.Type())
	case reflect.Struct:
		populated = populateStruct(elem.Type())
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

	slice.Index(0).Set(elem)

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
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

			elem.Set(populatePrimitive(elemType).Convert(elemType))
		case reflect.Struct:
			elem.Set(populateStruct(elemType))
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
	isValPointer := false

	if valType.Kind() == reflect.Ptr {
		isValPointer = true
		valType = valType.Elem()
	}

	key := reflect.New(keyType).Elem()
	val := reflect.New(valType).Elem()

	switch key.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

		key.Set(populatePrimitive(keyType).Convert(keyType))
	case reflect.Struct:
		key.Set(populateStruct(keyType).Convert(keyType))
	case reflect.Array:
		key.Set(populateArray(key).Convert(keyType))
	}

	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

		val.Set(populatePrimitive(valType).Convert(valType))
	case reflect.Struct:
		val.Set(populateStruct(valType).Convert(valType))
	case reflect.Slice:
		val.Set(populateSlice(val).Convert(valType))
	case reflect.Array:
		val.Set(populateArray(val).Convert(valType))
	case reflect.Map:
		val.Set(populateMap(val).Convert(valType))
	}

	if isValPointer {
		ptr := reflect.New(valType)
		ptr.Elem().Set(val)
		mapInstance.SetMapIndex(key, ptr)
	} else {
		mapInstance.SetMapIndex(key, val)
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

		switch fieldValue.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.String, reflect.Bool, reflect.Chan, reflect.Func:

			fieldValue.Set(populatePrimitive(fieldValue.Type()))
		case reflect.Struct:
			if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
				fieldValue.Set(reflect.ValueOf(NextTime()))
			}
			fieldValue.Set(populateStruct(fieldValue.Type()))
		case reflect.Slice:
			fieldValue.Set(populateSlice(fieldValue))
		case reflect.Array:
			fieldValue.Set(populateArray(fieldValue))
		case reflect.Map:
			fieldValue.Set(populateMap(fieldValue))
		}
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
		value = true
	case reflect.Chan:
		value = reflect.MakeChan(t, 0).Interface()
	case reflect.Func:
		fn := reflect.MakeFunc(t, func(args []reflect.Value) (results []reflect.Value) {
			for i := 0; i < t.NumOut(); i++ {
				results = append(results, reflect.Zero(t.Out(i)))
			}

			return
		})

		value = fn.Interface()
	}

	return reflect.ValueOf(value)
}
`
