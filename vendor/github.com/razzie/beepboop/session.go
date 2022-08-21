package beepboop

import (
	"context"
	"net/http"
	"time"

	"github.com/razzie/reqip"
)

// Session ...
type Session struct {
	ctx       context.Context
	sessionID string
	requestID string
	ip        string
	allAccess AccessMap
	newAccess AccessMap
	db        *DB
}

func newSession(r *PageRequest) *Session {
	sess := &Session{
		ctx:       r.Request.Context(),
		requestID: r.RequestID,
		ip:        reqip.GetClientIP(r.Request),
		allAccess: make(AccessMap),
		newAccess: make(AccessMap),
		db:        r.Context.DB,
	}

	for _, c := range r.Request.Cookies() {
		if c.Name == "session" {
			sess.sessionID = c.Value
			continue
		}
		sess.allAccess.fromCookie(c)
	}

	db := r.Context.DB
	if db != nil && len(sess.sessionID) > 0 {
		dbAccess, err := db.getAccessMap(sess.sessionID, sess.ip)
		if err == nil {
			sess.allAccess.Merge(dbAccess)
		} else {
			r.Log(err)
		}
	}

	return sess
}

// Context returns the context of the session
func (sess *Session) Context() context.Context {
	return sess.ctx
}

// SessionID returns the sessionID
func (sess *Session) SessionID() string {
	return sess.sessionID
}

// IP returns the session IP address
func (sess *Session) IP() string {
	return sess.ip
}

// GetAccessCode returns the access code to the given resource
func (sess *Session) GetAccessCode(accessType, resource string) (string, bool) {
	return sess.allAccess.Get(accessType, resource)
}

// AddAccess permits the requester to access the given resource
func (sess *Session) AddAccess(accessType, resource, accesscode string) error {
	access := make(AccessMap)
	access.Add(accessType, resource, accesscode)
	return sess.MergeAccess(access)
}

// RemoveAccess removes the requester's access to the given resource
func (sess *Session) RemoveAccess(accessType, resource string) error {
	revoke := AccessRevokeMap{
		AccessType(accessType): AccessResourceName(resource),
	}
	return sess.RevokeAccess(revoke)
}

// MergeAccess permits the requester to access the given resources
func (sess *Session) MergeAccess(access AccessMap) error {
	if sess.db != nil {
		if len(sess.sessionID) == 0 {
			sess.sessionID = sess.requestID
		}
		return sess.db.addSessionAccess(sess.sessionID, sess.ip, access)
	}
	sess.newAccess.Merge(access)
	sess.allAccess.Merge(access)
	return nil
}

// RevokeAccess revokes the requester's access to the given resources
func (sess *Session) RevokeAccess(revoke AccessRevokeMap) error {
	if sess.db != nil && len(sess.sessionID) > 0 {
		return sess.db.revokeSessionAccess(sess.sessionID, sess.ip, revoke)
	}
	sess.newAccess.Revoke(revoke, true)
	sess.allAccess.Revoke(revoke, false)
	return nil
}

func (sess *Session) getSessionCookie(expiration time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:    "session",
		Value:   sess.sessionID,
		Path:    "/",
		Expires: time.Now().Add(expiration),
	}
}

func (sess *Session) toCookies(expiration time.Duration) []*http.Cookie {
	if len(sess.sessionID) > 0 {
		return []*http.Cookie{sess.getSessionCookie(expiration)}
	}
	return sess.newAccess.ToCookies(expiration)
}
