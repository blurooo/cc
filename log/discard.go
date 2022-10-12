package log

var Discard Logger = &discard{}

type discard struct{}

func (d discard) Info(v ...interface{}) {}

func (d discard) Infof(format string, v ...interface{}) {}

func (d discard) Error(v ...interface{}) {}

func (d discard) Errorf(format string, v ...interface{}) {}

func (d discard) Debug(v ...interface{}) {}

func (d discard) Debugf(format string, v ...interface{}) {}

func (d discard) Warn(v ...interface{}) {}

func (d discard) Warnf(format string, v ...interface{}) {}
