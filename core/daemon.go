package core

import (
	"crypto/tls"
	"io"
	"os"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	"strings"
	"syscall"
	"net/url"
	"net/http"
	"os/signal"
	"encoding/json"
	"github.com/gorilla/websocket"
)

const TNC_PREFIX_STRING = "tnc_daemon."
const JOB_WAIT_STRING   = "core.job_wait"

type TruenasSession struct {
	url        string
	conn       *websocket.Conn
	ctx        *DaemonContext
	sessionKey string
	mapMtx     *sync.Mutex
	curCallId_ int64
	callMap_   map[int64]*Future[json.RawMessage]
	jobMap_    map[int64]*Future[json.RawMessage]
}

type DaemonContext struct {
	timeoutValue  time.Duration
	timeoutTimer  *time.Timer
	mapMtx        *sync.Mutex
	sessionMap_   map[string]*Future[*TruenasSession]
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
		log.Fatal(fmt.Sprint("Error:", err))
	}

	var timer *time.Timer
	if daemonTimeout != 0 {
		timer = time.NewTimer(daemonTimeout)
	}

	daemon := &DaemonContext{
		timeoutValue: daemonTimeout,
		timeoutTimer: timer,
		mapMtx: &sync.Mutex{},
		sessionMap_: make(map[string]*Future[*TruenasSession]),
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
		for _, future := range daemon.sessionMap_ {
			sessions = append(sessions, future)
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
	log.Println("Received request at", r.URL.String())
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

	if host == "" {
		return nil, fmt.Errorf("TNC-Host-Url was not provided")
	}
	if method == "" {
		return nil, fmt.Errorf("TNC-Call-Method was not provided")
	}

	var sessionKey string
	var withApiKey bool

	if key == "" {
		if user == "" || pass == "" {
			return nil, fmt.Errorf("TNC-Api-Key was not provided, nor TNC-Username nor TNC-Password")
		}
		sessionKey = host + user + pass
		withApiKey = false
	} else {
		sessionKey = host + key
		withApiKey = true
	}
	sessionKey += fmt.Sprint(allowInsecure)

	shouldCreate := false

	d.mapMtx.Lock()
	future, exists := d.sessionMap_[sessionKey]
	if !exists {
		future = MakeFuture[*TruenasSession]()
		d.sessionMap_[sessionKey] = future
		shouldCreate = true
	}
	d.mapMtx.Unlock()

	var s *TruenasSession
	var err error

	if shouldCreate {
		if withApiKey {
			s, err = d.createSession(allowInsecure, host, "auth.login_with_api_key", []interface{}{key})
		} else {
			s, err = d.createSession(allowInsecure, host, "auth.login", []interface{}{user, pass})
		}
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
		d.mapMtx.Lock()
		delete(d.sessionMap_, sessionKey)
		d.mapMtx.Unlock()
		return nil, err
	}

	//log.Println("Reading request body")

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	//log.Println("Calling method", method)

	return s.call(method, timeoutStr, data)
}

func (d *DaemonContext) createSession(allowInsecure bool, truenasServerUrl string, loginMethod string, loginParams []interface{}) (*TruenasSession, error) {
	u, err := url.Parse(truenasServerUrl)
	if err != nil {
		return nil, fmt.Errorf("Invalid URL: %w", err)
	}

	log.Println("Daemon: creating connection with allowInsecure=" + fmt.Sprint(allowInsecure))

	// Configure WebSocket connection with insecure TLS to accept self-signed certs
	dialer := &websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: allowInsecure,
		},
		Proxy: http.ProxyFromEnvironment,
	}

	// Establish the WebSocket connection
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect: %w", err)
	}

	sessionKey := truenasServerUrl + loginParams[0].(string)
	if len(loginParams) > 1 {
		sessionKey += loginParams[1].(string)
	}

	session := &TruenasSession{
		url: truenasServerUrl,
		conn: conn,
		ctx: d,
		sessionKey: sessionKey,
		mapMtx: &sync.Mutex{},
		curCallId_: 0,
		callMap_: make(map[int64]*Future[json.RawMessage]),
		jobMap_: make(map[int64]*Future[json.RawMessage]),
	}

	go session.listen()

	if _, err = session.callJson(loginMethod, "10s", loginParams); err != nil {
		return nil, err
	}
	if _, err = session.callJson("core.subscribe", "10s", []interface{}{"core.get_jobs"}); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *TruenasSession) call(method string, timeoutStr string, data json.RawMessage) (json.RawMessage, error) {
	var request interface{}
	err := json.Unmarshal(data, &request)
	if err != nil {
		return nil, err
	}

	return s.callJson(method, timeoutStr, request)
}

