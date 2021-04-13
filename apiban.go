/*
apiban.go is released under the MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM GmbH
Provides a client for APIBan writen in go.
*/

package baningo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// RootURL the root URL for apiban server
var (
	RootURL = "https://apiban.org/api/"
)

// basic error returned by the API
var (
	ErrRateLimit    = errors.New("rate limit exceeded")
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrEmptyID      = errors.New("empty ID received")
)

const (
	noneID = "none"
	banned = "/banned/"
	check  = "/check/"
)

type apibanObj struct {
	Ipaddress interface{}
	ID        string
}

// GetBannedIPs this function will return all the IPs that are banned by apiban
// in case of a rate limit exceeded will continue with the next apiKey
func GetBannedIPs(ctx context.Context, apiKeys ...string) (IPs []string, err error) {
	ID := "100"
	for _, apiKey := range apiKeys {
		u := RootURL + apiKey + banned
		var newID string
		var nextIPs []string
		for newID != noneID {
			if newID, nextIPs, err = getNextAPIBan(ctx, u+ID); err != nil {
				break
			}
			ID = newID
			IPs = append(IPs, nextIPs...)
		}
		// some error diferent that rate limit
		if err != nil &&
			err != ErrRateLimit {
			return
		}

		// no more data
		if newID == noneID {
			return
		}
	}
	return
}

func getNextAPIBan(ctx context.Context, url string) (ID string, IPs []string, err error) {
	var resp *http.Response
	if resp, err = getHTTP(ctx, url); err != nil {
		return
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
	case resp.StatusCode == http.StatusBadRequest:
	case resp.StatusCode == http.StatusNotFound:
	case resp.StatusCode == http.StatusTooManyRequests:
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		err = fmt.Errorf("client error<%s>", resp.Status)
		return
	case resp.StatusCode >= 500:
		err = fmt.Errorf("server error<%s>", resp.Status)
		return
	default:
		err = fmt.Errorf("unexpected status code<%s>", resp.Status)
		return
	}
	var obj *apibanObj
	if obj, err = decodeObj(resp.Body); err != nil {
		if err == io.EOF &&
			resp.StatusCode >= 400 &&
			resp.StatusCode < 500 {
			err = fmt.Errorf("client error<%s>", resp.Status)
		}
		return
	}
	ID = obj.ID
	if obj.Ipaddress == nil {
		return
	}
	val := obj.Ipaddress.([]interface{})
	if len(val) == 0 {
		return
	}
	if val[0] == "no new bans" {
		return
	}
	IPs = make([]string, len(val))
	for i, v := range val {
		IPs[i] = v.(string)
	}
	return
}

func getHTTP(ctx context.Context, url string) (rsp *http.Response, err error) {
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, "GET", url, nil); err != nil {
		return
	}
	return http.DefaultClient.Do(req)

}

// CheckIP this function will check if the IP is banned by apiban
// in case of a rate limit exceeded will continue with the next apiKey
func CheckIP(ctx context.Context, IP string, apiKeys ...string) (banned bool, err error) {
	if len(IP) == 0 {
		err = errors.New("IP address is required")
		return
	}
	if len(apiKeys) == 0 {
		err = errors.New("API keys are required")
		return
	}
	for _, apiKey := range apiKeys {
		url := RootURL + apiKey + check + IP
		var resp *http.Response
		if resp, err = getHTTP(ctx, url); err != nil {
			return
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 500 ||
			(resp.StatusCode >= 300 && resp.StatusCode < 400) {
			err = fmt.Errorf("unexpected status code<%s>", resp.Status)
			resp.Body.Close()
			return
		}
		var obj *apibanObj
		obj, err = decodeObj(resp.Body)
		resp.Body.Close()
		if err != nil {
			if err != ErrRateLimit {
				return
			}
			continue
		}
		val := obj.Ipaddress.([]interface{})

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if banned = !(obj.ID == noneID ||
				len(val) != 1 ||
				val[0] != IP); !banned {
				err = ErrBadRequest
				return
			}
		} else if obj.ID != noneID ||
			len(val) != 1 ||
			val[0] != "not blocked" {
			err = ErrBadRequest
		}
		return
	}
	return
}

func decodeObj(body io.Reader) (obj *apibanObj, err error) {
	obj = new(apibanObj)
	if err = json.NewDecoder(body).Decode(obj); err != nil {
		return
	}
	switch obj.ID {
	case noneID: // non-error case
	case "unauthorized":
		err = ErrUnauthorized
		return
	case "":
		err = ErrEmptyID
		return
	}
	if val, canCast := obj.Ipaddress.(string); canCast {
		if val == "rate limit exceeded" {
			err = ErrRateLimit
			return
		}
		err = ErrBadRequest
		return
	}
	return
}
