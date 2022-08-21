package beepboop

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AccessType is the type of access, like 'read' or 'write'
type AccessType string

// AccessResourceName is the name of the resource, like a user or a folder
type AccessResourceName string

// AccessCode is the code that proves the access is valid
type AccessCode string

// AccessMap is the map of available accesses to various resources
type AccessMap map[AccessType]map[AccessResourceName]AccessCode

// AccessRevokeMap is used to ravoke access to various resources
type AccessRevokeMap map[AccessType]AccessResourceName

// Add adds access to a resource
func (m AccessMap) Add(accessType, resource, code string) {
	resMap, ok := m[AccessType(accessType)]
	if !ok {
		resMap = make(map[AccessResourceName]AccessCode)
		m[AccessType(accessType)] = resMap
	}

	resMap[AccessResourceName(resource)] = AccessCode(code)
}

// Remove removes access to a resource
func (m AccessMap) Remove(accessType, resource string) {
	resMap, ok := m[AccessType(accessType)]
	if ok {
		delete(resMap, AccessResourceName(resource))
		if len(resMap) == 0 {
			delete(m, AccessType(accessType))
		}
	}
}

// Get gets the access code to a resource
func (m AccessMap) Get(accessType, resource string) (string, bool) {
	code, ok := m[AccessType(accessType)][AccessResourceName(resource)]
	return string(code), ok
}

// Merge merges an AccessMap into this AccessMap
func (m AccessMap) Merge(other AccessMap) {
	for typ, res := range other {
		for resname, code := range res {
			m.Add(string(typ), string(resname), string(code))
		}
	}
}

// Revoke revokes access to resources from this AccessMap
func (m AccessMap) Revoke(revoke AccessRevokeMap, keep bool) {
	if keep {
		for typ, resname := range revoke {
			m.Add(string(typ), string(resname), "")
		}
		return
	}
	for typ, resname := range revoke {
		m.Remove(string(typ), string(resname))
	}
}

// ToCookies returns a list of cookies containing access to the resources in the access token
func (m AccessMap) ToCookies(expiration time.Duration) []*http.Cookie {
	var cookies []*http.Cookie
	for typ, res := range m {
		for resname, code := range res {
			cookie := &http.Cookie{
				Name:  url.PathEscape(string(typ)) + "-" + url.PathEscape(string(resname)),
				Value: url.PathEscape(string(code)),
				Path:  "/",
			}
			if len(code) > 0 {
				cookie.Expires = time.Now().Add(expiration)
			}
			cookies = append(cookies, cookie)
		}
	}
	return cookies
}

func (m AccessMap) fromCookie(cookie *http.Cookie) {
	access := strings.SplitN(cookie.Name, "-", 2)
	if len(access) < 2 {
		return
	}
	typ, _ := url.PathUnescape(access[0])
	resname, _ := url.PathUnescape(access[1])
	code, _ := url.PathUnescape(cookie.Value)
	if len(typ) > 0 && len(resname) > 0 && len(code) > 0 {
		m.Add(typ, resname, code)
	}
}