func (s *TruenasSession) callJson(method string, timeoutStr string, request interface{}) (json.RawMessage, error) {
	s.ctx.UpdateCountdown()

	requestParams, ok := request.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Request body was not a JSON array")
	}

	if strings.HasPrefix(method, TNC_PREFIX_STRING) {
		return s.handleDaemonProcedure(method[len(TNC_PREFIX_STRING):], timeoutStr, requestParams)
	}

	s.mapMtx.Lock()
	s.curCallId_++
	callId := s.curCallId_
	fCall := MakeFuture[json.RawMessage]()
	s.callMap_[callId] = fCall
	s.mapMtx.Unlock()

	reqMsg := make(map[string]interface{})
	reqMsg["jsonrpc"] = "2.0"
	reqMsg["method"] = method
	reqMsg["params"] = requestParams
	reqMsg["id"] = callId

	//log.Println("Writing JSON request with callId:", callId, reqMsg)

	if err := s.conn.WriteJSON(reqMsg); err != nil {
		return nil, err
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = time.Duration(10) * time.Second
	}

	isDone, dataRes, err := AwaitFutureOrTimeout(fCall, timeout)
	if !isDone {
		timeoutParsed := timeout.String()
		return nil, fmt.Errorf("Request timed out (exceeded %s)", timeoutParsed)
	}

	if err != nil {
		return nil, err
	}

	jobId, _ := GetJobNumber(dataRes)

	s.mapMtx.Lock()
	if jobId > 0 {
		s.jobMap_[jobId] = MakeFuture[json.RawMessage]()
	}
	delete(s.callMap_, callId)
	s.mapMtx.Unlock()

	if jobId > 0 && method != JOB_WAIT_STRING {
		_, _ = s.callJson(JOB_WAIT_STRING, timeoutStr, []interface{}{jobId})
	}

	return dataRes, nil
}

func (s *TruenasSession) listen() {
	var err error
	defer func() {
		internalErr := fmt.Errorf("listen() exiting: %v", err)
		s.conn.Close()
		s.mapMtx.Lock()
		for _, f := range s.callMap_ {
			f.Fail(internalErr)
		}
		for _, f := range s.jobMap_ {
			f.Fail(internalErr)
		}
		s.mapMtx.Unlock()
		s.ctx.mapMtx.Lock()
		delete(s.ctx.sessionMap_, s.sessionKey)
		s.ctx.mapMtx.Unlock()
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

		method, _ := responseMap["method"].(string)
		if method == "collection_update" {
			params := responseMap["params"].(map[string]interface{})
			jobId := int64(params["id"].(float64))
			fields := params["fields"].(map[string]interface{})
			state, _ := fields["state"].(string)

			if state == "SUCCESS" || state == "FAILED" {
				innerJobId := jobId
				if innerMethod, _ := fields["method"].(string); innerMethod == JOB_WAIT_STRING {
					if args, ok := fields["arguments"].([]interface{}); ok && len(args) > 0 {
						if value, ok := args[0].(float64); ok {
							innerJobId = int64(value)
						}
					}
				}

				s.mapMtx.Lock()
				fJob, exists := s.jobMap_[innerJobId]
				s.mapMtx.Unlock()
				if exists {
					fJob.Reach(json.Marshal(fields))
				}
			}
		}

		if id, exists := responseMap["id"]; exists {
			idValue, _ := id.(float64)
			s.mapMtx.Lock()
			fCall, exists := s.callMap_[int64(idValue)]
			s.mapMtx.Unlock()
			if exists {
				fCall.Complete(message)
			}
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
		isDone, response, err := fJob.Peek()
		if isDone {
			return response, err
		}
		return MakeIncompleteJobStatus(firstParamAsNumber)
	case "await_job":
		if !isFirstParamNumber {
			return nil, fmt.Errorf("tnc_daemon.await_job expects the first parameter to be a job number")
		}
		fJob := s.getJobFuture(firstParamAsNumber)
		return fJob.Get()
	}

	return nil, fmt.Errorf("Unrecognised daemon command \"tnc_daemon.%s\"", proc)
}

func (s *TruenasSession) getJobFuture(id int64) (*Future[json.RawMessage]) {
	s.mapMtx.Lock()
	defer s.mapMtx.Unlock()
	f, exists := s.jobMap_[id]
	if !exists {
		return nil
	}
	return f
}

func (s *TruenasSession) getCallFuture(id int64) (*Future[json.RawMessage]) {
	s.mapMtx.Lock()
	defer s.mapMtx.Unlock()
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
