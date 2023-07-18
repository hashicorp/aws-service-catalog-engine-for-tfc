package main

type Error struct {
	errorMessage string `json:"errorMessage"`
	errorType    string `json:"errorString"`
}

func FormatError(errorString string) string {
	//{"errorMessage":"unauthorized","errorType":"errorString"}
	errorString

	providerOverride := &ProviderOverride{}
	err := json.Unmarshal([]byte(entry.FileContents), providerOverride)
	if err != nil {
		t.Error(err)
	}
}
