package kr

/*
#cgo darwin LDFLAGS: -lsqlite3 -framework Security -framework Security -framework CoreFoundation -lSystem -lresolv -lc -lm
#cgo LDFLAGS: -L ${SRCDIR}/krcommand/target/release -lkrcommand

#include <stdlib.h>
#include "krcommand/target/include/krcommand.h"
*/
import "C"

func CreateTeam(name string) {
	bytes := C.CBytes([]byte(name))
	C.create_team((*C.uint8_t)(bytes), C.uintptr_t(len(name)))
	C.free(bytes)
}

func CreateInvite() {
	C.create_invite()
}

func SetApprovalWindow(approval_window *int64) {
	C.set_policy((*C.int64_t)(approval_window))
}
