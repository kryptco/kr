package sigchaingobridge

/*
#cgo darwin LDFLAGS: -framework Security -framework Security -framework CoreFoundation -lSystem -lresolv -lc -lm
#cgo LDFLAGS: -lsigchain
#cgo linux LDFLAGS: -lutil -lutil -lrt -lpthread -ldl -lgcc_s -lc -lm -lrt -lpthread -lutil -lutil

#include <stdlib.h>
#include "../sigchain/target/include/sigchain.h"
*/
import (
	"C"
)
import (
	"os"
	"strings"
	"unsafe"
)

func InviteEmails(emails []string) {
	emailsStringSlice := []byte(strings.Join(emails, ","))
	bytes := C.CBytes(emailsStringSlice)
	C.create_indirect_emails_invite((*C.uint8_t)(bytes), C.uintptr_t(len(emailsStringSlice)))
	C.free(bytes)
}

func InviteDomain(domain string) {
	domainSlice := []byte(domain)
	bytes := C.CBytes(domainSlice)
	C.create_indirect_domain_invite((*C.uint8_t)(bytes), C.uintptr_t(len(domainSlice)))
	C.free(bytes)
}

func CancelInvite() {
	C.cancel_invite()
}

func RemoveMemberCommand(email string) {
	emailSlice := []byte(email)
	bytes := C.CBytes(emailSlice)
	C.remove_member((*C.uint8_t)(bytes), C.uintptr_t(len(emailSlice)))
	C.free(bytes)
}

func SetTeamName(name string) {
	nameSlice := []byte(name)
	bytes := C.CBytes(nameSlice)
	C.set_team_name((*C.uint8_t)(bytes), C.uintptr_t(len(nameSlice)))
	C.free(bytes)
}

func GetPolicy() {
	C.get_policy()
}

func SetApprovalWindow(approval_window *int64) {
	C.set_policy((*C.int64_t)(approval_window))
}

func GetMembers(email *string, printSSHPubkey bool, printPGPPubkey bool, admin bool) {
	if email != nil {
		emailSlice := []byte(*email)
		bytes := C.CBytes(emailSlice)
		C.get_members((*C.uint8_t)(bytes), C.uintptr_t(len(emailSlice)),
			C._Bool(printSSHPubkey), C._Bool(printPGPPubkey), C._Bool(admin))
		C.free(bytes)
	} else {
		C.get_members((*C.uint8_t)(nil), C.uintptr_t(0),
			C._Bool(printSSHPubkey), C._Bool(printPGPPubkey), C._Bool(admin))
	}
}

func IsAdmin() bool {
	return (bool)(C.is_admin())
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

func EnableLogging() {
	C.enable_logging()
}

func UpdateTeamLogs() {
	C.update_team_logs()
}

func OpenBilling() {
	C.open_billing()
}

func ViewLogs() {
	C.view_logs()
}

func ServeDashboard() {
	C.serve_dashboard()
}

func ServeDashboardIfParamsPresent() {
	C.serve_dashboard_if_params_present()
}

func KrAdd() {
	args, n := osArgsToCArray()
	C.kr_add(args, n)
	C.free(unsafe.Pointer(args))
}

func KrList() {
	args, n := osArgsToCArray()
	C.kr_list(args, n)
	C.free(unsafe.Pointer(args))
}

func KrRm() {
	args, n := osArgsToCArray()
	C.kr_rm(args, n)
	C.free(unsafe.Pointer(args))
}

// Caller must free returned array
func osArgsToCArray() (**C.char, C.uintptr_t) {
	// Pass arguments from Go since the rust native library std::env::args returns an empty list on Linux
	// https://stackoverflow.com/questions/41492071/how-do-i-convert-a-go-array-of-strings-to-a-c-array-of-strings
	goArgs := os.Args
	cArray := C.malloc(C.size_t(len(goArgs)) * C.size_t(unsafe.Sizeof(uintptr(0))))

	a := (*[1<<30 - 1]*C.char)(cArray)

	for idx, arg := range goArgs {
		a[idx] = C.CString(arg)
	}

	return (**C.char)(cArray), C.uintptr_t(len(goArgs))
}
