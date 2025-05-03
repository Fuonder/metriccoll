/*
Package noosexit предоставляет анализатор для инструмента go/analysis,
который проверяет наличие вызовов os.Exit в функции main пакета main.

# Назначение

Анализатор проходит по AST (abstract syntax tree) всех файлов в пакете и ищет функцию:

	func main()

в пакете `main`. Если внутри этой функции находится вызов `os.Exit`, то анализатор
регистрирует диагностическое сообщение с предупреждением.

# Использование

Для интеграции с go vet или как часть кастомного анализа:

	import "path/to/noosexit"

	analyses := []*analysis.Analyzer{
	    noosexit.Analyzer,
	}

Можно также использовать с инструментами на основе golang.org/x/tools/go/analysis.

# Пример диагностируемого кода

	package main

	import "os"

	func main() {
	    os.Exit(1) // <-- этот вызов будет обнаружен анализатором
	}
*/
package noosexit

import (
	"go/ast"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

// Analyzer — это анализатор go/analysis, который ищет вызовы os.Exit
// в функции main основного пакета.
//
// Возвращает предупреждение, если os.Exit вызывается напрямую в main.main.
var Analyzer = &analysis.Analyzer{
	Name:     "noosexit",
	Doc:      "Check for os.Exit calls in pkg main, function main",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// run — это основная функция анализа, вызываемая фреймворком go/analysis.
//
// Она выполняет следующие шаги:
//  1. Проверяет, является ли анализируемый пакет "main".
//  2. Находит функцию с именем "main" без receiver'а.
//  3. Выполняет обход AST тела этой функции и ищет вызовы os.Exit.
//  4. Если вызов найден, добавляет диагностическое сообщение.
//
// Принимаемые параметры:
//   - pass (*analysis.Pass): структура с информацией о текущем проходе анализа.
//
// Возвращаемые значения:
//   - interface{}: nil, т.к. анализатор не возвращает конкретные данные.
//   - error: ошибка выполнения анализа (обычно nil).
func run(pass *analysis.Pass) (interface{}, error) {
	// fmt.Printf("Analyzing package: %s\n", pass.Pkg.Name())

	// Проверяем, что анализируется именно пакет "main"
	if pass.Pkg.Name() != "main" {
		// fmt.Printf("Skipping package %s (not 'main')\n", pass.Pkg.Name())
		return nil, nil
	}

	// Множество для отслеживания уже проанализированных функций, чтобы избежать повторной проверки
	analyzedFuncs := make(map[string]bool)

	// Проходим по всем файлам пакета
	for _, file := range pass.Files {
		// fmt.Printf("Analyzing file: %s\n", file.Name.Name)

		// Перебираем все объявления в файле
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)

			// Ищем функцию с именем "main", без receiver'а и с телом
			if !ok || fn.Name.Name != "main" || fn.Recv != nil || fn.Body == nil {
				// fmt.Printf("Skipping non-main function or invalid function\n")
				continue
			}

			// fmt.Printf("Found function: %s\n", fn.Name.Name)

			// Пропускаем уже проанализированную функцию
			if analyzedFuncs[fn.Name.Name] {
				// fmt.Printf("Function 'main' already analyzed, skipping\n")
				continue
			}

			// Отмечаем функцию как проанализированную
			analyzedFuncs[fn.Name.Name] = true
			// fmt.Printf("Analyzing main function...\n")

			// Обход тела функции main для поиска вызовов os.Exit
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				// Ищем вызовы функций
				callExpr, ok := n.(*ast.CallExpr)
				if !ok {
					// fmt.Printf("Not a call expression, skipping\n")
					return true
				}

				// Проверяем, что вызов — это селектор (например, os.Exit)
				sel, ok := callExpr.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Exit" {
					// fmt.Printf("Not an Exit call, skipping\n")
					return true
				}

				// Проверяем, что вызывается объект с идентификатором (например, "os")
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					// fmt.Printf("Not an identifier, skipping\n")
					return true
				}

				// fmt.Printf("Found call to: %s\n", ident.Name)

				// Получаем объект, связанный с идентификатором
				obj := pass.TypesInfo.ObjectOf(ident)
				if obj == nil {
					// fmt.Printf("Object of identifier is nil, skipping\n")
					return true
				}

				// Проверяем, является ли объект импортированным пакетом
				pkgName, ok := obj.(*types.PkgName)
				if !ok || pkgName == nil {
					// fmt.Printf("Object is not *types.PkgName or is nil\n")
					return true
				}
				// Получаем информацию об импортированном пакете
				importedPkg := pkgName.Imported()
				if importedPkg == nil {
					// fmt.Printf("pkgName.Imported() is nil, skipping\n")
					return true
				}

				// Проверяем, что вызывается os.Exit
				if importedPkg.Path() == "os" {
					// fmt.Printf("Direct call to os.Exit found, reporting...\n")
					// Регистрируем проблему — прямой вызов os.Exit в main.main
					pass.Reportf(callExpr.Pos(), "direct call to os.Exit is forbidden in main.main")
				}
				/*	else {
					// fmt.Printf("Not os.Exit, skipping\n")
				}*/

				return true
			})
		}
	}

	// fmt.Printf("Analysis complete\n")
	return nil, nil
}
