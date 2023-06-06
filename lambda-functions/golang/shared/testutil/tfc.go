package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/testutil/mocking"
	"github.com/hashicorp/go-tfe"
	"log"
)

// MockTFC handles mocking the agent-related TFC APIs. It
// exposes methods for creating server-side state, such as jobs in the queue.
type MockTFC struct {
	Address string

	OrganizationName string

	http *httptest.Server

	// projects is a map of all the projects the mock TFC contains, with their respective id as the keys
	projects map[string]*tfe.Project

	// workspaces is a map of all the workspaces the mock TFC contains, with their respective id as the keys
	workspaces map[string]*tfe.Workspace

	// runs is a map containing the all the runs the mock TFC contains, the keys are the paths for the runs
	runs map[string]*tfe.Run

	// vars is a map of all the vars the mock TFC server contains, the keys are the IDs of the workspaces that own them
	vars map[string][]*tfe.Variable

	retryAfter     int
	retryAfterLock sync.Mutex

	tokenLock sync.Mutex
	token     string

	mockLock     sync.Mutex
	requestMocks mocking.RequestMocks

	fails int32

	flashIndex int
}

func NewMockTFC() *MockTFC {
	mock := &MockTFC{
		OrganizationName: "team-rocket-blast-off",
		projects:         map[string]*tfe.Project{},
		workspaces:       map[string]*tfe.Workspace{},
		runs:             map[string]*tfe.Run{},
		vars:             map[string][]*tfe.Variable{},
	}
	mock.http = httptest.NewServer(mock)
	mock.Address = mock.http.URL
	return mock
}

// SetRetryAfter sets the value which will be used as the Retry-After header.
// This value will be used exactly once and discarded.
func (srv *MockTFC) SetRetryAfter(i int) {
	srv.retryAfterLock.Lock()
	srv.retryAfter = i
	srv.retryAfterLock.Unlock()
}

func (srv *MockTFC) delay() int {
	srv.retryAfterLock.Lock()
	i := srv.retryAfter
	srv.retryAfter = 0
	srv.retryAfterLock.Unlock()

	return i
}

func (srv *MockTFC) SetToken(token string) {
	srv.tokenLock.Lock()
	srv.token = token
	srv.tokenLock.Unlock()
}

func (srv *MockTFC) authToken() string {
	srv.tokenLock.Lock()
	defer srv.tokenLock.Unlock()

	return srv.token
}

// FailRequests makes the mock server return a 500 error for the subsequent
// n requests. This is useful for testing retries.
func (srv *MockTFC) FailRequests(n int32) {
	atomic.StoreInt32(&(srv.fails), n+1)
}

// MockRequest allows for requests to be mocked, which is especially useful if you want to test error cases. The
// predicate that you pass as the first argument can be used to make sure that you don't accidentally end up mocking the
// wrong request, such as the status update requests that agent instances send regularly in the background.
func (srv *MockTFC) MockRequest(predicate mocking.RequestHandlerPredicate, h http.HandlerFunc) {
	srv.mockLock.Lock()
	defer srv.mockLock.Unlock()
	srv.requestMocks = append(srv.requestMocks, mocking.CreateMock(predicate, h))
}

func (srv *MockTFC) checkForMockHandler(r *http.Request) http.HandlerFunc {
	srv.mockLock.Lock()
	defer srv.mockLock.Unlock()
	return mocking.CheckForMock(srv.requestMocks, r)
}

func (srv *MockTFC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Default().Printf("mock TFC server handling request: %s %s", r.Method, r.URL.Path)

	// Check if the request should be handled via mock instead
	if mockHandler := srv.checkForMockHandler(r); mockHandler != nil {
		mockHandler(w, r)
		return
	}

	if n := atomic.AddInt32(&(srv.fails), -1); n > 0 {
		w.WriteHeader(500)
		return
	}

	if code, body := srv.checkBaseRequest(r); code > 0 {
		w.WriteHeader(code)
		w.Write(body)
		return
	}

	switch r.Method {
	case "POST":
		srv.handlePOST(w, r)

	case "GET":
		srv.handleGET(w, r)

	case "PUT":
		srv.handlePUT(w, r)

	default:
		w.WriteHeader(400)
	}
}

func (srv *MockTFC) handlePOST(w http.ResponseWriter, r *http.Request) {
	if srv.HandleProjectsPostRequests(w, r) {
		return
	}
	if srv.HandleWorkspacesPostRequests(w, r) {
		return
	}
	if srv.HandleVarsPostRequests(w, r) {
		return
	}

	// Not found error
	w.WriteHeader(404)
}

func (srv *MockTFC) handleGET(w http.ResponseWriter, r *http.Request) {
	// Handle requests with static paths
	switch r.URL.Path {
	case "/api/v2/ping":
		w.WriteHeader(200)
		return
	}

	if srv.HandleProjectsGetRequests(w, r) {
		return
	}
	if srv.HandleWorkspacesGetRequests(w, r) {
		return
	}
	if srv.HandleRunsRequests(w, r) {
		return
	}
	if srv.HandleVarsGetRequests(w, r) {
		return
	}

	// Not found error
	w.WriteHeader(404)
}

func (srv *MockTFC) handlePUT(w http.ResponseWriter, r *http.Request) {

}

func (srv *MockTFC) checkBaseRequest(r *http.Request) (int, []byte) {
	expectHeaders := []string{
		"User-Agent",
		"Authorization",
	}

	for _, hdr := range expectHeaders {
		if v := r.Header.Get(hdr); v == "" {
			detail := "bad request"
			apiError := struct{ Error string }{detail}
			body, _ := json.Marshal(apiError)
			return 400, body
		}
	}

	// Check the auth token, when present.
	if token := srv.authToken(); token != "" {
		v := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if v != token {
			detail := "Agent token invalid"
			apiError := struct{ Error string }{detail}
			body, _ := json.Marshal(apiError)
			return 401, body
		}
	}

	return 0, []byte{}
}

func (srv *MockTFC) Stop() {
	srv.http.Close()
}
