// Copyright (C) 2015 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package discover

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/http2"

	"github.com/syncthing/syncthing/internal/slogutil"
	"github.com/syncthing/syncthing/lib/connections/registry"
	"github.com/syncthing/syncthing/lib/dialer"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
)

type globalClient struct {
	// Embedded fields should be listed first
	errorHolder
	// Regular fields
	server         string
	addrList       AddressLister
	announceClient httpClient
	queryClient    httpClient
	noAnnounce     bool
	noLookup       bool
	evLogger       events.Logger
	// Add circuit breaker for server communication
	circuitBreaker *circuitBreaker
	// Add backoff for retry logic
	backoff        *exponentialBackoff
}

type httpClient interface {
	Get(ctx context.Context, url string) (*http.Response, error)
	Post(ctx context.Context, url, ctype string, data io.Reader) (*http.Response, error)
}

const (
	defaultReannounceInterval             = 30 * time.Minute
	announceErrorRetryInterval            = 5 * time.Minute
	requestTimeout                        = 30 * time.Second
	maxAddressChangesBetweenAnnouncements = 10
	// Cache TTL constants
	// defaultCacheTTL                       = 10 * time.Minute
	// negativeCacheTTL                      = 2 * time.Minute
	// Circuit breaker constants
	circuitBreakerFailureThreshold        = 5
	circuitBreakerRetryTimeout            = 1 * time.Minute
)

// circuitBreaker implements a simple circuit breaker pattern
type circuitBreaker struct {
	mut           sync.Mutex
	failureCount  int
	lastFailure   time.Time
	timeout       time.Duration
	failureThreshold int
	open          bool
}

func newCircuitBreaker(failureThreshold int, timeout time.Duration) *circuitBreaker {
	return &circuitBreaker{
		failureThreshold: failureThreshold,
		timeout:          timeout,
	}
}

// Call executes the given function with circuit breaker protection
func (cb *circuitBreaker) Call(fn func() error) error {
	cb.mut.Lock()
	
	// Check if circuit breaker is open
	if cb.open {
		// Check if we should try to close it
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.open = false
			cb.failureCount = 0
		} else {
			cb.mut.Unlock()
			return errors.New("circuit breaker is open")
		}
	}
	
	cb.mut.Unlock()
	
	// Execute the function
	err := fn()
	
	cb.mut.Lock()
	defer cb.mut.Unlock()
	
	if err != nil {
		cb.failureCount++
		cb.lastFailure = time.Now()
		
		// Check if we should open the circuit breaker
		if cb.failureCount >= cb.failureThreshold {
			cb.open = true
			slog.Warn("Circuit breaker opened due to repeated failures", 
				"failureCount", cb.failureCount,
				"threshold", cb.failureThreshold)
		}
	} else {
		// Success, reset failure count
		cb.failureCount = 0
	}
	
	return err
}

type announcement struct {
	Addresses []string `json:"addresses"`
}

func (a announcement) MarshalJSON() ([]byte, error) {
	type announcementCopy announcement

	a.Addresses = sanitizeRelayAddresses(a.Addresses)

	aCopy := announcementCopy(a)
	return json.Marshal(aCopy)
}

type serverOptions struct {
	insecure   bool   // don't check certificate
	noAnnounce bool   // don't announce
	noLookup   bool   // don't use for lookups
	id         string // expected server device ID
}

// A lookupError is any other error but with a cache validity time attached.
type lookupError struct {
	msg      string
	cacheFor time.Duration
}

func (e *lookupError) Error() string { return e.msg }

func (e *lookupError) CacheFor() time.Duration {
	return e.cacheFor
}

