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
	"context"
	"encoding/json"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
)

// NetworkEntity represent low-level network instance
type NetworkEntity interface {
	Push(route string, v interface{}) error
	ResponseMID(ctx context.Context, mid uint, v interface{}, isError ...bool) error
	Close() error
	Kick(ctx context.Context) error
	RemoteAddr() net.Addr
	SendRequest(ctx context.Context, serverID, route string, v interface{}) (*protos.Response, error)
}

var (
	sessionBindCallbacks = make([]func(ctx context.Context, s *Session) error, 0)
	afterBindCallbacks   = make([]func(ctx context.Context, s *Session) error, 0)
	// SessionCloseCallbacks contains global session close callbacks
	SessionCloseCallbacks = make([]func(s *Session), 0)
	sessionsByUID         sync.Map
	sessionsByID          sync.Map
	sessionIDSvc          = newSessionIDService()
	// SessionCount keeps the current number of sessions
	SessionCount int64
)

// HandshakeClientData represents information about the client sent on the handshake.
type HandshakeClientData struct {
	Platform    string `json:"platform"`
	LibVersion  string `json:"libVersion"`
	BuildNumber string `json:"clientBuildNumber"`
	Version     string `json:"clientVersion"`
}

// HandshakeData represents information about the handshake sent by the client.
// `sys` corresponds to information independent from the app and `user` information
// that depends on the app and is customized by the user.
type HandshakeData struct {
	Sys  HandshakeClientData    `json:"sys"`
	User map[string]interface{} `json:"user,omitempty"`
}

// Session represents a client session, which can store data during the connection.
// All data is released when the low-level connection is broken.
// Session instance related to the client will be passed to Handler method in the
// context parameter.
type Session struct {
	sync.RWMutex                             // protect data
	id                int64                  // session global unique id
	uid               string                 // binding user id
	lastTime          int64                  // last heartbeat time
	entity            NetworkEntity          // low-level network entity
	data              map[string]interface{} // session data store
	handshakeData     *HandshakeData         // handshake data received by the client
	encodedData       []byte                 // session data encoded as a byte array
	OnCloseCallbacks  []func()               //onClose callbacks
	IsFrontend        bool                   // if session is a frontend session
	frontendID        string                 // the id of the frontend that owns the session
	frontendSessionID int64                  // the id of the session on the frontend server
	Subscriptions     []*nats.Subscription   // subscription created on bind when using nats rpc server
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
func New(entity NetworkEntity, frontend bool, UID ...string) *Session {
	s := &Session{
		id:               sessionIDSvc.sessionID(),
		entity:           entity,
		data:             make(map[string]interface{}),
		handshakeData:    nil,
		lastTime:         time.Now().Unix(),
		OnCloseCallbacks: []func(){},
		IsFrontend:       frontend,
	}
	if frontend {
		sessionsByID.Store(s.id, s)
		atomic.AddInt64(&SessionCount, 1)
	}
	if len(UID) > 0 {
		s.uid = UID[0]
	}
	return s
}

// GetSessionByUID return a session bound to an user id
func GetSessionByUID(uid string) *Session {
	// TODO: Block this operation in backend servers
	if val, ok := sessionsByUID.Load(uid); ok {
		return val.(*Session)
	}
	return nil
}

// GetSessionByID return a session bound to a frontend server id
func GetSessionByID(id int64) *Session {
	// TODO: Block this operation in backend servers
	if val, ok := sessionsByID.Load(id); ok {
		return val.(*Session)
	}
	return nil
}

// OnSessionBind adds a method to be called when a session is bound
// same function cannot be added twice!
func OnSessionBind(f func(ctx context.Context, s *Session) error) {
	// Prevents the same function to be added twice in onSessionBind
	sf1 := reflect.ValueOf(f)
	for _, fun := range sessionBindCallbacks {
		sf2 := reflect.ValueOf(fun)
		if sf1.Pointer() == sf2.Pointer() {
			return
		}
	}
	sessionBindCallbacks = append(sessionBindCallbacks, f)
}

// OnAfterSessionBind adds a method to be called when session is bound and after all sessionBind callbacks
func OnAfterSessionBind(f func(ctx context.Context, s *Session) error) {
	// Prevents the same function to be added twice in onSessionBind
	sf1 := reflect.ValueOf(f)
	for _, fun := range afterBindCallbacks {
		sf2 := reflect.ValueOf(fun)
		if sf1.Pointer() == sf2.Pointer() {
			return
		}
	}
	afterBindCallbacks = append(afterBindCallbacks, f)
}

// OnSessionClose adds a method that will be called when every session closes
func OnSessionClose(f func(s *Session)) {
	sf1 := reflect.ValueOf(f)
	for _, fun := range SessionCloseCallbacks {
		sf2 := reflect.ValueOf(fun)
		if sf1.Pointer() == sf2.Pointer() {
			return
		}
	}
	SessionCloseCallbacks = append(SessionCloseCallbacks, f)
}

// CloseAll calls Close on all sessions
func CloseAll() {
	logger.Log.Debugf("closing all sessions, %d sessions", SessionCount)
	sessionsByID.Range(func(_, value interface{}) bool {
		s := value.(*Session)
		s.Close()
		return true
	})
	logger.Log.Debug("finished closing sessions")
}

func (s *Session) updateEncodedData() error {
	var b []byte
	b, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	s.encodedData = b
	return nil
}

// Push message to client
func (s *Session) Push(route string, v interface{}) error {
	return s.entity.Push(route, v)
}

// ResponseMID responses message to client, mid is
// request message ID
func (s *Session) ResponseMID(ctx context.Context, mid uint, v interface{}, err ...bool) error {
	return s.entity.ResponseMID(ctx, mid, v, err...)
}

// ID returns the session id
func (s *Session) ID() int64 {
	return s.id
}

// UID returns uid that bind to current session
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
	err := json.Unmarshal(encodedData, &data)
	if err != nil {
		return err
	}
	return s.SetData(data)
}

