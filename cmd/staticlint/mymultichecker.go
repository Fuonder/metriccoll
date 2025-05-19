/*
Программа main собирает и запускает набор статических анализаторов кода
на базе пакета golang.org/x/tools/go/analysis/multichecker.

Она объединяет:
  - стандартные анализаторы Go (из пакета analysis/passes),
  - сторонние анализаторы (например, errcheck и ineffassign),
  - пользовательские кастомные анализаторы (например, noosexit),
  - правила из staticcheck, отфильтрованные через конфигурационный JSON-файл.

# Механизм запуска

На вход multichecker.Main подаётся список объектов `*analysis.Analyzer`. Это модуль go/analysis,
который позволяет создавать расширяемые проверки кода. Каждая такая проверка — отдельный анализатор
с собственным алгоритмом обхода AST, типовой информации и отчётности.

Программа запускается аналогично go vet:

	go run main.go ./...
	# или
	go build -o staticlint
	./staticlint ./...

# Конфигурация

Файл `config.json` в корне проекта определяет, какие правила staticcheck активировать.

Пример:

	{
		"Staticcheck": ["SA1000", "SA4006", "ST1000"]
	}

Эти имена берутся из документации к Staticcheck: https://staticcheck.io/docs/checks/

# Встроенные анализаторы (из analysis/passes)

Эти проверки аналогичны встроенным проверкам компилятора и go vet:

  - asmdecl — соответствие объявлений функций на ассемблере.
  - assign — поиск неиспользуемых или бессмысленных присваиваний.
  - atomic — правильное использование atomic.Value.
  - bools — подозрительные логические выражения.
  - buildtag — корректность build-тегов.
  - cgocall — предупреждение о медленных вызовах cgo.
  - composite — ошибки в composite literals.
  - copylock — ошибки при копировании структур с мьютексами.
  - ctrlflow — утечки управления.
  - deepequalerrors — некорректное использование reflect.DeepEqual с ошибками.
  - errorsas — корректность errors.As.
  - httpresponse — закрытие тел HTTP-ответов.
  - ifaceassert — некорректные type assertions интерфейсов.
  - loopclosure — замыкания в циклах.
  - lostcancel — забытый вызов cancel() у контекста.
  - nilfunc — вызов nil-функций.
  - printf — ошибки форматирования строк.
  - shift — неверные побитовые сдвиги.
  - sigchanyzer — ошибки в использовании каналов сигналов.
  - stdmethods — некорректные сигнатуры стандартных методов.
  - stringintconv — подозрительные преобразования string <-> int.
  - structtag — ошибки в тегах структур.
  - testinggoroutine — создание горутин в тестах без синхронизации.
  - tests — ошибки в сигнатурах тестов.
  - unmarshal — ошибки в использовании json.Unmarshal и подобных.
  - unreachable — недостижимый код.
  - unsafeptr — опасное использование unsafe.Pointer.
  - unusedresult — игнорирование результатов важных функций.

# Сторонние анализаторы

  - errcheck — проверка на необработанные ошибки из возвращаемых значений.
    Документация: https://github.com/kisielk/errcheck

  - ineffassign — обнаружение неэффективных (лишних) присваиваний.
    Документация: https://github.com/gordonklaus/ineffassign

# Пользовательские анализаторы

  - noosexit — собственная проверка, запрещающая прямые вызовы os.Exit в main.main.
    Это важно для улучшения тестируемости программы и выделения бизнес-логики.

# Как добавить свой анализатор

 1. Импортируйте пакет с анализатором.
 2. Добавьте его в срез `allChecks`.
 3. Убедитесь, что его имя не конфликтует с другими.

Пример:

	allChecks = append(allChecks, myanalyzer.Analyzer)
*/
package main

import (
	"encoding/json"
	"fmt"
	"github.com/Fuonder/metriccoll.git/noosexit"
	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/kisielk/errcheck/errcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/staticcheck"
	"path/filepath"

	"os"
)

// Config — имя JSON-файла конфигурации анализаторов staticcheck.
const Config = `config.json`

// ConfigData описывает структуру JSON-файла конфигурации, содержащего
// список имен анализаторов из пакета staticcheck, которые нужно активировать.
type ConfigData struct {
	Staticcheck []string // Имена анализаторов (например: "SA1000", "ST1005")
}

func main() {
	// Считывание и разбор файла конфигурации
	CFGFile, err := os.ReadFile(filepath.Join(filepath.Dir("./"), Config))
	if err != nil {
		fmt.Println("Can not read config file")
	}

	var config ConfigData
	err = json.Unmarshal(CFGFile, &config)
	if err != nil {
		fmt.Println("Can not parse config file")
	}
	// Список активных анализаторов
	allChecks := []*analysis.Analyzer{
		// Встроенные анализаторы Go (основные проверки стандартных ошибок)
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		ctrlflow.Analyzer,
		deepequalerrors.Analyzer,
		errorsas.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	}
	// Преобразуем список включённых staticcheck-правил в map для быстрого поиска
	staticChecks := make(map[string]bool)
	for _, name := range config.Staticcheck {
		staticChecks[name] = true
	}

	// Подключение анализаторов staticcheck на основе конфигурации
	for _, v := range staticcheck.Analyzers {
		if staticChecks[v.Analyzer.Name] {
			allChecks = append(allChecks, v.Analyzer)
		}
	}

	// Добавление сторонних статических анализаторов.
	allChecks = append(allChecks, ineffassign.Analyzer)
	allChecks = append(allChecks, errcheck.Analyzer)

	// Добавление кастомного анализатора, запрещающего вызовы os.Exit в main.main
	allChecks = append(allChecks, noosexit.Analyzer)

	// Запуск всех анализаторов через multichecker
	multichecker.Main(
		allChecks...)
}