func NewGlobal(server string, cert tls.Certificate, addrList AddressLister, evLogger events.Logger, registry *registry.Registry) (FinderService, error) {
	server, opts, err := parseOptions(server)
	if err != nil {
		return nil, err
	}

	var devID protocol.DeviceID
	if opts.id != "" {
		devID, err = protocol.DeviceIDFromString(opts.id)
		if err != nil {
			return nil, err
		}
	}

	// The http.Client used for announcements. It needs to have our
	// certificate to prove our identity, and may or may not verify the server
	// certificate depending on the insecure setting.
	var dialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	if registry != nil {
		dialContext = dialer.DialContextReusePortFunc(registry)
	} else {
		dialContext = dialer.DialContext
	}
	var announceClient httpClient = &contextClient{&http.Client{
		Timeout: requestTimeout,
		Transport: http2EnabledTransport(&http.Transport{
			DialContext:       dialContext,
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true, // announcements are few and far between, so don't keep the connection open
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.insecure,
				Certificates:       []tls.Certificate{cert},
				MinVersion:         tls.VersionTLS12,
				ClientSessionCache: tls.NewLRUClientSessionCache(0),
			},
		}),
	}}
	if opts.id != "" {
		announceClient = newIDCheckingHTTPClient(announceClient, devID)
	}

	// The http.Client used for queries. We don't need to present our
	// certificate here, so lets not include it. May be insecure if requested.
	var queryClient httpClient = &contextClient{&http.Client{
		Timeout: requestTimeout,
		Transport: http2EnabledTransport(&http.Transport{
			DialContext:     dialer.DialContext,
			Proxy:           http.ProxyFromEnvironment,
			IdleConnTimeout: time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.insecure,
				MinVersion:         tls.VersionTLS12,
				ClientSessionCache: tls.NewLRUClientSessionCache(0),
			},
		}),
	}}
	if opts.id != "" {
		queryClient = newIDCheckingHTTPClient(queryClient, devID)
	}

	cl := &globalClient{
		server:         server,
		addrList:       addrList,
		announceClient: announceClient,
		queryClient:    queryClient,
		noAnnounce:     opts.noAnnounce,
		noLookup:       opts.noLookup,
		evLogger:       evLogger,
		circuitBreaker: newCircuitBreaker(circuitBreakerFailureThreshold, circuitBreakerRetryTimeout),
		backoff:        newExponentialBackoff(5, 1*time.Second, 30*time.Second),
	}
	if !opts.noAnnounce {
		// If we are supposed to announce, it's an error until we've done so.
		cl.setError(errors.New("not announced"))
	}

	return cl, nil
}

