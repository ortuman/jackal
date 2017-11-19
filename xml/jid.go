/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package xml

/*
#cgo LDFLAGS: -lidn
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "stringprep.h"

char *nodeprep(char *in) {
	int maxlen = strlen(in)*4 + 1;
	char *buf = (char *)(malloc(maxlen));

	strcpy(buf, in);
	int rc = stringprep_xmpp_nodeprep(buf, maxlen);
	if (rc != 0) {
		free(buf);
		return NULL;
	}
	return buf;
}

char *domainprep(char *str) {
	char *in = stringprep_convert(str, "ASCII", "UTF-8");

	int maxlen = strlen(in)*4 + 1;
	char *buf = (char *)(malloc(maxlen));

	strcpy(buf, in);
	free(in);

	int rc = stringprep_nameprep(buf, maxlen);
	if (rc != 0) {
		free(buf);
		return NULL;
	}
	return buf;
}

char *resourceprep(char *in) {
	int maxlen = strlen(in)*4 + 1;
	char *buf = (char *)(malloc(maxlen));

	strcpy(buf, in);
	int rc = stringprep_xmpp_resourceprep(buf, maxlen);
	if (rc != 0) {
		free(buf);
		return NULL;
	}
	return buf;
}
*/
import "C"
import (
	"errors"
	"fmt"
	"strings"
	"unsafe"
)

type JID struct {
	User     string
	Domain   string
	Resource string
}

// NewJID constructs a JID given a user, domain, and resource.
func NewJID(user, domain, resource string) (*JID, error) {
	prepUser, err := nodeprep(user)
	if err != nil {
		return nil, err
	}
	prepDomain, err := domainprep(domain)
	if err != nil {
		return nil, err
	}
	prepResource, err := resourceprep(resource)
	if err != nil {
		return nil, err
	}
	return &JID{
		User:     prepUser,
		Domain:   prepDomain,
		Resource: prepResource,
	}, nil
}

// NewJIDString constructs a JID from it's string representation.
func NewJIDString(str string) (*JID, error) {
	if len(str) == 0 {
		return &JID{}, nil
	}
	var user, domain, resource string

	atIndex := strings.Index(str, "@")
	slashIndex := strings.Index(str, "/")

	// user
	if atIndex > 0 {
		user = str[0:atIndex]
	}

	// domain
	if atIndex+1 > len(str) {
		return nil, errors.New("JID with empty domain not valid")
	}
	if atIndex < 0 {
		if slashIndex > 0 {
			domain = str[0:slashIndex]
		} else {
			domain = str
		}
	} else {
		if slashIndex > 0 {
			domain = str[atIndex+1 : slashIndex]
		} else {
			domain = str[atIndex+1:]
		}
	}

	// resource
	if slashIndex > 0 && slashIndex+1 < len(str) {
		resource = str[slashIndex+1:]
	}
	return NewJID(user, domain, resource)
}

// ToBareJID returns the string representation of the bare JID, which is the JID with resource information removed.
func (j *JID) ToBareJID() string {
	if len(j.User) == 0 {
		return j.Domain
	}
	return j.User + "@" + j.Domain
}

// ToFullJID returns the String representation of the full JID.
func (j *JID) ToFullJID() string {
	if len(j.Resource) == 0 {
		return j.ToBareJID()
	}
	if len(j.User) == 0 {
		return j.Domain + "/" + j.Resource
	}
	return j.User + "@" + j.Domain + "/" + j.Resource
}

// Equals returns true if two JID's are equivalent.
func (j *JID) Equals(j2 *JID) bool {
	if j == j2 {
		return true
	}
	if j.User != j2.User {
		return false
	}
	if j.Domain != j2.Domain {
		return false
	}
	if j.Resource != j2.Resource {
		return false
	}
	return true
}

func nodeprep(in string) (string, error) {
	cin := C.CString(in)
	defer C.free(unsafe.Pointer(cin))

	prep := C.nodeprep(cin)
	if prep == nil {
		return "", fmt.Errorf("input is not a valid JID node part: %s", in)
	}
	defer C.free(unsafe.Pointer(prep))
	if C.strlen(prep) > 1073 {
		return "", fmt.Errorf("node cannot be larger than 1073. Size is %d bytes", C.strlen(prep))
	}
	return C.GoString(prep), nil
}

func domainprep(in string) (string, error) {
	cin := C.CString(in)
	defer C.free(unsafe.Pointer(cin))

	prep := C.domainprep(cin)
	if prep == nil {
		return "", fmt.Errorf("input is not a valid JID domain part: %s", in)
	}
	defer C.free(unsafe.Pointer(prep))
	if C.strlen(prep) > 1073 {
		return "", fmt.Errorf("domain cannot be larger than 1073. Size is %d bytes", C.strlen(prep))
	}
	return C.GoString(prep), nil
}

func resourceprep(in string) (string, error) {
	cin := C.CString(in)
	defer C.free(unsafe.Pointer(cin))

	prep := C.resourceprep(cin)
	if prep == nil {
		return "", fmt.Errorf("input is not a valid JID resource part: %s", in)
	}
	defer C.free(unsafe.Pointer(prep))
	if C.strlen(prep) > 1073 {
		return "", fmt.Errorf("resource cannot be larger than 1073. Size is %d bytes", C.strlen(prep))
	}
	return C.GoString(prep), nil
}
