package pgexpect

// Function defines the function being replaced
type Function struct {
	Name           string
	Args           []Argument
	ReturnType     string
	Body           string
	RaiseErrorCode string
}

// Call defines the calls we are expecting to the function
type Call struct {
	Values []interface{}
}

// Argument define an argument to a function
type Argument struct {
	Name string
	Type string
}

// StubView replace a view with a fake version of it
func StubView(name, body string) {
	panic("NOT IMPLEMENTED")
}
