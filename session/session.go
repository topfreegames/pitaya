// Copyright (c) nano Author and TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package session

import (
	"bytes"
	"encoding/gob"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	nats "github.com/nats-io/go-nats"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/util"
)

// NetworkEntity represent low-level network instance
type NetworkEntity interface {
	Push(route string, v interface{}) error
	ResponseMID(mid uint, v interface{}) error
	Close() error
	RemoteAddr() net.Addr
	SendRequest(serverID, route string, v interface{}) (*protos.Response, error)
}

var (
	log = logger.Log
	//ErrIllegalUID represents a invalid uid
	ErrIllegalUID = errors.New("illegal uid")
	// OnSessionBind represents the function called after the session in bound
	OnSessionBind func(s *Session) error
	sessionsByUID = make(map[string]*Session)
	sessionsByID  = make(map[int64]*Session)
	sessionIDSvc  = newSessionIDService()
)

// Session represents a client session which could storage temp data during low-level
// keep connected, all data will be released when the low-level connection was broken.
// Session instance related to the client will be passed to Handler method as the first
// parameter.
type Session struct {
	sync.RWMutex                             // protect data
	id                int64                  // session global unique id
	uid               string                 // binding user id
	lastTime          int64                  // last heartbeat time
	entity            NetworkEntity          // low-level network entity
	data              map[string]interface{} // session data store
	encodedData       []byte                 // session data encoded as a byte array
	OnCloseCallbacks  []func()               //onClose callbacks
	IsFrontend        bool                   // if session is a frontend session
	frontendID        string                 // the id of the frontend that owns the session
	frontendSessionID int64                  // the id of the session on the frontend server
	Subscription      *nats.Subscription     // subscription created on bind when using nats rpc server
}

// Data to send over rpc
type Data struct {
	ID   int64
	UID  string
	Data map[string]interface{}
}

type sessionIDService struct {
	sid int64
}

func newSessionIDService() *sessionIDService {
	return &sessionIDService{
		sid: 0,
	}
}

// SessionID returns the session id
func (c *sessionIDService) sessionID() int64 {
	return atomic.AddInt64(&c.sid, 1)
}

// New returns a new session instance
// a NetworkEntity is a low-level network instance
func New(entity NetworkEntity, frontend bool) *Session {
	s := &Session{
		id:               sessionIDSvc.sessionID(),
		entity:           entity,
		data:             make(map[string]interface{}),
		lastTime:         time.Now().Unix(),
		OnCloseCallbacks: []func(){},
		IsFrontend:       frontend,
	}
	if frontend {
		sessionsByID[s.id] = s
	}
	return s
}

// GetSessionByUID return a session bound to an user id
func GetSessionByUID(uid string) *Session {
	if val, ok := sessionsByUID[uid]; ok {
		return val
	}
	return nil
}

// GetSessionByID return a session bound to a frontend server id
func GetSessionByID(id int64) *Session {
	if val, ok := sessionsByID[id]; ok {
		return val
	}
	return nil
}

func (s *Session) updateEncodedData() error {
	buf := bytes.NewBuffer([]byte(nil))
	err := gob.NewEncoder(buf).Encode(s.data)
	if err != nil {
		return err
	}
	s.encodedData = buf.Bytes()
	return nil
}

// Push message to client
func (s *Session) Push(route string, v interface{}) error {
	return s.entity.Push(route, v)
}

// ResponseMID responses message to client, mid is
// request message ID
func (s *Session) ResponseMID(mid uint, v interface{}) error {
	return s.entity.ResponseMID(mid, v)
}

// ID returns the session id
func (s *Session) ID() int64 {
	return s.id
}

// UID returns uid that bind to current session
// TODO this used to use atomic, is it necessary?
func (s *Session) UID() string {
	return s.uid
}

// GetData gets the data
func (s *Session) GetData() map[string]interface{} {
	s.RLock()
	defer s.RUnlock()

	return s.data
}

// SetData sets the whole session data
func (s *Session) SetData(data map[string]interface{}) error {
	s.Lock()
	defer s.Unlock()

	s.data = data
	return s.updateEncodedData()
}

// GetDataEncoded returns the session data as an encoded value
func (s *Session) GetDataEncoded() []byte {
	return s.encodedData
}

// SetDataEncoded sets the whole session data from an encoded value
func (s *Session) SetDataEncoded(encodedData []byte) error {
	if len(encodedData) == 0 {
		return nil
	}
	var data map[string]interface{}
	err := util.GobDecode(&data, encodedData)
	if err != nil {
		return err
	}
	return s.SetData(data)
}

// SetUID sets uid but without binding, TODO remove this method
// Better to have a backend session type
func (s *Session) SetUID(uid string) {
	s.uid = uid
}

// SetFrontendData sets frontend id and session id
func (s *Session) SetFrontendData(frontendID string, frontendSessionID int64) {
	s.frontendID = frontendID
	s.frontendSessionID = frontendSessionID
}

// Bind bind UID to current session
func (s *Session) Bind(uid string) error {
	if uid == "" {
		return ErrIllegalUID
	}

	if s.UID() != "" {
		return constants.ErrSessionAlreadyBound
	}

	s.uid = uid
	// TODO should we overwrite or return an error if the session was already bound
	// TODO MUTEX OR SYNCMAP!
	if OnSessionBind != nil {
		err := OnSessionBind(s)
		if err != nil {
			s.uid = ""
			return err
		}
	}

	// if code running on frontend server
	if s.IsFrontend {
		sessionsByUID[uid] = s
	} else {
		// If frontentID is set this means it is a remote call and the current server
		// is not the frontend server that received the user request
		err := s.bindInFront()
		if err != nil {
			log.Error("error while trying to push session to front: ", err)
			s.uid = ""
			return err
		}
	}
	return nil
}

