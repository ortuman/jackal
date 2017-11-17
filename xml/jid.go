/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

/*
#cgo LDFLAGS: -lidn
#include <stdlib.h>
#include "stringprep.h"

int nameprep(char *in, size_t maxlen) {
	return stringprep_nameprep(in, maxlen);
}

int nodeprep(char *in, size_t maxlen) {
	return stringprep_xmpp_nodeprep(in, maxlen);
}

int resourceprep(char *in, size_t maxlen) {
	return stringprep_xmpp_resourceprep(in, maxlen);
}
*/
import "C"
import "unsafe"

type JID struct {
	user     string
	domain   string
	resource string
}

func NewJID(user string, domain string, resource string) (*JID, error) {
	return &JID{
		user:     user,
		domain:   domain,
		resource: resource,
	}, nil
}

func NewJIDString(jidStr string) (*JID, error) {
	return nil, nil
}

func nodeprep(in string) (string, error) {
	var cin *C.char = C.CString(in)
	defer C.free(unsafe.Pointer(cin))
	if C.nodeprep(cin, C.size_t(len(in))) != 0 {
		return "", nil
	}
	return C.GoString(cin), nil
}

func domainprep(in string) (string, error) {
	var cin *C.char = C.CString(in)
	defer C.free(unsafe.Pointer(cin))
	if C.nameprep(cin, C.size_t(len(in))) != 0 {
		return "", nil
	}
	return C.GoString(cin), nil
}

func resourceprep(in string) (string, error) {
	var cin *C.char = C.CString(in)
	defer C.free(unsafe.Pointer(cin))
	if C.resourceprep(cin, C.size_t(len(in))) != 0 {
		return "", nil
	}
	return C.GoString(cin), nil
}