// SetFrontendData sets frontend id and session id
func (s *Session) SetFrontendData(frontendID string, frontendSessionID int64) {
	s.frontendID = frontendID
	s.frontendSessionID = frontendSessionID
}

// Bind bind UID to current session
func (s *Session) Bind(ctx context.Context, uid string) error {
	if uid == "" {
		return constants.ErrIllegalUID
	}

	if s.UID() != "" {
		return constants.ErrSessionAlreadyBound
	}

	s.uid = uid
	for _, cb := range sessionBindCallbacks {
		err := cb(ctx, s)
		if err != nil {
			s.uid = ""
			return err
		}
	}

	for _, cb := range afterBindCallbacks {
		err := cb(ctx, s)
		if err != nil {
			s.uid = ""
			return err
		}
	}

	// if code running on frontend server
	if s.IsFrontend {
		sessionsByUID.Store(uid, s)
	} else {
		// If frontentID is set this means it is a remote call and the current server
		// is not the frontend server that received the user request
		err := s.bindInFront(ctx)
		if err != nil {
			logger.Log.Error("error while trying to push session to front: ", err)
			s.uid = ""
			return err
		}
	}
	return nil
}

// Kick kicks the user
func (s *Session) Kick(ctx context.Context) error {
	err := s.entity.Kick(ctx)
	if err != nil {
		return err
	}
	return s.entity.Close()
}

// OnClose adds the function it receives to the callbacks that will be called
// when the session is closed
func (s *Session) OnClose(c func()) error {
	if !s.IsFrontend {
		return constants.ErrOnCloseBackend
	}
	s.OnCloseCallbacks = append(s.OnCloseCallbacks, c)
	return nil
}

// Close terminates current session, session related data will not be released,
// all related data should be cleared explicitly in Session closed callback
func (s *Session) Close() {
	atomic.AddInt64(&SessionCount, -1)
	sessionsByID.Delete(s.ID())
	sessionsByUID.Delete(s.UID())
	// TODO: this logic should be moved to nats rpc server
	if s.IsFrontend && s.Subscriptions != nil && len(s.Subscriptions) > 0 {
		// if the user is bound to an userid and nats rpc server is being used we need to unsubscribe
		for _, sub := range s.Subscriptions {
			err := sub.Unsubscribe()
			if err != nil {
				logger.Log.Errorf("error unsubscribing to user's messages channel: %s, this can cause performance and leak issues", err.Error())
			} else {
				logger.Log.Debugf("successfully unsubscribed to user's %s messages channel", s.UID())
			}
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

func (s *Session) bindInFront(ctx context.Context) error {
	return s.sendRequestToFront(ctx, constants.SessionBindRoute, false)
}

// PushToFront updates the session in the frontend
func (s *Session) PushToFront(ctx context.Context) error {
	if s.IsFrontend {
		return constants.ErrFrontSessionCantPushToFront
	}
	return s.sendRequestToFront(ctx, constants.SessionPushRoute, true)
}

// Clear releases all data related to current session
func (s *Session) Clear() {
	s.Lock()
	defer s.Unlock()

	s.uid = ""
	s.data = map[string]interface{}{}
	s.updateEncodedData()
}

// SetHandshakeData sets the handshake data received by the client.
func (s *Session) SetHandshakeData(data *HandshakeData) {
	s.Lock()
	defer s.Unlock()

	s.handshakeData = data
}

// GetHandshakeData gets the handshake data received by the client.
func (s *Session) GetHandshakeData() *HandshakeData {
	return s.handshakeData
}

func (s *Session) sendRequestToFront(ctx context.Context, route string, includeData bool) error {
	sessionData := &protos.Session{
		Id:  s.frontendSessionID,
		Uid: s.uid,
	}
	if includeData {
		sessionData.Data = s.encodedData
	}
	b, err := proto.Marshal(sessionData)
	if err != nil {
		return err
	}
	res, err := s.entity.SendRequest(ctx, s.frontendID, route, b)
	if err != nil {
		return err
	}
	logger.Log.Debugf("%s Got response: %+v", route, res)
	return nil
}
