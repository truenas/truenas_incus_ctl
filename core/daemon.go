package core

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const TNC_PREFIX_STRING = "tnc_daemon."
const JOB_WAIT_STRING = "core.job_wait"
const DEFAULT_CALL_TIMEOUT = "30s" // also see: cmd.defaultCallTimeout

type TruenasSession struct {
	url             string
	conn            *websocket.Conn
	ctx             *DaemonContext
	sessionKey      string
	channel         int
	connMtx         *sync.Mutex
	callInProgress_ bool
	curCallId_      int64
	callMap_        map[int64]*Future[json.RawMessage]
	jobMap_         map[int64]*Future[json.RawMessage]
}

type DaemonContext struct {
	timeoutValue time.Duration
	timeoutTimer *time.Timer
	mapMtx       *sync.Mutex
	sessionMap_  map[string][]*Future[*TruenasSession]
}

type CallInfo struct {
	method string
	params []interface{}
}

type LoginInfo struct {
	call          CallInfo
	serverUrl     string
	allowInsecure bool
}

func RunDaemon(serverSockAddr string, globalTimeoutStr string) {
	var err error
	var daemonTimeout time.Duration
	if globalTimeoutStr != "" {
		daemonTimeout, err = time.ParseDuration(globalTimeoutStr)
		if err != nil {
			log.Fatal("Error: could not parse duration \"" + globalTimeoutStr + "\":" + err.Error())
		}
	}

	fmt.Println("Serving on", serverSockAddr)
	if daemonTimeout != 0 {
		fmt.Println("With a daemon timeout of", daemonTimeout.String())
	}

	ls, err := net.Listen("unix", serverSockAddr)
	if err != nil {
		fmt.Println("Listen error:", err)
		return
	}

	var timer *time.Timer
	if daemonTimeout != 0 {
		timer = time.NewTimer(daemonTimeout)
	}

	daemon := &DaemonContext{
		timeoutValue: daemonTimeout,
		timeoutTimer: timer,
		mapMtx:       &sync.Mutex{},
		sessionMap_:  make(map[string][]*Future[*TruenasSession]),
	}

	doneCh := make(chan os.Signal)
	signal.Notify(doneCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if daemon.timeoutTimer != nil {
			select {
			case <-daemon.timeoutTimer.C:
				log.Println("tncdaemon timed out (" + daemonTimeout.String() + " elapsed)")
				break
			case <-doneCh:
				log.Println("tncdaemon exiting")
				break
			}
		} else {
			<-doneCh
		}
		_ = ls.Close()
		sessions := make([]*Future[*TruenasSession], 0)
		daemon.mapMtx.Lock()
		for _, futures := range daemon.sessionMap_ {
			sessions = append(sessions, futures...)
		}
		daemon.mapMtx.Unlock()
		for _, future := range sessions {
			_, s, _ := future.Peek()
			if s != nil {
				s.conn.Close()
			}
		}
		os.Remove(serverSockAddr)
		os.Exit(0)
	}()

	http.Serve(ls, daemon)

	doneCh <- syscall.SIGTERM
}

func (d *DaemonContext) UpdateCountdown() {
	if d.timeoutTimer != nil {
		d.timeoutTimer.Reset(d.timeoutValue)
	}
}

func (d *DaemonContext) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	out, err := d.serveImpl(r)
	if err != nil {
		//log.Println(err)
		w.WriteHeader(500)
		io.WriteString(w, err.Error())
	} else {
		//log.Println(string(out))
		w.WriteHeader(200)
		w.Write(out)
	}
}

