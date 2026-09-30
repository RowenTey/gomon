//go:build js && wasm

// Package httpclient performs outbound HTTP requests with a real timeout.
//
// It exists because github.com/syumai/workers/cloudflare/fetch ignores the
// request context: its Client.Do calls the JS fetch() without passing an
// AbortSignal, and blocks on the promise without selecting on ctx.Done().
// A context.WithTimeout around it is therefore a no-op, so a slow origin can
// stall a monitoring tick indefinitely.
//
// Here the deadline is handed to the platform's own AbortSignal.timeout(),
// so the runtime cancels the request and rejects the promise with a real
// TimeoutError. Doing the bookkeeping by hand (an AbortController driven by
// setTimeout, or by a Go timer) does not work: setTimeout never returns when
// handed a js.Func, and workerd resolves an aborted fetch with a synthetic
// 200 rather than rejecting it, so an abort looks like a success.
package httpclient

import (
	"errors"
	"fmt"
	"sync"
	"syscall/js"
	"time"
)

// ErrTimeout is returned when a request is aborted because it exceeded its
// timeout. It is distinct from a transport-level failure.
var ErrTimeout = errors.New("request timed out")

// The JS globals are resolved on first use rather than at package-init time.
// Package init runs while the module is still starting up, and touching
// js.Global() from there deadlocks the runtime on the very first invocation.
var (
	globalFetchVal     js.Value
	abortSignalCtorVal js.Value
	globalsOnce        sync.Once
)

func globals() (js.Value, js.Value) {
	globalsOnce.Do(func() {
		global := js.Global()
		globalFetchVal = global.Get("fetch")
		abortSignalCtorVal = global.Get("AbortSignal")
	})
	return globalFetchVal, abortSignalCtorVal
}

// Request describes a single outbound HTTP request.
type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	// Body is sent as-is when non-empty.
	Body string
	// Timeout bounds the whole exchange, including reading the response body.
	// A non-positive value means no timeout.
	Timeout time.Duration
}

// Response is the subset of an HTTP response that gomon needs.
type Response struct {
	StatusCode int
	// Body is populated only when Do was asked to read it.
	Body string
}

// Do performs req, aborting it if it outlives Request.Timeout. When
// readBody is false the response body is cancelled rather than buffered.
func Do(req Request, readBody bool) (Response, error) {
	fetchFunc, abortSignalCtor := globals()

	reqInit := map[string]interface{}{
		"method":  req.Method,
		"headers": toJSObject(req.Headers),
		"signal":  timeoutSignal(abortSignalCtor, req.Timeout),
	}
	if req.Body != "" {
		reqInit["body"] = req.Body
	}

	jsRes, reason, err := awaitPromise(fetchFunc.Invoke(req.URL, js.ValueOf(reqInit)))
	if err != nil {
		if isAbortReason(reason) {
			return Response{}, ErrTimeout
		}
		return Response{}, fmt.Errorf("fetch %s: %w", req.URL, err)
	}

	return readResponse(jsRes, req.URL, readBody)
}

// timeoutSignal returns an AbortSignal that aborts after timeout, or undefined
// when no timeout was requested or the runtime lacks AbortSignal.timeout.
func timeoutSignal(abortSignalCtor js.Value, timeout time.Duration) js.Value {
	if timeout <= 0 || !abortSignalCtor.Truthy() {
		return js.Undefined()
	}
	signal := abortSignalCtor.Call("timeout", timeout.Milliseconds())
	if !signal.Truthy() {
		return js.Undefined()
	}
	return signal
}

// isAbortReason reports whether a rejection is the runtime cancelling the
// request rather than the request itself failing.
func isAbortReason(reason js.Value) bool {
	if reason.IsNull() || reason.IsUndefined() {
		return false
	}
	name := reason.Get("name")
	if !name.Truthy() {
		return false
	}
	switch name.String() {
	case "TimeoutError", "AbortError":
		return true
	default:
		return false
	}
}

func reasonString(reason js.Value) string {
	if reason.IsNull() || reason.IsUndefined() {
		return "undefined"
	}
	return reason.Call("toString").String()
}

// readResponse turns a settled fetch Response into a Response, cancelling the
// body stream when the caller does not want it.
func readResponse(jsRes js.Value, url string, readBody bool) (Response, error) {
	resp := Response{StatusCode: jsRes.Get("status").Int()}

	if readBody {
		text, _, err := awaitPromise(jsRes.Call("text"))
		if err != nil {
			return Response{StatusCode: resp.StatusCode}, fmt.Errorf("read body of %s: %w", url, err)
		}
		resp.Body = text.String()
		return resp, nil
	}

	// Release the connection without paying to buffer a body we discard. The
	// cancel promise is deliberately not awaited: blocking on a second promise
	// here is what wedged the runtime, and nothing depends on the result.
	if body := jsRes.Get("body"); !body.IsNull() && !body.IsUndefined() {
		body.Call("cancel")
	}
	return resp, nil
}

// awaitPromise blocks until promise settles. On fulfilment it returns the
// resolved value; on rejection it returns the rejection reason alongside the
// error, so callers can tell a cancellation from a transport failure.
func awaitPromise(promise js.Value) (js.Value, js.Value, error) {
	resultCh := make(chan js.Value, 1)
	errCh := make(chan js.Value, 1)

	var thenFn, catchFn js.Func
	thenFn = js.FuncOf(func(this js.Value, args []js.Value) any {
		resultCh <- args[0]
		return nil
	})
	catchFn = js.FuncOf(func(this js.Value, args []js.Value) any {
		var reason js.Value
		if len(args) > 0 {
			reason = args[0]
		}
		errCh <- reason
		return nil
	})
	defer thenFn.Release()
	defer catchFn.Release()

	promise.Call("then", thenFn).Call("catch", catchFn)

	select {
	case result := <-resultCh:
		return result, js.Undefined(), nil
	case reason := <-errCh:
		return js.Undefined(), reason, errors.New(reasonString(reason))
	}
}

func toJSObject(values map[string]string) js.Value {
	obj := make(map[string]interface{}, len(values))
	for key, value := range values {
		obj[key] = value
	}
	return js.ValueOf(obj)
}
