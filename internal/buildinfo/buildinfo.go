/*
Package buildinfo предоставляет структуру и функции для хранения и отображения
информации о сборке приложения, включая версию, коммит, дату сборки и используемый компилятор.
*/
package buildinfo

import (
	"fmt"
	"runtime"
	"time"
)

// BuildInfo содержит метаданные о сборке приложения.
type BuildInfo struct {
	// BuildVersion — версия сборки, полученная из Git-тегов.
	BuildVersion string `json:"version"`

	// BuildCommit — короткий хеш коммита Git, отражающий состояние репозитория.
	BuildCommit string `json:"commit_id"`

	// BuildDate — дата и время сборки в формате RFC3339.
	BuildDate time.Time `json:"time"`

	// Compiler — версия компилятора Go, использованного при сборке.
	Compiler string `json:"compiler"`
}

// NewBuildInfo создаёт и возвращает объект BuildInfo,
// используя переданные значения версии, коммита и даты сборки,
// полученные, как правило, через флаги линковки (`-ldflags`).
//
// Если также предоставлен `GeneratedBuild` (например, сгенерированный заранее через `go generate`),
// то он будет использован как основа. Однако приоритет имеют параметры Version, Commit и Date:
// если они заданы (не равны "N/A"), они переопределяют соответствующие поля `GeneratedBuild`.
//
// Это позволяет комбинировать статическую генерацию структуры и динамическое внедрение информации при сборке.
//
// Параметры:
//   - Version: строка с версией сборки, полученная через `-ldflags`; если "N/A", используется значение из `GeneratedBuild`.
//   - Commit: строка с хешем коммита Git; если "N/A", используется значение из `GeneratedBuild`.
//   - Date: строка с датой и временем сборки в формате RFC3339; если "N/A", используется значение из `GeneratedBuild`.
//   - GeneratedBuild: указатель на структуру BuildInfo, сгенерированную на этапе генерации исходников (`go generate`).
//
// Возвращает:
//   - *BuildInfo: объект с заполненными данными сборки, основанный на `GeneratedBuild` и/или переданных параметрах.
func NewBuildInfo(Version string, Commit string, Date string, GeneratedBuild *BuildInfo) *BuildInfo {
	if GeneratedBuild != nil {
		if Version != "N/A" {
			GeneratedBuild.BuildVersion = Version
		}
		if Commit != "N/A" {
			GeneratedBuild.BuildCommit = Commit
		}
		if Date != "N/A" {
			GeneratedBuild.BuildDate = MustParseTime(Date)
		}
		return GeneratedBuild
	}
	bInfo := &BuildInfo{
		BuildVersion: Version,
		BuildCommit:  Commit,
		BuildDate:    MustParseTime(Date),
		Compiler:     runtime.Version(),
	}
	return bInfo
}

// String возвращает форматированное строковое представление объекта BuildInfo.
//
// Выходной формат включает версию сборки, коммит, дату и версию компилятора.
//
// Возвращает:
//   - string: человекочитаемое представление метаданных сборки.
func (b *BuildInfo) String() string {
	return fmt.Sprintf(
		"Build Version : %s\nBuild Commit  : %s\nBuild Date    : %s\nCompiler      : %s\n",
		b.BuildVersion, b.BuildCommit, b.BuildDate.Format(time.RFC3339), b.Compiler,
	)
}

// MustParseTime преобразует строку даты в формате RFC3339 во временное значение time.Time.
//
// Если строка не может быть распознана, возвращается текущее время UTC.
//
// Параметры:
//   - val: строка, представляющая дату и время.
//
// Возвращает:
//   - time.Time: распарсенная дата/время или текущий момент времени в UTC при ошибке.
func MustParseTime(val string) time.Time {
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return time.Now().UTC()
	}
	return t
}
