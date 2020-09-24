/*
apiban.go is released under the MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH
Provides a client for APIBan writen in go.
*/

package baningo

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var (
	bannedResponses = map[string]struct {
		code int
		body []byte
	}{
		"/returnNothing/banned/100":  {code: http.StatusOK, body: []byte{}},
		"/returnNoID/banned/100":     {code: http.StatusOK, body: []byte("{}")},
		"/return500/banned/100":      {code: http.StatusInternalServerError},
		"/badAuth/banned/100":        {code: http.StatusUnauthorized, body: []byte(`{"ID": "unauthorized"}`)},
		"/badAuth2/banned/100":       {code: http.StatusOK, body: []byte(`{"ID": "unauthorized"}`)},
		"/badReq/banned/100":         {code: http.StatusBadRequest, body: []byte(`{"ipaddress": "exceeded","ID":"none"}`)},
		"/testKey3/banned/100":       {code: http.StatusBadRequest, body: []byte(`{"ipaddress": ["no new bans"],"ID":"none"}`)},
		"/testRateLimit/banned/100":  {code: http.StatusTooManyRequests, body: []byte(`{"ipaddress": "rate limit exceeded", "ID":"none"}`)},
		"/testRateLimit2/banned/100": {code: http.StatusBadRequest, body: []byte(`{"ipaddress": "rate limit exceeded", "ID":"none"}`)},
		"/emptyIP/banned/100":        {code: http.StatusOK, body: []byte(`{"ipaddress": [],"ID":"none"}`)},
		"/badStatus/banned/100":      {code: http.StatusNotModified},
	}
	counter      int
	bannedServer = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if val, has := bannedResponses[r.URL.EscapedPath()]; has {
			w.WriteHeader(val.code)
			if val.body != nil {
				w.Write(val.body)
			}
			return
		}
		if r.URL.EscapedPath() != "/testKey/banned/100" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		counter++
		w.WriteHeader(http.StatusOK)
		if counter < 2 {
			_, _ = w.Write([]byte(`{"ipaddress": ["1.2.3.251", "1.2.3.252"], "ID": "100"}`))
		} else {
			_, _ = w.Write([]byte(`{"ID": "none"}`))
			counter = 0
		}
	})

	checkResponses = map[string]struct {
		code int
		body []byte
	}{
		"/testKey/check/1.2.3.251":                               {code: http.StatusOK, body: []byte(`{"ipaddress":["1.2.3.251"], "ID":"987654321"}`)},
		"/testKey/check/1.2.3.254":                               {code: http.StatusBadRequest, body: []byte(`{"ipaddress":["not blocked"], "ID":"none"}`)},
		"/testKey/check/1.2.3.257":                               {code: http.StatusBadRequest, body: []byte(`{"ipaddress":["invalid address"], "ID":"none"}`)},
		"/testKey/check/1000:0000:0000:0000:0000:0000:0000:000g": {code: http.StatusBadRequest, body: []byte(`{"ipaddress":["invalid address"], "ID":"none"}`)},
		"/testKey/check/foo.bar":                                 {code: http.StatusBadRequest, body: []byte(`{"ipaddress":["invalid address"], "ID":"none"}`)},
		"/testRateLimit/check/1.2.3.251":                         {code: http.StatusTooManyRequests, body: []byte(`{"ipaddress": "rate limit exceeded","ID":"none"}`)},
		"/badReq/check/1.2.3.251":                                {code: http.StatusOK, body: []byte(`{"ipaddress":["1.2.3.252"], "ID":"987654321"}`)},
		"/badStatus/check/1.2.3.251":                             {code: http.StatusInternalServerError, body: []byte(`{"ipaddress":["1.2.3.252"], "ID":"987654321"}`)},
	}
	checkServer = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if val, has := checkResponses[r.URL.EscapedPath()]; has {
			w.WriteHeader(val.code)
			if val.body != nil {
				w.Write(val.body)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
)

func TestGetBannedIPs(t *testing.T) {
	// initialize our test server
	testServer := httptest.NewServer(bannedServer)
	defer testServer.Close()

	checkBanned := func(keys, ips []string, expErr error) string {
		result, err := GetBannedIPs(keys...)
		if expErr == nil {
			if err != nil {
				return fmt.Sprintf("Expected error: %s, received: %s", expErr, err)
			}
		} else if err == nil {
			return fmt.Sprintf("Expected error: %s, received: %s", expErr, err)
		} else if expErr.Error() != err.Error() {
			return fmt.Sprintf("Expected error: %s, received: %s", expErr, err)
		}
		if !reflect.DeepEqual(ips, result) {
			return fmt.Sprintf("Expected: %s ,received: %s", ips, result)
		}
		return ""
	}
	RootURL = testServer.URL + "/"

	if m := checkBanned([]string{"testKey"}, []string{"1.2.3.251", "1.2.3.252"}, nil); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{""}, nil, fmt.Errorf("client error<404 Not Found>")); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"badKey"}, nil, fmt.Errorf("client error<404 Not Found>")); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"returnNothing"}, nil, io.EOF); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"testKey3"}, nil, nil); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"returnNoID"}, nil, ErrEmptyID); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"return500"}, nil, fmt.Errorf("server error<500 Internal Server Error>")); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"badAuth"}, nil, fmt.Errorf("client error<401 Unauthorized>")); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"badAuth2"}, nil, ErrUnauthorized); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"testRateLimit"}, nil, fmt.Errorf("client error<429 Too Many Requests>")); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"testRateLimit2"}, nil, ErrRateLimit); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"badReq"}, nil, ErrBadRequest); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"emptyIP"}, nil, nil); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{"badStatus"}, nil, fmt.Errorf("unexpected status code<304 Not Modified>")); len(m) != 0 {
		t.Error(m)
	}
	if m := checkBanned([]string{}, nil, nil); len(m) != 0 {
		t.Error(m)
	}

	RootURL = "http://127.0.0.1:80/"

	if m := checkBanned([]string{"testKey"}, nil, fmt.Errorf(`Get "http://127.0.0.1:80/testKey/banned/100": dial tcp 127.0.0.1:80: connect: connection refused`)); len(m) != 0 {
		t.Error(m)
	}
}

