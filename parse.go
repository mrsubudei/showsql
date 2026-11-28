package showsql

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type SignatureData struct {
	StructName string
	MethodName string
	Arguments  []string
	Returns    []string
}

func ExtractSignature(
	file *ast.File,
) map[string]SignatureData {
	var (
		result       = map[string]SignatureData{}
		structName   string
		nestingLevel int
		methodName   string
		args         []*ast.Field
		returns      []*ast.Field
	)

	ast.Inspect(file, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.FuncDecl:
			if expr.Recv == nil {
				return true
			}

			nestingLevel = 1
			if expr.Recv != nil && len(expr.Recv.List) > 0 {
				if starExpr, ok := expr.Recv.List[0].Type.(*ast.StarExpr); ok {
					structName = exprToString(starExpr.X)
				} else {
					structName = exprToString(expr.Recv.List[0].Type)
				}
			}

			methodName = expr.Name.Name
			if expr.Type.Params != nil {
				args = expr.Type.Params.List
			}
			if expr.Type.Results != nil {
				returns = expr.Type.Results.List
			}
		case *ast.FuncLit:
			nestingLevel++
			args = []*ast.Field{}
			if expr.Type.Params != nil {
				args = expr.Type.Params.List
			}
			returns = []*ast.Field{}
			if expr.Type.Results != nil {
				returns = expr.Type.Results.List
			}
		}

		data := SignatureData{}
		data.MethodName = methodName
		data.StructName = structName

		for _, param := range args {
			for range param.Names {
				data.Arguments = append(data.Arguments, exprToString(param.Type))
			}
		}

		for _, result := range returns {
			data.Returns = append(data.Returns, exprToString(result.Type))
		}

		result[structName+"_"+methodName+"_"+strconv.Itoa(nestingLevel)] = data

		return true
	})

	return result
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		lenStr := ""
		if lenExpr, ok := t.Len.(*ast.BasicLit); ok {
			lenStr = lenExpr.Value
		}

		return fmt.Sprintf("[%s]%s", lenStr, exprToString(t.Elt))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprToString(t.Key), exprToString(t.Value))
	case *ast.ChanType:
		chanDir := "chan "

		return chanDir + exprToString(t.Value)
	case *ast.FuncType:
		params := []string{}
		for _, param := range t.Params.List {
			params = append(params, exprToString(param.Type))
		}

		results := []string{}
		if t.Results != nil {
			for _, result := range t.Results.List {
				results = append(results, exprToString(result.Type))
			}
		}

		return fmt.Sprintf("func(%s) (%s)", strings.Join(params, ", "), strings.Join(results, ", "))
	case *ast.Ellipsis:
		return exprToString(t.Elt)
	default:
		return "unknown"
	}
}

func getImports(file *ast.File) []string {
	result := make([]string, 0, len(file.Imports))

	for _, imp := range file.Imports {
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}

		currImport := ""
		if alias != "" {
			currImport = fmt.Sprintf("%s %s", alias, imp.Path.Value)
		} else {
			currImport = imp.Path.Value
		}

		result = append(result, currImport)
	}

	return result
}

func getSquirrelImport(allImports []string) string {
	result := "squirrel"

	for _, imp := range allImports {
		if !strings.Contains(imp, "github.com/Masterminds/squirrel") {
			continue
		}

		sl := strings.Split(imp, " ")
		if len(sl) > 1 {
			result = sl[0]
		}

		break
	}

	return result
}

var caseFunc = func(key, name string) bool {
	_, ok := QueriesMap[key][name]

	return ok
}

var QueriesMap = map[string]map[string]struct{}{
	"Exec": {
		"Exec":     {},
		"Query":    {},
		"QueryRow": {},
	},
	"Get": {
		"Get":    {},
		"Select": {},
	},
	"Execx": {
		"Execx":     {},
		"Queryx":    {},
		"QueryxRow": {},
	},
	"Getx": {
		"Getx":    {},
		"Selectx": {},
	},
}

type CallData struct {
	ParentMethodName string
	IsSqlizer        bool
	SqlizerVarName   string
	SqlVarName       string
	SqlArgsVarName   string
	Pos              int
}

