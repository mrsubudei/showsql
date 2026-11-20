package showsql

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
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
	result := map[string]SignatureData{}

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}

		data := SignatureData{}

		var structName string
		if len(funcDecl.Recv.List) > 0 {
			if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
				structName = exprToString(starExpr.X)
			} else {
				structName = exprToString(funcDecl.Recv.List[0].Type)
			}
		}

		data.MethodName = funcDecl.Name.Name
		data.StructName = structName

		for _, param := range funcDecl.Type.Params.List {
			for range param.Names {
				data.Arguments = append(data.Arguments, exprToString(param.Type))
			}
		}

		if funcDecl.Type.Results != nil {
			for _, result := range funcDecl.Type.Results.List {
				data.Returns = append(data.Returns, exprToString(result.Type))
			}
		}

		result[structName+"_"+funcDecl.Name.Name] = data
	}

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
	result := make(map[int]CallData)
	var currentFunc string

	ast.Inspect(node, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.FuncDecl:
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
				currentFunc = receiverName + "_" + expr.Name.Name
			} else {
				currentFunc = expr.Name.Name
			}
		case *ast.CallExpr:
			if fun, ok := expr.Fun.(*ast.SelectorExpr); ok {
				baseIdent := getBaseIdentifier(fun.X)

				if baseIdent != "" && baseIdent != squirrelImport {
					position := fset.Position(expr.Pos())
					isSqlizer := false
					sqlizerVarName := ""
					sqlVarName := ""
					sqlArgsVarName := ""

					switch fun.Sel.Name {
					case "Exec", "Query", "QueryRow":
						if name, ok := getArgumentName(expr, 1, fset); ok {
							sqlVarName = name
						}
						if name, ok := getArgumentName(expr, 2, fset); ok {
							sqlArgsVarName = strings.TrimRight(name, ".")
						}
					case "Get", "Select":
						if name, ok := getArgumentName(expr, 2, fset); ok {
							sqlVarName = name
						}
						if name, ok := getArgumentName(expr, 3, fset); ok {
							sqlArgsVarName = strings.TrimRight(name, ".")
						}
					case "Execx", "Queryx", "QueryxRow":
						isSqlizer = true
						if name, ok := getArgumentName(expr, 1, fset); ok {
							sqlizerVarName = name
						}
					case "Getx", "Selectx":
						isSqlizer = true
						if name, ok := getArgumentName(expr, 2, fset); ok {
							sqlizerVarName = name
						}
					}

					if sqlVarName == "" && sqlizerVarName == "" {
						return true
					}

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
