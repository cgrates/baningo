/*
apiban.go is released under the MIT License <http://www.opensource.org/licenses/mit-license.php
Copyright (C) ITsysCOM. All Rights Reserved.
Provides a client for APIBan writen in go.
*/

package baningo

import (
	"encoding/json"
	"errors"
	"fmt"
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
)

type apibanObj struct {
	Ipaddress interface{}
	ID        string
}

// GetBannedIPs this function will return all the IPs that are banned by apiban
// in case of a rate limit exceeded will continue with the next apiKey
func GetBannedIPs(apiKeys ...string) (IPs []string, err error) {
	ID := "100"
	for _, apiKey := range apiKeys {
		u := RootURL + apiKey + banned
		var newID string
		var nextIPs []string
		for newID != noneID {
			if newID, nextIPs, err = getNextAPIBan(u + ID); err != nil {
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

func getNextAPIBan(url string) (ID string, IPs []string, err error) {
	var resp *http.Response
	if resp, err = http.Get(url); err != nil {
		return
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode > 400 && resp.StatusCode < 500:
		err = fmt.Errorf("client error<%s>", resp.Status)
		return
	case resp.StatusCode >= 500:
		err = fmt.Errorf("server error<%s>", resp.Status)
		return
	case resp.StatusCode == http.StatusBadRequest:
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
	default:
		err = fmt.Errorf("unexpected status code<%s>", resp.Status)
		return
	}
	obj := new(apibanObj)
	if err = json.NewDecoder(resp.Body).Decode(obj); err != nil {
		return
	}
	switch ID = obj.ID; ID {
	case "none": // non-error case
	case "unauthorized":
		err = ErrUnauthorized
		return
	case "":
		err = ErrEmptyID
		return
	}
	switch val := obj.Ipaddress.(type) {
	case string:
		if val == "rate limit exceeded" {
			err = ErrRateLimit
			return
		}
		err = ErrBadRequest
		return
	case []interface{}:
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
	}
	return
}