// Lookup returns the list of addresses where the given device is available
func (c *globalClient) Lookup(ctx context.Context, device protocol.DeviceID) (addresses []string, err error) {
	if c.noLookup {
		return nil, &lookupError{
			msg:      "lookups not supported",
			cacheFor: time.Hour,
		}
	}

	qURL, err := url.Parse(c.server)
	if err != nil {
		return nil, err
	}

	q := qURL.Query()
	q.Set("device", device.String())
	qURL.RawQuery = q.Encode()

	// Use circuit breaker for lookup requests
	var resp *http.Response
	err = c.circuitBreaker.Call(func() error {
		var innerErr error
		resp, innerErr = c.queryClient.Get(ctx, qURL.String())
		if innerErr != nil {
			slog.DebugContext(ctx, "globalClient.Lookup", "url", qURL, slogutil.Error(innerErr))
			return innerErr
		}
		return nil
	})

	if err != nil {
		// Use exponential backoff for retry delay on lookup failures
		delay := c.backoff.NextDelay()
		slog.DebugContext(ctx, "Using exponential backoff for lookup retry", "delay", delay)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.DebugContext(ctx, "globalClient.Lookup", "url", qURL, "status", resp.Status)
		err := errors.New(resp.Status)
		if secs, atoiErr := strconv.Atoi(resp.Header.Get("Retry-After")); atoiErr == nil && secs > 0 {
			err = &lookupError{
				msg:      resp.Status,
				cacheFor: time.Duration(secs) * time.Second,
			}
		}
		return nil, err
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ann announcement
	err = json.Unmarshal(bs, &ann)
	return ann.Addresses, err
}

func (c *globalClient) String() string {
	return "global@" + c.server
}

func (c *globalClient) Serve(ctx context.Context) error {
	if c.noAnnounce {
		// We're configured to not do announcements, only lookups. To maintain
		// the same interface, we just pause here if Serve() is run.
		<-ctx.Done()
		return ctx.Err()
	}

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	eventSub := c.evLogger.Subscribe(events.ListenAddressesChanged)
	defer eventSub.Unsubscribe()

	timerResetCount := 0

	for {
		select {
		case <-eventSub.C():
			if timerResetCount < maxAddressChangesBetweenAnnouncements {
				// Defer announcement by 2 seconds, essentially debouncing
				// if we have a stream of events incoming in quick succession.
				timer.Reset(2 * time.Second)
			} else if timerResetCount == maxAddressChangesBetweenAnnouncements {
				// Yet only do it if we haven't had to reset maxAddressChangesBetweenAnnouncements times in a row,
				// so if something is flip-flopping within 2 seconds, we don't end up in a permanent reset loop.
				slog.ErrorContext(ctx, "Detected a flip-flopping listener", slog.String("server", c.server))
				c.setError(errors.New("flip flopping listener"))
				// Incrementing the count above 10 will prevent us from warning or setting the error again
				// It will also suppress event based resets until we've had a proper round after announceErrorRetryInterval
				timer.Reset(announceErrorRetryInterval)
			}
			timerResetCount++
		case <-timer.C:
			timerResetCount = 0
			c.sendAnnouncement(ctx, timer)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *globalClient) sendAnnouncement(ctx context.Context, timer *time.Timer) {
	var ann announcement
	if c.addrList != nil {
		ann.Addresses = c.addrList.ExternalAddresses()
	}

	if len(ann.Addresses) == 0 {
		// There are legitimate cases for not having anything to announce,
		// yet still using global discovery for lookups. Do not error out
		// here.
		c.setError(nil)
		timer.Reset(announceErrorRetryInterval)
		return
	}

	// The marshal doesn't fail, I promise.
	postData, _ := json.Marshal(ann)

	slog.DebugContext(ctx, "send announcement", "server", c.server, "announcement", ann)

	// Use circuit breaker and exponential backoff for announcement
	var serverRecommendedInterval time.Duration = -1
	err := c.circuitBreaker.Call(func() error {
		resp, err := c.announceClient.Post(ctx, c.server, "application/json", bytes.NewReader(postData))
		if err != nil {
			slog.DebugContext(ctx, "announce POST", "server", c.server, slogutil.Error(err))
			return err
		}
		defer resp.Body.Close()
		
		slog.DebugContext(ctx, "announce POST", "server", c.server, "status", resp.Status)

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			slog.DebugContext(ctx, "announce POST", "server", c.server, "status", resp.Status)
			return errors.New(resp.Status)
		}

		// Check for server-recommended reannouncement time
		if h := resp.Header.Get("Reannounce-After"); h != "" {
			if secs, err := strconv.Atoi(h); err == nil && secs > 0 {
				slog.DebugContext(ctx, "announce sets reannounce-after", "server", c.server, "seconds", secs)
				serverRecommendedInterval = time.Duration(secs) * time.Second
			}
		}

		return nil
	})

	if err != nil {
		slog.WarnContext(ctx, "Failed to send announcement", "server", c.server, "error", err)
		c.setError(err)
		
		// Use exponential backoff for retry delay
		delay := c.backoff.NextDelay()
		slog.DebugContext(ctx, "Using exponential backoff for retry", "delay", delay)
		timer.Reset(delay)
		return
	}

	// Success, reset backoff
	c.backoff.Reset()
	c.setError(nil)

	// Use server-recommended interval if provided, otherwise default
	if serverRecommendedInterval > 0 {
		timer.Reset(serverRecommendedInterval)
	} else {
		timer.Reset(defaultReannounceInterval)
	}
}

func (*globalClient) Cache() map[protocol.DeviceID]CacheEntry {
	// The globalClient doesn't do caching
	return nil
}

// parseOptions parses and strips away any ?query=val options, setting the
// corresponding field in the serverOptions struct. Unknown query options are
// ignored and removed.
func parseOptions(dsn string) (server string, opts serverOptions, err error) {
	p, err := url.Parse(dsn)
	if err != nil {
		return "", serverOptions{}, err
	}

	// Grab known options from the query string
	q := p.Query()
	opts.id = q.Get("id")
	opts.insecure = opts.id != "" || queryBool(q, "insecure")
	opts.noAnnounce = queryBool(q, "noannounce")
	opts.noLookup = queryBool(q, "nolookup")

	// Check for disallowed combinations
	if p.Scheme == "http" {
		if !opts.insecure {
			return "", serverOptions{}, errors.New("http without insecure not supported")
		}
		if !opts.noAnnounce {
			return "", serverOptions{}, errors.New("http without noannounce not supported")
		}
	} else if p.Scheme != "https" {
		return "", serverOptions{}, errors.New("unsupported scheme " + p.Scheme)
	}

	// Remove the query string
	p.RawQuery = ""
	server = p.String()

	return server, opts, err
}

// queryBool returns the query parameter parsed as a boolean. An empty value
// ("?foo") is considered true, as is any value string except false
// ("?foo=false").
func queryBool(q url.Values, key string) bool {
	if _, ok := q[key]; !ok {
		return false
	}

	return q.Get(key) != "false"
}

type idCheckingHTTPClient struct {
	httpClient
	id protocol.DeviceID
}

func newIDCheckingHTTPClient(client httpClient, id protocol.DeviceID) *idCheckingHTTPClient {
	return &idCheckingHTTPClient{
		httpClient: client,
		id:         id,
	}
}

func (c *idCheckingHTTPClient) check(resp *http.Response) error {
	if resp.TLS == nil {
		return errors.New("security: not TLS")
	}

	if len(resp.TLS.PeerCertificates) == 0 {
		return errors.New("security: no certificates")
	}

	id := protocol.NewDeviceID(resp.TLS.PeerCertificates[0].Raw)
	if !id.Equals(c.id) {
		return errors.New("security: incorrect device id")
	}

	return nil
}

func (c *idCheckingHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	resp, err := c.httpClient.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := c.check(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *idCheckingHTTPClient) Post(ctx context.Context, url, ctype string, data io.Reader) (*http.Response, error) {
	resp, err := c.httpClient.Post(ctx, url, ctype, data)
	if err != nil {
		return nil, err
	}
	if err := c.check(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

type errorHolder struct {
	err error
	mut sync.Mutex // uses stdlib sync as I want this to be trivially embeddable, and there is no risk of blocking
}

func (e *errorHolder) setError(err error) {
	e.mut.Lock()
	e.err = err
	e.mut.Unlock()
}

func (e *errorHolder) Error() error {
	e.mut.Lock()
	err := e.err
	e.mut.Unlock()
	return err
}

type contextClient struct {
	*http.Client
}

func (c *contextClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *contextClient) Post(ctx context.Context, url, ctype string, data io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ctype)
	return c.Client.Do(req)
}

type exponentialBackoff struct {
	mut        sync.Mutex
	attempts   int
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
}

func newExponentialBackoff(maxRetries int, baseDelay, maxDelay time.Duration) *exponentialBackoff {
	return &exponentialBackoff{
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		maxDelay:   maxDelay,
	}
}

// NextDelay calculates the next delay based on exponential backoff
func (eb *exponentialBackoff) NextDelay() time.Duration {
	eb.mut.Lock()
	defer eb.mut.Unlock()
	
	if eb.attempts >= eb.maxRetries {
		return eb.maxDelay
	}
	
	delay := time.Duration(float64(eb.baseDelay) * pow(2, float64(eb.attempts)))
	if delay > eb.maxDelay {
		delay = eb.maxDelay
	}
	
	eb.attempts++
	return delay
}

// Reset resets the backoff counter
func (eb *exponentialBackoff) Reset() {
	eb.mut.Lock()
	defer eb.mut.Unlock()
	eb.attempts = 0
}

// pow calculates base^exp efficiently
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func globalDiscoveryIdentity(addr string) string {
	return "global discovery server " + addr
}

func ipv4Identity(port int) string {
	return fmt.Sprintf("IPv4 local broadcast discovery on port %d", port)
}

func ipv6Identity(addr string) string {
	return fmt.Sprintf("IPv6 local multicast discovery on address %s", addr)
}

func http2EnabledTransport(t *http.Transport) *http.Transport {
	_ = http2.ConfigureTransport(t)
	return t
}
