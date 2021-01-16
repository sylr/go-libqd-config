package config

// Logger is the interface that describes the logger used by this module.
// You can wrap any logger lib that you alreadey use in a struct that comply
// with this interface if your logger does not already implement this interface.
// For instance github.com/sirupsen/logrus is already compliant with this interface.
// You can lookup for testLogger in config_test.go to see an example of wrapping.
type Logger interface {
	Tracef(string, ...interface{})
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
}
