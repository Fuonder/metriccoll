// Package storage содержит константы для определения прав доступа к файлам,
// такие как права на чтение, запись и выполнение, а также смещения для пользователей, групп и других.
// Эти константы могут быть использованы для работы с правами доступа при открытии и создании файлов.
package storage

const (
	// OsRead — право на чтение.
	OsRead = 04
	// OsWrite — право на запись.
	OsWrite = 02
	// OsEx — право на выполнение.
	OsEx = 01

	// OsUserShift — смещение для прав пользователя (User).
	OsUserShift = 6
	// OsGroupShift — смещение для прав группы (Group).
	OsGroupShift = 3
	// OsOthShift — смещение для прав остальных (Others).
	OsOthShift = 0

	// OsUserR — право на чтение для пользователя.
	OsUserR = OsRead << OsUserShift
	// OsUserW — право на запись для пользователя.
	OsUserW = OsWrite << OsUserShift
	// OsUserX — право на выполнение для пользователя.
	OsUserX = OsEx << OsUserShift
	// OsUserRw — право на чтение и запись для пользователя.
	OsUserRw = OsUserR | OsUserW
	// OsUserRwx — право на чтение, запись и выполнение для пользователя.
	OsUserRwx = OsUserRw | OsUserX

	// OsGroupR — право на чтение для группы.
	OsGroupR = OsRead << OsGroupShift
	// OsGroupW — право на запись для группы.
	OsGroupW = OsWrite << OsGroupShift
	// OsGroupX — право на выполнение для группы.
	OsGroupX = OsEx << OsGroupShift
	// OsGroupRw — право на чтение и запись для группы.
	OsGroupRw = OsGroupR | OsGroupW
	// OsGroupRwx — право на чтение, запись и выполнение для группы.
	OsGroupRwx = OsGroupRw | OsGroupX

	// OsOthR — право на чтение для остальных пользователей.
	OsOthR = OsRead << OsOthShift
	// OsOthW — право на запись для остальных пользователей.
	OsOthW = OsWrite << OsOthShift
	// OsOthX — право на выполнение для остальных пользователей.
	OsOthX = OsEx << OsOthShift
	// OsOthRw — право на чтение и запись для остальных пользователей.
	OsOthRw = OsOthR | OsOthW
	// OsOthRwx — право на чтение, запись и выполнение для остальных пользователей.
	OsOthRwx = OsOthRw | OsOthX

	// OsAllR — права на чтение для всех пользователей (пользователь, группа и другие).
	OsAllR = OsUserR | OsGroupR | OsOthR
	// OsAllW — права на запись для всех пользователей.
	OsAllW = OsUserW | OsGroupW | OsOthW
	// OsAllX — права на выполнение для всех пользователей.
	OsAllX = OsUserX | OsGroupX | OsOthX
	// OsAllRw — права на чтение и запись для всех пользователей.
	OsAllRw = OsAllR | OsAllW
	// OsAllRwx — права на чтение, запись и выполнение для всех пользователей.
	OsAllRwx = OsAllRw | OsAllX
)