func TestCheck(t *testing.T) {
	// initialize our test server
	testServer := httptest.NewServer(checkServer)
	defer testServer.Close()

	check := func(keys []string, ip string, expErr error, exp bool) string {
		result, err := CheckIP(ip, keys...)
		if expErr == nil {
			if err != nil {
				return fmt.Sprintf("Expected error: %s, received: %s", expErr, err)
			}
		} else if err == nil {
			return fmt.Sprintf("Expected error: %s, received: %s", expErr, err)
		} else if expErr.Error() != err.Error() {
			return fmt.Sprintf("Expected error: %s, received: %s", expErr, err)
		}
		if exp != result {
			return fmt.Sprintf("Expected: %v ,received: %v", exp, result)
		}
		return ""
	}

	RootURL = testServer.URL + "/"

	if m := check([]string{"testKey"}, "1.2.3.251", nil, true); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{"testKey"}, "1.2.3.254", nil, false); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{}, "1.2.3.251", errors.New("API keys are required"), false); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{"testKey"}, "", errors.New("IP address is required"), false); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{"testKey"}, "1.2.3.257", ErrBadRequest, false); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{"testKey"}, "foo.bar", ErrBadRequest, false); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{"testKey"}, "1000:0000:0000:0000:0000:0000:0000:000g", ErrBadRequest, false); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{"badKey"}, "1.2.3.251", io.EOF, false); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{"testRateLimit"}, "1.2.3.251", ErrRateLimit, false); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{"badReq"}, "1.2.3.251", ErrBadRequest, false); len(m) != 0 {
		t.Error(m)
	}
	if m := check([]string{"badStatus"}, "1.2.3.251", fmt.Errorf("unexpected status code<500 Internal Server Error>"), false); len(m) != 0 {
		t.Error(m)
	}

	RootURL = "http://127.0.0.1:80/"

	if m := check([]string{"testKey"}, "1.2.3.251", fmt.Errorf(`Get "http://127.0.0.1:80/testKey/check/1.2.3.251": dial tcp 127.0.0.1:80: connect: connection refused`), false); len(m) != 0 {
		t.Error(m)
	}
}
