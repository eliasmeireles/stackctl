package oidclogin

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
)

//go:embed callback.html
var callbackPage string

var callbackTemplate = template.Must(template.New("callback").Parse(callbackPage))

// exchangeFunc turns the identity provider's redirect parameters into a Vault token.
type exchangeFunc func(params url.Values) (*api.Secret, error)

type loginResult struct {
	secret *api.Secret
	err    error
}

// callbackServer receives the identity provider redirect on a local port,
// lets exchange finish the login and shows the outcome in the browser.
type callbackServer struct {
	listener net.Listener
	server   *http.Server
	host     string
	path     string
	exchange exchangeFunc
	results  chan loginResult
	once     sync.Once
}

// listenCallback binds addr (host:port) and serves path until Close.
func listenCallback(addr, path string, exchange exchangeFunc) (*callbackServer, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid callback address %q: %w", addr, err)
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on %s for the login callback (is another login running?): %w", addr, err)
	}
	s := &callbackServer{listener: listener, host: host, path: path, exchange: exchange, results: make(chan loginResult, 1)}
	s.server = &http.Server{Handler: http.HandlerFunc(s.handle), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = s.server.Serve(listener) }()
	return s, nil
}

// RedirectURI is the URL the identity provider must redirect to.
func (s *callbackServer) RedirectURI() string {
	_, port, _ := net.SplitHostPort(s.listener.Addr().String())
	return "http://" + net.JoinHostPort(s.host, port) + s.path
}

// Wait blocks until the first callback finishes or ctx ends.
func (s *callbackServer) Wait(ctx context.Context) (*api.Secret, error) {
	select {
	case r := <-s.results:
		return r.secret, r.err
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, errors.New("timed out waiting for the browser login")
		}
		return nil, ctx.Err()
	}
}

// Close stops the listener.
func (s *callbackServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *callbackServer) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.path {
		http.NotFound(w, r)
		return
	}
	handled := false
	s.once.Do(func() {
		handled = true
		secret, err := s.complete(r.URL.Query())
		render(w, err)
		s.results <- loginResult{secret: secret, err: err}
	})
	if !handled {
		w.WriteHeader(http.StatusConflict)
		render(w, errors.New("this login was already completed; you can close this window"))
	}
}

func (s *callbackServer) complete(params url.Values) (*api.Secret, error) {
	if code := params.Get("error"); code != "" {
		if desc := params.Get("error_description"); desc != "" {
			return nil, fmt.Errorf("identity provider returned %s: %s", code, desc)
		}
		return nil, fmt.Errorf("identity provider returned %s", code)
	}
	if params.Get("code") == "" || params.Get("state") == "" {
		return nil, errors.New("callback is missing the code or state parameter")
	}
	return s.exchange(params)
}

func render(w http.ResponseWriter, err error) {
	data := struct {
		OK             bool
		Title, Message string
	}{OK: true, Title: "Signed in to Vault", Message: "You can close this window and go back to the terminal."}
	if err != nil {
		data.OK, data.Title, data.Message = false, "Vault login failed", err.Error()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = callbackTemplate.Execute(w, data)
}
