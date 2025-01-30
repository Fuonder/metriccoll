package storage

const (
	OsRead       = 04
	OsWrite      = 02
	OsEx         = 01
	OsUserShift  = 6
	OsGroupShift = 3
	OsOthShift   = 0

	OsUserR   = OsRead << OsUserShift
	OsUserW   = OsWrite << OsUserShift
	OsUserX   = OsEx << OsUserShift
	OsUserRw  = OsUserR | OsUserW
	OsUserRwx = OsUserRw | OsUserX

	OsGroupR   = OsRead << OsGroupShift
	OsGroupW   = OsWrite << OsGroupShift
	OsGroupX   = OsEx << OsGroupShift
	OsGroupRw  = OsGroupR | OsGroupW
	OsGroupRwx = OsGroupRw | OsGroupX

	OsOthR   = OsRead << OsOthShift
	OsOthW   = OsWrite << OsOthShift
	OsOthX   = OsEx << OsOthShift
	OsOthRw  = OsOthR | OsOthW
	OsOthRwx = OsOthRw | OsOthX

	OsAllR   = OsUserR | OsGroupR | OsOthR
	OsAllW   = OsUserW | OsGroupW | OsOthW
	OsAllX   = OsUserX | OsGroupX | OsOthX
	OsAllRw  = OsAllR | OsAllW
	OsAllRwx = OsAllRw | OsAllX
)
