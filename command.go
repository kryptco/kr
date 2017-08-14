package kr

/*
#cgo darwin LDFLAGS: -lsqlite3 -framework Security -framework Security -framework CoreFoundation -lSystem -lresolv -lc -lm
#cgo LDFLAGS: -L ${SRCDIR}/krcommand/target/release -lkrcommand

#include <stdlib.h>
#include "krcommand/target/include/krcommand.h"
*/
import "C"

func SaveAdminKeypair(seed []byte) {
	bytes := C.CBytes(seed)
	C.save_admin_keypair((*C.uint8_t)(bytes), C.uintptr_t(len(seed)))
	C.free(bytes)
}

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

func GetMembers(query *string, printSSHPubkey bool, printPGPPubkey bool) {
	if query != nil {
		bytes := C.CBytes([]byte(*query))
		C.get_members((*C.uint8_t)(bytes), C.uintptr_t(len(*query)),
			C._Bool(printSSHPubkey), C._Bool(printPGPPubkey))
		C.free(bytes)
	} else {
		C.get_members((*C.uint8_t)(nil), C.uintptr_t(0),
			C._Bool(printSSHPubkey), C._Bool(printPGPPubkey))
	}
}
