/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package mocking

import (
	"net/http"
)

type RequestMock struct {
	predicate RequestHandlerPredicate
	handler   http.HandlerFunc
}

type RequestHandlerPredicate func(r *http.Request) bool

type RequestMocks = []RequestMock

// CreateMock Create a RequestMock
func CreateMock(predicate RequestHandlerPredicate, handler http.HandlerFunc) RequestMock {
	return RequestMock{
		predicate: predicate,
		handler:   handler,
	}
}

// CheckForMock Find the first request mock that's predicate matches the request, or nil if no match is found
func CheckForMock(mocks RequestMocks, r *http.Request) http.HandlerFunc {
	for _, mock := range mocks {
		if mock.predicate(r) {
			return mock.handler
		}
	}
	return nil
}
