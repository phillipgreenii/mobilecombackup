package analyzer

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
)

// GoCodeAnalyzer implements code analysis for Go source files
type GoCodeAnalyzer struct {
	fileSet *token.FileSet
	logger  Logger
}

// NewGoCodeAnalyzer creates a new Go code analyzer
func NewGoCodeAnalyzer(logger Logger) *GoCodeAnalyzer {
	return &GoCodeAnalyzer{
		fileSet: token.NewFileSet(),
		logger:  logger,
	}
}

// AnalyzePackage analyzes a Go package and extracts symbols
func (gca *GoCodeAnalyzer) AnalyzePackage(packagePath string) types.Result[[]CodeSymbol] {
	gca.logger.Debug("Analyzing package", "path", packagePath)

	// Parse package files
	pkgs, err := parser.ParseDir(gca.fileSet, packagePath, nil, parser.ParseComments)
	if err != nil {
		return types.NewResultError[[]CodeSymbol](fmt.Errorf("failed to parse package %s: %w", packagePath, err))
	}

	var allSymbols []CodeSymbol

	// Process each package
	for pkgName, pkg := range pkgs {
		// Create documentation
		docPkg := doc.New(pkg, packagePath, doc.AllDecls)

		// Extract symbols from the package
		symbols := gca.extractPackageSymbols(pkgName, docPkg, packagePath)
		allSymbols = append(allSymbols, symbols...)
	}

	return types.NewResult(allSymbols)
}

// ExtractSymbols extracts symbols from a Go source file
func (gca *GoCodeAnalyzer) ExtractSymbols(filePath string) types.Result[[]CodeSymbol] {
	gca.logger.Debug("Extracting symbols from file", "file", filePath)

	// Parse the file
	src, err := parser.ParseFile(gca.fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return types.NewResultError[[]CodeSymbol](fmt.Errorf("failed to parse file %s: %w", filePath, err))
	}

	var symbols []CodeSymbol
	packageName := src.Name.Name

	// Extract symbols from AST
	ast.Inspect(src, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			symbol := gca.extractFunctionSymbol(node, packageName, filePath)
			symbols = append(symbols, symbol)

		case *ast.GenDecl:
			// Handle type, const, var declarations
			for _, spec := range node.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					symbol := gca.extractTypeSymbol(s, node, packageName, filePath)
					symbols = append(symbols, symbol)

				case *ast.ValueSpec:
					// Constants and variables
					for _, name := range s.Names {
						symbol := gca.extractValueSymbol(name, s, node, packageName, filePath)
						symbols = append(symbols, symbol)
					}
				}
			}
		}
		return true
	})

	return types.NewResult(symbols)
}

// GetPackageDocs extracts package-level documentation
func (gca *GoCodeAnalyzer) GetPackageDocs(packagePath string) types.Result[string] {
	gca.logger.Debug("Getting package documentation", "path", packagePath)

	// Parse package files
	pkgs, err := parser.ParseDir(gca.fileSet, packagePath, nil, parser.ParseComments)
	if err != nil {
		return types.NewResultError[string](fmt.Errorf("failed to parse package %s: %w", packagePath, err))
	}

	// Get the first package's documentation
	for _, pkg := range pkgs {
		docPkg := doc.New(pkg, packagePath, doc.AllDecls)
		return types.NewResult(docPkg.Doc)
	}

	return types.NewResult("")
}

// FindReferences finds references to a symbol across the codebase
func (gca *GoCodeAnalyzer) FindReferences(symbol CodeSymbol) types.Result[[]CodeSymbol] {
	// This is a simplified implementation
	// A full implementation would need to analyze imports and cross-references
	gca.logger.Debug("Finding references for symbol", "symbol", symbol.Name)

	// For now, return empty slice
	// TODO: Implement proper reference finding
	return types.NewResult([]CodeSymbol{})
}

// Helper methods