func (d *DaemonContext) serveImpl(r *http.Request) (json.RawMessage, error) {
	host := r.Header.Get("TNC-Host-Url")
	key := r.Header.Get("TNC-Api-Key")
	user := r.Header.Get("TNC-Username")
	pass := r.Header.Get("TNC-Password")
	method := r.Header.Get("TNC-Call-Method")
	timeoutStr := r.Header.Get("TNC-Timeout")
	allowInsecure := false
	if str := r.Header.Get("TNC-Allow-Insecure"); str != "" {
		allowInsecure = strings.ToLower(str) == "true"
	}

	log.Println("Received request at", r.URL.String(), "for method", method)

	if method == "" {
		return nil, fmt.Errorf("TNC-Call-Method was not provided")
	}

	if method == TNC_PREFIX_STRING+"ping" {
		return []byte("\"pong\""), nil
	}

	if host == "" {
		return nil, fmt.Errorf("TNC-Host-Url was not provided")
	}

	var sessionKey string
	var methodLogin string
	var paramsLogin []interface{}

	if key == "" {
		if user == "" || pass == "" {
			return nil, fmt.Errorf("TNC-Api-Key was not provided, nor TNC-Username nor TNC-Password")
		}
		sessionKey = host + user + pass
		methodLogin = "auth.login"
		paramsLogin = []interface{}{user, pass}
	} else {
		sessionKey = host + key
		methodLogin = "auth.login_with_api_key"
		paramsLogin = []interface{}{key}
	}
	sessionKey += fmt.Sprint(allowInsecure)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var params []interface{}
	if err = json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	call := CallInfo{
		method: method,
		params: params,
	}
	login := LoginInfo{
		call: CallInfo{
			method: methodLogin,
			params: paramsLogin,
		},
		serverUrl:     host,
		allowInsecure: allowInsecure,
	}

retry:
	out, err, shouldRetry := d.maybeCreateSessionAndCall(sessionKey, timeoutStr, call, login)
	if shouldRetry {
		goto retry
	}
	return out, err
}

func (d *DaemonContext) maybeCreateSessionAndCall(sessionKey string, timeoutStr string, call CallInfo, login LoginInfo) (json.RawMessage, error, bool) {
	shouldCreate := false
	channel := -1
	var future *Future[*TruenasSession]

	d.mapMtx.Lock()
	futureSessions, exists := d.sessionMap_[sessionKey]
	if !exists {
		channel = 0
		future = MakeFuture[*TruenasSession]()
		futureSessions = []*Future[*TruenasSession]{future}
		d.sessionMap_[sessionKey] = futureSessions
		shouldCreate = true
	} else {
		for i := 0; i < len(futureSessions); i++ {
			if futureSessions[i] == nil {
				channel = i
				future = MakeFuture[*TruenasSession]()
				futureSessions[i] = future
				d.sessionMap_[sessionKey] = futureSessions
				shouldCreate = true
				break
			}
			done, session, err := futureSessions[i].Peek()
			if !done || err != nil {
				continue
			}
			session.connMtx.Lock()
			if !session.callInProgress_ {
				channel = i
				future = futureSessions[i]
			}
			session.connMtx.Unlock()
			if future != nil {
				break
			}
		}
		if channel < 0 {
			channel = len(futureSessions)
			future = MakeFuture[*TruenasSession]()
			futureSessions = append(futureSessions, future)
			d.sessionMap_[sessionKey] = futureSessions
			shouldCreate = true
		}
	}
	d.mapMtx.Unlock()

	var s *TruenasSession
	var err error

	if shouldCreate {
		s, err = d.createSession(sessionKey, login, channel)
		if err != nil {
			future.Fail(err)
		} else if s == nil {
			err = fmt.Errorf("Failed to launch TrueNAS client session (unknown error)")
			future.Fail(err)
		} else {
			future.Complete(s)
		}
	} else {
		s, err = future.Get()
	}

	//log.Println("Done waiting for session")

	if s == nil {
		if err == nil {
			err = fmt.Errorf("serveImpl: session doesn't exist after waiting for its creation without (unknown error)")
		}
	}
	if err != nil {
		d.deleteSession(sessionKey, channel)
		return nil, err, false
	}

	//log.Println("Calling method", method)

	out, err, shouldRetry := s.callJson(call.method, timeoutStr, call.params)
	if shouldRetry {
		d.deleteSession(sessionKey, channel)
	}

	return out, err, shouldRetry
}

