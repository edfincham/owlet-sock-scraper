package api

type (
	OwletError               struct{ message string }
	OwletConnectionError     struct{ OwletError }
	OwletAuthenticationError struct{ OwletError }
	OwletPasswordError       struct{ OwletError }
	OwletEmailError          struct{ OwletError }
	OwletCredentialsError    struct{ OwletError }
	OwletDevicesError        struct{ OwletError }
)

func (e OwletError) Error() string { return e.message }

func NewOwletError(message string) OwletError {
	return OwletError{message: message}
}

func NewOwletAuthenticationError(message string) OwletAuthenticationError {
	return OwletAuthenticationError{OwletError{message: message}}
}

func NewOwletConnectionError(message string) OwletConnectionError {
	return OwletConnectionError{OwletError{message: message}}
}

func NewOwletPasswordError(message string) OwletPasswordError {
	return OwletPasswordError{OwletError{message: message}}
}

func NewOwletEmailError(message string) OwletEmailError {
	return OwletEmailError{OwletError{message: message}}
}

func NewOwletCredentialsError(message string) OwletCredentialsError {
	return OwletCredentialsError{OwletError{message: message}}
}