func (gca *GoCodeAnalyzer) extractPackageSymbols(pkgName string, docPkg *doc.Package, packagePath string) []CodeSymbol {
	var symbols []CodeSymbol

	// Extract functions
	for _, fn := range docPkg.Funcs {
		symbol := CodeSymbol{
			Name:      fn.Name,
			Type:      "function",
			Package:   pkgName,
			File:      gca.getFileName(fn.Decl.Pos()),
			Line:      gca.getLineNumber(fn.Decl.Pos()),
			Signature: gca.getFunctionSignature(fn.Decl),
			Comment:   fn.Doc,
			Exported:  ast.IsExported(fn.Name),
			Metadata:  make(map[string]interface{}),
		}
		symbols = append(symbols, symbol)
	}

	// Extract types
	for _, tp := range docPkg.Types {
		symbol := CodeSymbol{
			Name:      tp.Name,
			Type:      "type",
			Package:   pkgName,
			File:      gca.getFileName(tp.Decl.Pos()),
			Line:      gca.getLineNumber(tp.Decl.Pos()),
			Signature: gca.getTypeSignature(tp.Decl),
			Comment:   tp.Doc,
			Exported:  ast.IsExported(tp.Name),
			Metadata:  make(map[string]interface{}),
		}
		symbols = append(symbols, symbol)

		// Extract methods
		for _, method := range tp.Methods {
			methodSymbol := CodeSymbol{
				Name:      fmt.Sprintf("%s.%s", tp.Name, method.Name),
				Type:      "method",
				Package:   pkgName,
				File:      gca.getFileName(method.Decl.Pos()),
				Line:      gca.getLineNumber(method.Decl.Pos()),
				Signature: gca.getFunctionSignature(method.Decl),
				Comment:   method.Doc,
				Exported:  ast.IsExported(method.Name),
				Metadata:  map[string]interface{}{"receiver": tp.Name},
			}
			symbols = append(symbols, methodSymbol)
		}
	}

	// Extract variables
	for _, variable := range docPkg.Vars {
		for _, name := range variable.Names {
			symbol := CodeSymbol{
				Name:     name,
				Type:     "variable",
				Package:  pkgName,
				File:     gca.getFileName(variable.Decl.Pos()),
				Line:     gca.getLineNumber(variable.Decl.Pos()),
				Comment:  variable.Doc,
				Exported: ast.IsExported(name),
				Metadata: make(map[string]interface{}),
			}
			symbols = append(symbols, symbol)
		}
	}

	// Extract constants
	for _, constant := range docPkg.Consts {
		for _, name := range constant.Names {
			symbol := CodeSymbol{
				Name:     name,
				Type:     "constant",
				Package:  pkgName,
				File:     gca.getFileName(constant.Decl.Pos()),
				Line:     gca.getLineNumber(constant.Decl.Pos()),
				Comment:  constant.Doc,
				Exported: ast.IsExported(name),
				Metadata: make(map[string]interface{}),
			}
			symbols = append(symbols, symbol)
		}
	}

	return symbols
}

func (gca *GoCodeAnalyzer) extractFunctionSymbol(fn *ast.FuncDecl, packageName, filePath string) CodeSymbol {
	comment := ""
	if fn.Doc != nil {
		comment = fn.Doc.Text()
	}

	symbol := CodeSymbol{
		Name:      fn.Name.Name,
		Type:      "function",
		Package:   packageName,
		File:      filePath,
		Line:      gca.getLineNumber(fn.Pos()),
		Signature: gca.getFunctionSignature(fn),
		Comment:   comment,
		Exported:  ast.IsExported(fn.Name.Name),
		Metadata:  make(map[string]interface{}),
	}

	// Check if it's a method
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		symbol.Type = "method"
		// Extract receiver type
		if recv := fn.Recv.List[0]; recv.Type != nil {
			if ident, ok := recv.Type.(*ast.Ident); ok {
				symbol.Metadata["receiver"] = ident.Name
				symbol.Name = fmt.Sprintf("%s.%s", ident.Name, fn.Name.Name)
			}
		}
	}

	return symbol
}

func (gca *GoCodeAnalyzer) extractTypeSymbol(typeSpec *ast.TypeSpec, genDecl *ast.GenDecl, packageName, filePath string) CodeSymbol {
	comment := ""
	if genDecl.Doc != nil {
		comment = genDecl.Doc.Text()
	} else if typeSpec.Doc != nil {
		comment = typeSpec.Doc.Text()
	}

	typeKind := "type"
	switch typeSpec.Type.(type) {
	case *ast.InterfaceType:
		typeKind = "interface"
	case *ast.StructType:
		typeKind = "struct"
	}

	return CodeSymbol{
		Name:      typeSpec.Name.Name,
		Type:      typeKind,
		Package:   packageName,
		File:      filePath,
		Line:      gca.getLineNumber(typeSpec.Pos()),
		Signature: gca.getTypeSignature(genDecl),
		Comment:   comment,
		Exported:  ast.IsExported(typeSpec.Name.Name),
		Metadata:  make(map[string]interface{}),
	}
}

