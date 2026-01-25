package common

type FakeLogger struct{}

func (fl FakeLogger) Debug(fmtStr string, vals ...any)   {}
func (fl FakeLogger) Info(fmtStr string, vals ...any)    {}
func (fl FakeLogger) Warning(fmtStr string, vals ...any) {}
func (fl FakeLogger) Error(fmtStr string, vals ...any)   {}