func GetCallData(
	node *ast.File,
	fset *token.FileSet,
	squirrelImport string,
) map[int]CallData {
	var (
		result       = make(map[int]CallData)
		currentFunc  string
		nestingLevel int
		funcEndLines = Stack{}
	)

	ast.Inspect(node, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.FuncDecl:
			position := fset.Position(expr.End())
			nestingLevel = 1
			funcEndLines = Stack{}
			funcEndLines.Add(position.Line)

			if expr.Recv != nil && len(expr.Recv.List) > 0 {
				receiverName := ""
				switch t := expr.Recv.List[0].Type.(type) {
				case *ast.Ident:
					receiverName = t.Name
				case *ast.StarExpr:
					if ident, ok := t.X.(*ast.Ident); ok {
						receiverName = ident.Name
					}
				}
				currentFunc = receiverName + "_" + expr.Name.Name + "_" + strconv.Itoa(nestingLevel)
			} else {
				currentFunc = expr.Name.Name + "_" + strconv.Itoa(nestingLevel)
			}
		case *ast.FuncLit:
			nestingLevel++
			position := fset.Position(expr.End())
			funcEndLines.Add(position.Line)
		case *ast.CallExpr:
			if fun, ok := expr.Fun.(*ast.SelectorExpr); ok {
				baseIdent := getBaseIdentifier(fun.X)

				if baseIdent != "" && baseIdent != squirrelImport {
					isSqlizer := false
					sqlizerVarName := ""
					sqlVarName := ""
					sqlArgsVarName := ""

					switch {
					case caseFunc("Exec", fun.Sel.Name):
						if name, ok := getArgumentName(expr, 1, fset); ok {
							sqlVarName = name
						}
						if name, ok := getArgumentName(expr, 2, fset); ok {
							sqlArgsVarName = strings.TrimRight(name, ".")
						}
					case caseFunc("Get", fun.Sel.Name):
						if name, ok := getArgumentName(expr, 2, fset); ok {
							sqlVarName = name
						}
						if name, ok := getArgumentName(expr, 3, fset); ok {
							sqlArgsVarName = strings.TrimRight(name, ".")
						}
					case caseFunc("Execx", fun.Sel.Name):
						isSqlizer = true
						if name, ok := getArgumentName(expr, 1, fset); ok {
							sqlizerVarName = name
						}
					case caseFunc("Getx", fun.Sel.Name):
						isSqlizer = true
						if name, ok := getArgumentName(expr, 2, fset); ok {
							sqlizerVarName = name
						}
					}

					if sqlVarName == "" && sqlizerVarName == "" {
						return true
					}

					if sqlArgsVarName == "" {
						sqlArgsVarName = "nil"
					}

					position := fset.Position(expr.Pos())
					currFuncEndLine := funcEndLines.Get()
					for position.Line > currFuncEndLine {
						nestingLevel--
						funcEndLines.Delete()
						currFuncEndLine = funcEndLines.Get()
					}

					sl := strings.Split(currentFunc, "_")
					currentFunc = strings.Join(sl[:len(sl)-1], "_") + "_" + strconv.Itoa(nestingLevel)

					result[position.Line] = CallData{
						ParentMethodName: currentFunc,
						IsSqlizer:        isSqlizer,
						SqlizerVarName:   sqlizerVarName,
						SqlVarName:       sqlVarName,
						SqlArgsVarName:   sqlArgsVarName,
						Pos:              position.Line,
					}
				}
			}
		}

		return true
	})

	return result
}

func getArgumentName(callExpr *ast.CallExpr, idx int, fset *token.FileSet) (string, bool) {
	if idx >= len(callExpr.Args) {
		return "", false
	}

	var buf bytes.Buffer
	err := printer.Fprint(&buf, fset, callExpr.Args[idx])
	if err != nil {
		return "", false
	}

	return buf.String(), true
}

func getBaseIdentifier(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return getBaseIdentifier(e.X)
	case *ast.CallExpr:
		return getBaseIdentifier(e.Fun)
	default:
		return ""
	}
}

type MutationQueries struct {
	Exec  []string `yaml:"exec"`
	Get   []string `yaml:"get"`
	Execx []string `yaml:"execx"`
	Getx  []string `yaml:"getx"`
}

func ParseMutationQueries(queriesFile string) {
	path := "scripts/showq.yaml"
	if queriesFile != "" {
		path = queriesFile
	}

	f, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var q MutationQueries

	if err := yaml.Unmarshal(f, &q); err != nil {
		return
	}

	if len(q.Exec) != 0 {
		QueriesMap["Exec"] = fillMap(q.Exec)
	}

	if len(q.Get) != 0 {
		QueriesMap["Get"] = fillMap(q.Get)
	}

	if len(q.Execx) != 0 {
		QueriesMap["Execx"] = fillMap(q.Execx)
	}

	if len(q.Getx) != 0 {
		QueriesMap["Getx"] = fillMap(q.Getx)
	}
}

func fillMap(names []string) map[string]struct{} {
	execMap := make(map[string]struct{}, 3)
	for _, v := range names {
		execMap[v] = struct{}{}
	}

	return execMap
}

type Stack struct {
	sl []int
}

func (s *Stack) Add(n int) {
	s.sl = append(s.sl, n)
}

func (s *Stack) Get() int {
	if len(s.sl) > 0 {
		return s.sl[len(s.sl)-1]
	}

	return 0
}

func (s *Stack) Delete() {
	if len(s.sl) <= 1 {
		return
	}

	s.sl = s.sl[:len(s.sl)-1]
}