func (gca *GoCodeAnalyzer) extractValueSymbol(name *ast.Ident, valueSpec *ast.ValueSpec, genDecl *ast.GenDecl, packageName, filePath string) CodeSymbol {
	comment := ""
	if genDecl.Doc != nil {
		comment = genDecl.Doc.Text()
	} else if valueSpec.Doc != nil {
		comment = valueSpec.Doc.Text()
	}

	symbolType := "variable"
	if genDecl.Tok == token.CONST {
		symbolType = "constant"
	}

	return CodeSymbol{
		Name:      name.Name,
		Type:      symbolType,
		Package:   packageName,
		File:      filePath,
		Line:      gca.getLineNumber(name.Pos()),
		Signature: gca.getValueSignature(valueSpec),
		Comment:   comment,
		Exported:  ast.IsExported(name.Name),
		Metadata:  make(map[string]interface{}),
	}
}

func (gca *GoCodeAnalyzer) getFunctionSignature(fn *ast.FuncDecl) string {
	var parts []string

	// Add function name
	parts = append(parts, "func")

	// Add receiver if method
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := fn.Recv.List[0]
		if recv.Names != nil && len(recv.Names) > 0 {
			parts = append(parts, fmt.Sprintf("(%s", recv.Names[0].Name))
		} else {
			parts = append(parts, "(")
		}
		parts = append(parts, gca.typeToString(recv.Type))
		parts = append(parts, ")")
	}

	parts = append(parts, fn.Name.Name)

	// Add parameters
	if fn.Type.Params != nil {
		params := gca.fieldListToString(fn.Type.Params)
		parts = append(parts, fmt.Sprintf("(%s)", params))
	} else {
		parts = append(parts, "()")
	}

	// Add return type
	if fn.Type.Results != nil {
		results := gca.fieldListToString(fn.Type.Results)
		if strings.Contains(results, ",") {
			parts = append(parts, fmt.Sprintf("(%s)", results))
		} else {
			parts = append(parts, results)
		}
	}

	return strings.Join(parts, " ")
}

func (gca *GoCodeAnalyzer) getTypeSignature(decl *ast.GenDecl) string {
	if len(decl.Specs) == 0 {
		return ""
	}

	typeSpec, ok := decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return ""
	}

	return fmt.Sprintf("type %s %s", typeSpec.Name.Name, gca.typeToString(typeSpec.Type))
}

func (gca *GoCodeAnalyzer) getValueSignature(valueSpec *ast.ValueSpec) string {
	var parts []string

	// Add names
	var names []string
	for _, name := range valueSpec.Names {
		names = append(names, name.Name)
	}
	parts = append(parts, strings.Join(names, ", "))

	// Add type if specified
	if valueSpec.Type != nil {
		parts = append(parts, gca.typeToString(valueSpec.Type))
	}

	return strings.Join(parts, " ")
}

func (gca *GoCodeAnalyzer) fieldListToString(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}

	var fields []string
	for _, field := range fl.List {
		var names []string
		for _, name := range field.Names {
			names = append(names, name.Name)
		}

		typeStr := gca.typeToString(field.Type)

		if len(names) > 0 {
			fields = append(fields, fmt.Sprintf("%s %s", strings.Join(names, ", "), typeStr))
		} else {
			fields = append(fields, typeStr)
		}
	}

	return strings.Join(fields, ", ")
}

func (gca *GoCodeAnalyzer) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + gca.typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + gca.typeToString(t.Elt)
		}
		return "[...]" + gca.typeToString(t.Elt)
	case *ast.SelectorExpr:
		return gca.typeToString(t.X) + "." + t.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{...}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", gca.typeToString(t.Key), gca.typeToString(t.Value))
	case *ast.ChanType:
		return "chan " + gca.typeToString(t.Value)
	default:
		return "unknown"
	}
}

func (gca *GoCodeAnalyzer) getFileName(pos token.Pos) string {
	position := gca.fileSet.Position(pos)
	return filepath.Base(position.Filename)
}

func (gca *GoCodeAnalyzer) getLineNumber(pos token.Pos) int {
	position := gca.fileSet.Position(pos)
	return position.Line
}
