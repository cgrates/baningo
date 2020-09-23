/*
apiban.go is released under the MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM. All Rights Reserved.
Provides a client for APIBan writen in go.
*/

package baningo

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var counter int

var mockServer = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// for whatever reason the application doesn't let you NOT pass an ID (it defaults it to 100 when empty)
	ID := 100

	testPathBanned := fmt.Sprintf("/%s/banned/%d", "testKey", ID)
	testPathBannedReturnNothing := fmt.Sprintf("/%s/banned/%d", "returnNothing", ID)
	testPathBannedReturnNoID := fmt.Sprintf("/%s/banned/%d", "returnNoID", ID)
	testPathBannedReturn400 := fmt.Sprintf("/%s/banned/%s", "testKey", "badInput")
	testPathBannedReturn500 := fmt.Sprintf("/%s/banned/%d", "return500", ID)
	testPathBannedBadAuth := fmt.Sprintf("/%s/banned/%d", "badAuth", ID)
	testPathBannedBadAuth2 := fmt.Sprintf("/%s/banned/%d", "badAuth2", ID)
	testPathBannedNothingNew := fmt.Sprintf("/%s/banned/%d", "testKey3", ID)
	testPathBannedRateLimit := fmt.Sprintf("/%s/banned/%d", "testRateLimit", ID)
	testPathBannedRateLimit2 := fmt.Sprintf("/%s/banned/%d", "testRateLimit2", ID)
	testPathBannedBadReq := fmt.Sprintf("/%s/banned/%d", "badReq", ID)
	testPathBannedEmptyIPs := fmt.Sprintf("/%s/banned/%d", "emptyIP", ID)
	testPathBannedBadStatus := fmt.Sprintf("/%s/banned/%d", "badStatus", ID)

	switch r.URL.EscapedPath() {
	case testPathBannedReturnNothing:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(""))
		return
	case testPathBannedReturnNoID:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
		return
	case testPathBannedReturn400:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{}"))
		return
	case testPathBannedReturn500:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case testPathBannedBadAuth:
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("{\"ID\": \"unauthorized\"}"))
		return
	case testPathBannedBadAuth2:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"ID\": \"unauthorized\"}"))
		return
	case testPathBannedNothingNew:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{\"ipaddress\":[\"no new bans\"], \"ID\":\"none\"}"))
		return
	case testPathBannedRateLimit:
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("{\"ipaddress: \"rate limit exceeded\"}"))
		return
	case testPathBannedRateLimit2:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{\"ipaddress\": \"rate limit exceeded\",\"ID\":\"none\"}"))
		return
	case testPathBannedBadReq:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{\"ipaddress\": \"exceeded\",\"ID\":\"none\"}"))
		return
	case testPathBannedEmptyIPs:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"ipaddress\": [],\"ID\":\"none\"}"))
		return
	case testPathBannedBadStatus:
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if r.URL.EscapedPath() != testPathBanned {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	counter++
	w.WriteHeader(http.StatusOK)
	if counter < 2 {
		_, _ = w.Write([]byte(fmt.Sprintf("{\"ipaddress\": [\"1.2.3.251\", \"1.2.3.252\"], \"ID\": \"%d\"}", ID)))
	} else {
		_, _ = w.Write([]byte("{\"ID\": \"none\"}"))
		counter = 0
	}
})

func TestGetBannedIPs(t *testing.T) {
	// initialize our test server
	testServer := httptest.NewServer(mockServer)
	defer testServer.Close()

	type mockInput struct {
		keys        []string
		badEndpoint bool
	}

	type mockOutput struct {
		ips []string
		err error
	}

	testCases := map[string]struct {
		input    mockInput
		expected mockOutput
	}{
		"succesful lookup": {
			input: mockInput{
				keys: []string{"testKey"},
			},
			expected: mockOutput{
				ips: []string{"1.2.3.251", "1.2.3.252"},
			},
		},
		"succesful with ID": {
			input: mockInput{
				keys: []string{"testKey"},
			},
			expected: mockOutput{
				ips: []string{"1.2.3.251", "1.2.3.252"},
			},
		},
		"no key": {
			input: mockInput{keys: []string{""}},
			expected: mockOutput{
				err: fmt.Errorf("client error<404 Not Found>"),
			},
		},
		"unknown key": {
			input: mockInput{
				keys: []string{"badKey"},
			},
			expected: mockOutput{
				err: fmt.Errorf("client error<404 Not Found>"),
			},
		},
		"unreachable destination": {
			input: mockInput{
				keys:        []string{"testKey"},
				badEndpoint: true,
			},
			expected: mockOutput{
				err: fmt.Errorf(`Get "http://127.0.0.1:80/testKey/banned/100": dial tcp 127.0.0.1:80: connect: connection refused`),
			},
		},
		"nothing returned": {
			input: mockInput{
				keys: []string{"returnNothing"},
			},
			expected: mockOutput{
				err: io.EOF,
			},
		},
		"no new bans returned": {
			input: mockInput{
				keys: []string{"testKey3"},
			},
		},
		"no id returned": {
			input: mockInput{
				keys: []string{"returnNoID"},
			},
			expected: mockOutput{
				err: fmt.Errorf("empty ID received"),
			},
		},
		"Simulate unknown server error": {
			input: mockInput{
				keys: []string{"return500"},
			},
			expected: mockOutput{
				err: fmt.Errorf("server error<500 Internal Server Error>"),
			},
		},
		"Simulate bad auth": {
			input: mockInput{
				keys: []string{"badAuth"},
			},
			expected: mockOutput{
				err: fmt.Errorf("client error<401 Unauthorized>"),
			},
		},
		"Simulate bad auth 2": {
			input: mockInput{
				keys: []string{"badAuth2"},
			},
			expected: mockOutput{
				err: fmt.Errorf("unauthorized"),
			},
		},
		"simulate rate limiter": {
			input: mockInput{
				keys: []string{"testRateLimit"},
			},
			expected: mockOutput{
				err: fmt.Errorf("client error<429 Too Many Requests>"),
			},
		},
		"simulate rate limiter2": {
			input: mockInput{
				keys: []string{"testRateLimit2"},
			},
			expected: mockOutput{
				err: ErrRateLimit,
			},
		},
		"simulate bad request": {
			input: mockInput{
				keys: []string{"badReq"},
			},
			expected: mockOutput{
				err: ErrBadRequest,
			},
		},
		"simulate empty IPs": {
			input: mockInput{
				keys: []string{"emptyIP"},
			},
		},
		"simulate bad status": {
			input: mockInput{
				keys: []string{"badStatus"},
			},
			expected: mockOutput{
				err: fmt.Errorf("unexpected status code<304 Not Modified>"),
			},
		},
		"no keys": {},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			RootURL = fmt.Sprintf("%s/", testServer.URL)
			if tc.input.badEndpoint {
				RootURL = "http://127.0.0.1:80/"
			}
			result, err := GetBannedIPs(tc.input.keys...)
			if tc.expected.err == nil {
				if err != nil {
					t.Errorf("Expected error: %s, received: %s", tc.expected.err, err)
				}
			} else if err == nil {
				t.Errorf("Expected error: %s, received: %s", tc.expected.err, err)
			} else if tc.expected.err.Error() != err.Error() {
				t.Errorf("Expected error: %s, received: %s", tc.expected.err, err)
			}
			if !reflect.DeepEqual(tc.expected.ips, result) {
				t.Errorf("Expected: %s ,received: %s", tc.expected.ips, result)
			}
		})
	}
}
