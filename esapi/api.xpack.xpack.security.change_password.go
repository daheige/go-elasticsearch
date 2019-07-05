// Code generated from specification version 6.8.2: DO NOT EDIT

package esapi

import (
	"context"
	"io"
	"net/http"
	"strings"
)

func newXPackSecurityChangePasswordFunc(t Transport) XPackSecurityChangePassword {
	return func(body io.Reader, o ...func(*XPackSecurityChangePasswordRequest)) (*Response, error) {
		var r = XPackSecurityChangePasswordRequest{Body: body}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// XPackSecurityChangePassword - https://www.elastic.co/guide/en/elasticsearch/reference/current/security-api-change-password.html
//
type XPackSecurityChangePassword func(body io.Reader, o ...func(*XPackSecurityChangePasswordRequest)) (*Response, error)

// XPackSecurityChangePasswordRequest configures the Xpack Security  Change Password API request.
//
type XPackSecurityChangePasswordRequest struct {
	Body io.Reader

	Username string

	Refresh string

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do executes the request and returns response or error.
//
func (r XPackSecurityChangePasswordRequest) Do(ctx context.Context, transport Transport) (*Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = "PUT"

	path.Grow(1 + len("_xpack") + 1 + len("security") + 1 + len("user") + 1 + len(r.Username) + 1 + len("_password"))
	path.WriteString("/")
	path.WriteString("_xpack")
	path.WriteString("/")
	path.WriteString("security")
	path.WriteString("/")
	path.WriteString("user")
	if r.Username != "" {
		path.WriteString("/")
		path.WriteString(r.Username)
	}
	path.WriteString("/")
	path.WriteString("_password")

	params = make(map[string]string)

	if r.Refresh != "" {
		params["refresh"] = r.Refresh
	}

	if r.Pretty {
		params["pretty"] = "true"
	}

	if r.Human {
		params["human"] = "true"
	}

	if r.ErrorTrace {
		params["error_trace"] = "true"
	}

	if len(r.FilterPath) > 0 {
		params["filter_path"] = strings.Join(r.FilterPath, ",")
	}

	req, _ := newRequest(method, path.String(), r.Body)

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	if r.Body != nil {
		req.Header[headerContentType] = headerContentTypeJSON
	}

	if len(r.Header) > 0 {
		if len(req.Header) == 0 {
			req.Header = r.Header
		} else {
			for k, vv := range r.Header {
				for _, v := range vv {
					req.Header.Add(k, v)
				}
			}
		}
	}

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	res, err := transport.Perform(req)
	if err != nil {
		return nil, err
	}

	response := Response{
		StatusCode: res.StatusCode,
		Body:       res.Body,
		Header:     res.Header,
	}

	return &response, nil
}

// WithContext sets the request context.
//
func (f XPackSecurityChangePassword) WithContext(v context.Context) func(*XPackSecurityChangePasswordRequest) {
	return func(r *XPackSecurityChangePasswordRequest) {
		r.ctx = v
	}
}

// WithUsername - the username of the user to change the password for.
//
func (f XPackSecurityChangePassword) WithUsername(v string) func(*XPackSecurityChangePasswordRequest) {
	return func(r *XPackSecurityChangePasswordRequest) {
		r.Username = v
	}
}

// WithRefresh - if `true` (the default) then refresh the affected shards to make this operation visible to search, if `wait_for` then wait for a refresh to make this operation visible to search, if `false` then do nothing with refreshes..
//
func (f XPackSecurityChangePassword) WithRefresh(v string) func(*XPackSecurityChangePasswordRequest) {
	return func(r *XPackSecurityChangePasswordRequest) {
		r.Refresh = v
	}
}

// WithPretty makes the response body pretty-printed.
//
func (f XPackSecurityChangePassword) WithPretty() func(*XPackSecurityChangePasswordRequest) {
	return func(r *XPackSecurityChangePasswordRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
//
func (f XPackSecurityChangePassword) WithHuman() func(*XPackSecurityChangePasswordRequest) {
	return func(r *XPackSecurityChangePasswordRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
//
func (f XPackSecurityChangePassword) WithErrorTrace() func(*XPackSecurityChangePasswordRequest) {
	return func(r *XPackSecurityChangePasswordRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
//
func (f XPackSecurityChangePassword) WithFilterPath(v ...string) func(*XPackSecurityChangePasswordRequest) {
	return func(r *XPackSecurityChangePasswordRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
//
func (f XPackSecurityChangePassword) WithHeader(h map[string]string) func(*XPackSecurityChangePasswordRequest) {
	return func(r *XPackSecurityChangePasswordRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}
