package exceptions

type TFEUnauthorized struct {
	Message string
}

func (e TFEUnauthorized) Error() string {
	return e.Message
}

var TFEUnauthorizedToken = TFEUnauthorized{
	Message: "The current authorization token is not valid. Please refer to the instructions in the README https://github.com/hashicorp/aws-service-catalog-engine-for-tfc to reset your authorization token",
}

type TFEException struct {
	Message string
}

func (e TFEException) Error() string {
	return e.Message
}