// OnClose adds the function it receives to the callbacks that will be called
// when the session is closed
func (s *Session) OnClose(c func()) {
	s.OnCloseCallbacks = append(s.OnCloseCallbacks, c)
}

// Close terminate current session, session related data will not be released,
// all related data should be cleared explicitly in Session closed callback
func (s *Session) Close() {
	delete(sessionsByUID, s.UID())
	delete(sessionsByID, s.ID())
	if s.IsFrontend && s.Subscription != nil {
		// if the user is bound to an userid and nats rpc server is being used we need to unsubscribe
		err := s.Subscription.Unsubscribe()
		if err != nil {
			log.Errorf("error unsubscribing to user's messages channel: %s, this can cause performance and leak issues", err.Error())
		} else {
			log.Debugf("successfully unsubscribed to user's %s messages channel", s.UID())
		}
	}
	s.entity.Close()
}

// RemoteAddr returns the remote network address.
func (s *Session) RemoteAddr() net.Addr {
	return s.entity.RemoteAddr()
}

// Remove delete data associated with the key from session storage
func (s *Session) Remove(key string) error {
	s.Lock()
	defer s.Unlock()

	delete(s.data, key)
	return s.updateEncodedData()
}

// Set associates value with the key in session storage
func (s *Session) Set(key string, value interface{}) error {
	s.Lock()
	defer s.Unlock()

	s.data[key] = value
	return s.updateEncodedData()
}

// SetOnSessionBind sets the method to be called when a session is bound
func SetOnSessionBind(f func(s *Session) error) {
	OnSessionBind = f
}

// HasKey decides whether a key has associated value
func (s *Session) HasKey(key string) bool {
	s.RLock()
	defer s.RUnlock()

	_, has := s.data[key]
	return has
}

// Get returns a key value
func (s *Session) Get(key string) interface{} {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return nil
	}
	return v
}

// Int returns the value associated with the key as a int.
func (s *Session) Int(key string) int {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int)
	if !ok {
		return 0
	}
	return value
}

// Int8 returns the value associated with the key as a int8.
func (s *Session) Int8(key string) int8 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int8)
	if !ok {
		return 0
	}
	return value
}

// Int16 returns the value associated with the key as a int16.
func (s *Session) Int16(key string) int16 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int16)
	if !ok {
		return 0
	}
	return value
}

// Int32 returns the value associated with the key as a int32.
func (s *Session) Int32(key string) int32 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int32)
	if !ok {
		return 0
	}
	return value
}

// Int64 returns the value associated with the key as a int64.
func (s *Session) Int64(key string) int64 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(int64)
	if !ok {
		return 0
	}
	return value
}

// Uint returns the value associated with the key as a uint.
func (s *Session) Uint(key string) uint {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint)
	if !ok {
		return 0
	}
	return value
}

// Uint8 returns the value associated with the key as a uint8.
func (s *Session) Uint8(key string) uint8 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint8)
	if !ok {
		return 0
	}
	return value
}

// Uint16 returns the value associated with the key as a uint16.
func (s *Session) Uint16(key string) uint16 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint16)
	if !ok {
		return 0
	}
	return value
}

// Uint32 returns the value associated with the key as a uint32.
func (s *Session) Uint32(key string) uint32 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint32)
	if !ok {
		return 0
	}
	return value
}

// Uint64 returns the value associated with the key as a uint64.
func (s *Session) Uint64(key string) uint64 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(uint64)
	if !ok {
		return 0
	}
	return value
}

// Float32 returns the value associated with the key as a float32.
func (s *Session) Float32(key string) float32 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(float32)
	if !ok {
		return 0
	}
	return value
}

// Float64 returns the value associated with the key as a float64.
func (s *Session) Float64(key string) float64 {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return 0
	}

	value, ok := v.(float64)
	if !ok {
		return 0
	}
	return value
}

// String returns the value associated with the key as a string.
func (s *Session) String(key string) string {
	s.RLock()
	defer s.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return ""
	}

	value, ok := v.(string)
	if !ok {
		return ""
	}
	return value
}

// Value returns the value associated with the key as a interface{}.
func (s *Session) Value(key string) interface{} {
	s.RLock()
	defer s.RUnlock()

	return s.data[key]
}

func (s *Session) bindInFront() error {
	sessionData := &Data{
		ID:  s.frontendSessionID,
		UID: s.uid,
	}
	b, err := util.GobEncode(sessionData)
	if err != nil {
		return err
	}
	res, err := s.entity.SendRequest(s.frontendID, constants.SessionBindRoute, b)
	if err != nil {
		return err
	}
	log.Debug("session/bindInFront Got response: ", res)
	return nil

}

// PushToFront updates the session in the frontend
func (s *Session) PushToFront() error {
	if s.IsFrontend {
		return constants.ErrFrontSessionCantPushToFront
	}
	sessionData := &Data{
		ID:   s.frontendSessionID,
		UID:  s.uid,
		Data: s.data,
	}
	b, err := util.GobEncode(sessionData)
	if err != nil {
		return err
	}
	res, err := s.entity.SendRequest(s.frontendID, constants.SessionPushRoute, b)
	if err != nil {
		return err
	}
	log.Debug("session/PushToFront Got response: ", res)
	return nil
}

// Clear releases all data related to current session
func (s *Session) Clear() {
	s.Lock()
	defer s.Unlock()

	s.uid = ""
	s.data = map[string]interface{}{}
	s.updateEncodedData()
}