func (d *DaemonContext) createSession(sessionKey string, login LoginInfo, channel int) (*TruenasSession, error) {
	u, err := url.Parse(login.serverUrl)
	if err != nil {
		return nil, fmt.Errorf("Invalid URL: %w", err)
	}

	log.Println("Daemon: creating connection with allowInsecure=" + fmt.Sprint(login.allowInsecure))

	// Configure WebSocket connection with insecure TLS to accept self-signed certs
	dialer := &websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: login.allowInsecure,
		},
		Proxy: http.ProxyFromEnvironment,
	}

	// Establish the WebSocket connection
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect: %w", err)
	}

	session := &TruenasSession{
		url:        login.serverUrl,
		conn:       conn,
		ctx:        d,
		sessionKey: sessionKey,
		channel:    channel,
		connMtx:    &sync.Mutex{},
		curCallId_: 0,
		callMap_:   make(map[int64]*Future[json.RawMessage]),
		jobMap_:    make(map[int64]*Future[json.RawMessage]),
	}

	go session.listen()

	_, err, _ = session.callJson(login.call.method, DEFAULT_CALL_TIMEOUT, login.call.params)
	if err != nil {
		return nil, err
	}

	_, err, _ = session.callJson("core.subscribe", DEFAULT_CALL_TIMEOUT, []interface{}{"core.get_jobs"})
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (d *DaemonContext) deleteSession(sessionKey string, channel int) {
	d.mapMtx.Lock()
	if sessionList, exists := d.sessionMap_[sessionKey]; exists {
		if channel >= 0 && channel < len(sessionList) {
			sessionList[channel] = nil
			d.sessionMap_[sessionKey] = sessionList
		}
	}
	d.mapMtx.Unlock()
}

func (s *TruenasSession) callJson(method string, timeoutStr string, request []interface{}) (json.RawMessage, error, bool) {
	s.ctx.UpdateCountdown()

	if strings.HasPrefix(method, TNC_PREFIX_STRING) {
		out, err := s.handleDaemonProcedure(method[len(TNC_PREFIX_STRING):], timeoutStr, request)
		return out, err, false
	}

	s.connMtx.Lock()
	s.curCallId_++
	callId := s.curCallId_
	fCall := MakeFuture[json.RawMessage]()
	s.callMap_[callId] = fCall
	s.callInProgress_ = true
	s.connMtx.Unlock()

	reqMsg := make(map[string]interface{})
	reqMsg["jsonrpc"] = "2.0"
	reqMsg["method"] = method
	reqMsg["params"] = request
	reqMsg["id"] = callId

	//log.Println("Writing JSON request with callId:", callId, reqMsg)

	if err := wrapWriteJSON(s.conn, reqMsg); err != nil {
		errMsg := err.Error()
		shouldRetry := strings.Contains(errMsg, "use of closed network connection") || strings.Contains(errMsg, "gorilla panic")
		return nil, err, shouldRetry
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = time.Duration(10) * time.Second
	}

	isDone, dataRes, err := AwaitFutureOrTimeout(fCall, timeout)
	if !isDone {
		timeoutParsed := timeout.String()
		return nil, fmt.Errorf("Request timed out (exceeded %s)", timeoutParsed), false
	}

	s.connMtx.Lock()
	delete(s.callMap_, callId)
	s.callInProgress_ = false
	s.connMtx.Unlock()

	if err != nil {
		return nil, err, false
	}

	return dataRes, nil, false
}

func wrapWriteJSON(conn *websocket.Conn, reqMsg interface{}) (err error) {
	err = nil
	defer func() {
		if r := recover(); r != nil {
			msg := "gorilla panic: "
			if rErr, ok := r.(error); ok {
				innerMsg := rErr.Error()
				if innerMsg != "" {
					msg += innerMsg
				}
			} else {
				msg += fmt.Sprint(r)
			}
			err = errors.New(msg)
		}
	}()
	err = conn.WriteJSON(reqMsg)
	return err
}

