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
	nameSlice := []byte(name)
	bytes := C.CBytes(nameSlice)
	C.create_team((*C.uint8_t)(bytes), C.uintptr_t(len(nameSlice)))
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
		querySlice := []byte(*query)
		bytes := C.CBytes(querySlice)
		C.get_members((*C.uint8_t)(bytes), C.uintptr_t(len(querySlice)),
			C._Bool(printSSHPubkey), C._Bool(printPGPPubkey))
		C.free(bytes)
	} else {
		C.get_members((*C.uint8_t)(nil), C.uintptr_t(0),
			C._Bool(printSSHPubkey), C._Bool(printPGPPubkey))
	}
}

func AddAdmin(email string) {
	emailSlice := []byte(email)
	bytes := C.CBytes(emailSlice)
	C.add_admin((*C.uint8_t)(bytes), C.uintptr_t(len(emailSlice)))
	C.free(bytes)
}

func RemoveAdmin(email string) {
	emailSlice := []byte(email)
	bytes := C.CBytes(emailSlice)
	C.remove_admin((*C.uint8_t)(bytes), C.uintptr_t(len(emailSlice)))
	C.free(bytes)
}

func GetAdmins() {
	C.get_admins()
}

func PinHostKey(host string, publicKey []byte) {
	hostSlice := []byte(host)
	hostBytes := C.CBytes(hostSlice)
	defer C.free(hostBytes)
	publicKeyBytes := C.CBytes(publicKey)
	defer C.free(publicKeyBytes)

	C.pin_host_key(
		(*C.uint8_t)(hostBytes), C.uintptr_t(len(hostSlice)),
		(*C.uint8_t)(publicKeyBytes), C.uintptr_t(len(publicKey)),
	)
}

func PinKnownHostKeys(host string, updateFromServer bool) {
	hostSlice := []byte(host)
	hostBytes := C.CBytes(hostSlice)
	defer C.free(hostBytes)

	C.pin_known_host_keys((*C.uint8_t)(hostBytes), C.uintptr_t(len(hostSlice)),
		C._Bool(updateFromServer))
}

func UnpinHostKey(host string, publicKey []byte) {
	hostSlice := []byte(host)
	hostBytes := C.CBytes(hostSlice)
	defer C.free(hostBytes)
	publicKeyBytes := C.CBytes(publicKey)
	defer C.free(publicKeyBytes)

	C.unpin_host_key(
		(*C.uint8_t)(hostBytes), C.uintptr_t(len(hostSlice)),
		(*C.uint8_t)(publicKeyBytes), C.uintptr_t(len(publicKey)),
	)
}

func GetAllPinnedHostKeys() {
	C.get_all_pinned_host_keys()
}

func GetPinnedHostKeys(host string, search bool) {
	hostSlice := []byte(host)
	hostBytes := C.CBytes(hostSlice)
	defer C.free(hostBytes)

	C.get_pinned_host_keys(
		(*C.uint8_t)(hostBytes), C.uintptr_t(len(hostSlice)),
		C._Bool(search),
	)
}
