/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package exceptions

type ParserInvalidParameterException struct {
	Message string
}

type ParserAccessDeniedException struct {
	Message string
}

func (e ParserInvalidParameterException) Error() string {
	return e.Message
}

func (e ParserAccessDeniedException) Error() string {
	return e.Message
}