func (s *TruenasSession) listen() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from panic:", r)
		}
		internalErr := fmt.Errorf("listen() exiting: %v", err)
		_ = s.conn.Close()
		s.connMtx.Lock()
		for _, f := range s.callMap_ {
			f.Fail(internalErr)
		}
		for _, f := range s.jobMap_ {
			f.Fail(internalErr)
		}
		s.connMtx.Unlock()
		s.ctx.deleteSession(s.sessionKey, s.channel)
		log.Println("listen exiting")
	}()
	for true {
		var message json.RawMessage
		_, message, err = s.conn.ReadMessage()
		if err != nil {
			log.Println("listen s.conn.ReadMessage:", err)
			return
		}

		var response interface{}
		if err = json.Unmarshal(message, &response); err != nil {
			continue
		}
		responseMap, ok := response.(map[string]interface{})
		if !ok {
			continue
		}

		s.ctx.UpdateCountdown()

		innerJobId := int64(-1)

		method, _ := responseMap["method"].(string)
		var fields map[string]interface{}

		if method == "collection_update" {
			params, _ := responseMap["params"].(map[string]interface{})
			jobIdF, _ := params["id"].(float64)
			fields, _ = params["fields"].(map[string]interface{})
			state, _ := fields["state"].(string)

			if state == "SUCCESS" || state == "FAILED" {
				innerJobId = int64(jobIdF)
				if innerMethod, _ := fields["method"].(string); innerMethod == JOB_WAIT_STRING {
					if args, ok := fields["arguments"].([]interface{}); ok && len(args) > 0 {
						if value, ok := args[0].(float64); ok {
							innerJobId = int64(value)
						}
					}
				}
			}
		}

		idValue := int64(-1)
		if id, exists := responseMap["id"]; exists {
			idFloat, _ := id.(float64)
			idValue = int64(idFloat)
		}

		var fJob *Future[json.RawMessage]
		var fCall *Future[json.RawMessage]
		var exists bool

		if innerJobId >= 0 || idValue >= 0 {
			s.connMtx.Lock()
			if innerJobId >= 0 {
				fJob, exists = s.jobMap_[innerJobId]
				if !exists {
					fJob = MakeFuture[json.RawMessage]()
					s.jobMap_[innerJobId] = fJob
				}
			}
			if idValue >= 0 {
				fCall, exists = s.callMap_[idValue]
			}
			s.connMtx.Unlock()
		}

		if fJob != nil && fields != nil {
			fJob.Reach(json.Marshal(fields))
		}
		if fCall != nil {
			fCall.Complete(message)
		}
	}
}

func (s *TruenasSession) handleDaemonProcedure(proc string, timeoutStr string, params []interface{}) (json.RawMessage, error) {
	isFirstParamNumber := false
	firstParamAsNumber := int64(0)
	nParams := len(params)
	if nParams > 0 {
		if n, ok := params[0].(float64); ok {
			firstParamAsNumber = int64(n)
			isFirstParamNumber = true
		}
	}

	switch proc {
	case "peek_job":
		if !isFirstParamNumber {
			return nil, fmt.Errorf("tnc_daemon.peek_job expects the first parameter to be a job number")
		}
		fJob := s.getJobFuture(firstParamAsNumber)
		if fJob == nil {
			return nil, fmt.Errorf("Job #%d could not be found internally", firstParamAsNumber)
		}
		isDone, response, err := fJob.Peek()
		if isDone {
			return response, err
		}
		return MakeIncompleteJobStatus(firstParamAsNumber)

	case "await_job":
		if !isFirstParamNumber {
			return nil, fmt.Errorf("tnc_daemon.await_job expects the first parameter to be a job number")
		}

		s.connMtx.Lock()
		fJob, exists := s.jobMap_[firstParamAsNumber]
		if !exists {
			fJob = MakeFuture[json.RawMessage]()
			s.jobMap_[firstParamAsNumber] = fJob
		}
		s.connMtx.Unlock()

		if exists {
			isDone, response, err := fJob.Peek()
			if isDone {
				return response, err
			}
		}

		_, err, _ := s.callJson(JOB_WAIT_STRING, timeoutStr, []interface{}{firstParamAsNumber})
		if err != nil {
			return nil, err
		}

		return fJob.Get()
	}

	return nil, fmt.Errorf("Unrecognised daemon command \"tnc_daemon.%s\"", proc)
}

func (s *TruenasSession) getJobFuture(id int64) *Future[json.RawMessage] {
	s.connMtx.Lock()
	defer s.connMtx.Unlock()
	f, exists := s.jobMap_[id]
	if !exists {
		return nil
	}
	return f
}

func (s *TruenasSession) getCallFuture(id int64) *Future[json.RawMessage] {
	s.connMtx.Lock()
	defer s.connMtx.Unlock()
	f, exists := s.callMap_[id]
	if !exists {
		return nil
	}
	return f
}

func MakeIncompleteJobStatus(jobId int64) (json.RawMessage, error) {
	params := make(map[string]interface{})
	params["arguments"] = []interface{}{jobId}
	params["state"] = "WAITING"
	return json.Marshal(params)
}
