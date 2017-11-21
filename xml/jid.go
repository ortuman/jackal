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

// JID represents an XMPP address (JID).
// A JID is made up of a node (generally a username), a domain, and a resource.
// The node and resource are optional; domain is required.
type JID struct {
	node     string
	domain   string
	resource string
}

// NewJID constructs a JID given a user, domain, and resource.
// This construction allows the caller to specify if stringprep should be applied or not.
func NewJID(node, domain, resource string, skipStringPrep bool) (*JID, error) {
	if skipStringPrep {
		return &JID{
			node:     node,
			domain:   domain,
			resource: resource,
		}, nil
	}
	prepNode, err := nodeprep(node)
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
		node:     prepNode,
		domain:   prepDomain,
		resource: prepResource,
	}, nil
}

// NewJIDString constructs a JID from it's string representation.
// This construction allows the caller to specify if stringprep should be applied or not.
func NewJIDString(str string, skipStringPrep bool) (*JID, error) {
	if len(str) == 0 {
		return &JID{}, nil
	}
	var node, domain, resource string

	atIndex := strings.Index(str, "@")
	slashIndex := strings.Index(str, "/")

	// node
	if atIndex > 0 {
		node = str[0:atIndex]
	}

	// domain
	if atIndex+1 == len(str) {
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
	return NewJID(node, domain, resource, skipStringPrep)
}

// Node returns the node, or empty string if this JID does not contain node information.
func (j *JID) Node() string {
	return j.node
}

// Domain returns the domain.
func (j *JID) Domain() string {
	return j.domain
}

// Resource returns the resource, or empty string if this JID does not contain resource information.
func (j *JID) Resource() string {
	return j.resource
}

// ToBareJID returns the string representation of the bare JID, which is the JID with resource information removed.
func (j *JID) ToBareJID() string {
	if len(j.node) == 0 {
		return j.domain
	}
	return j.node + "@" + j.domain
}

// ToFullJID returns the String representation of the full JID.
func (j *JID) ToFullJID() string {
	if len(j.resource) == 0 {
		return j.ToBareJID()
	}
	if len(j.node) == 0 {
		return j.domain + "/" + j.resource
	}
	return j.node + "@" + j.domain + "/" + j.resource
}

// Equals returns true if two JID's are equivalent.
func (j *JID) IsEqual(j2 *JID) bool {
	if j == j2 {
		return true
	}
	if j.node != j2.node {
		return false
	}
	if j.domain != j2.domain {
		return false
	}
	if j.resource != j2.resource {
		return false
	}
	return true
}

// String returns a string representation of the JID.
func (j *JID) String() string {
	return j.ToFullJID()
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
