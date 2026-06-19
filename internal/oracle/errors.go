package oracle

import "errors"

var (
	errOracleNotConfigured = errors.New("external live database network connection is not configured for this scope")
	errOracleUnreachable   = errors.New("external price oracle is currently unreachable")
	errInvalidPrice        = errors.New("oracle calculated an unstable or corrupt price asset baseline")
)
