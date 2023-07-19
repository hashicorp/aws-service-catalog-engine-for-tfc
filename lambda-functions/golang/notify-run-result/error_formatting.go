package main

import "encoding/json"

type Error struct {
	ErrorMessage string `json:"errorMessage"`
	ErrorType    string `json:"errorType"`
}

// SimplifyError simplifies errors that come in the form of {"errorMessage":"unauthorized","errorType":"errorString"}, to simply be "unauthorized"
func SimplifyError(errorString string) string {

	errorParsed := &Error{}
	err := json.Unmarshal([]byte(errorString), errorParsed)
	if err != nil {
		// if the error failed to be parsed, return it as is, without any additional formatting, as it was likely parsed previously
		return errorString
	}

	if errorParsed.ErrorType == "errorString" && errorParsed.ErrorMessage != "" {
		// if the errorString is recognized, inject some contextual information
		return errorString
	}

	// not 100% if the error should be formatted or not, so use the original errorString in order to guarantee we don't hide any useful information
	return errorString
}
