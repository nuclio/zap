/*
Copyright 2017 The Nuclio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nucliozap

import (
	"bytes"
	"testing"
	"time"

	"github.com/nuclio/logger"
	"github.com/stretchr/testify/suite"
)

type BufferLoggerTestSuite struct {
	suite.Suite
}

func (suite *BufferLoggerTestSuite) TestRedactor() {
	redactor := NewRedactor(nil)
	redactor.AddValueRedactions([]string{"password"})
	redactor.AddRedactions([]string{"replaceme"})
	bufferLogger, err := NewBufferLogger("test", "json", InfoLevel, redactor)
	suite.Require().NoError(err, "Failed creating buffer logger")

	bufferLogger.Logger.customEncoderConfig = NewEncoderConfig()
	bufferLogger.Logger.customEncoderConfig.JSON.VarGroupName = "testVars"
	bufferLogger.Logger.customEncoderConfig.JSON.VarGroupMode = VarGroupModeStructured
	bufferLogger.Logger.prepareVarsCallback = bufferLogger.Logger.prepareVarsStructured

	// log
	bufferLogger.Logger.InfoWith("Check", "password", "123456", "replaceme", "55")

	// get log entries
	logEntries, err := bufferLogger.GetLogEntries()
	suite.Require().NoError(err, "Failed to get log entries")

	// verify (debug should be filtered)
	suite.Require().Equal("Check", logEntries[0]["message"])
	suite.Require().Equal("info", logEntries[0]["level"])
	suite.Require().Equal(map[string]interface{}{
		"*****":    "55",
		"password": "[redacted]",
	}, logEntries[0][bufferLogger.Logger.customEncoderConfig.JSON.VarGroupName])
}

func (suite *BufferLoggerTestSuite) TestJSONEncoding() {
	bufferLogger, err := NewBufferLogger("test", "json", InfoLevel, nil)
	suite.Require().NoError(err, "Failed creating buffer logger")

	suite.verifyLoggedJSONEntries(bufferLogger)
}

func (suite *BufferLoggerTestSuite) TestJSONEncodingAndStructuredVars() {
	bufferLogger, err := NewBufferLogger("test", "json", InfoLevel, nil)
	suite.Require().NoError(err, "Failed creating buffer logger")

	bufferLogger.Logger.customEncoderConfig = NewEncoderConfig()
	bufferLogger.Logger.customEncoderConfig.JSON.VarGroupName = "testVars"
	bufferLogger.Logger.customEncoderConfig.JSON.VarGroupMode = VarGroupModeStructured
	bufferLogger.Logger.prepareVarsCallback = bufferLogger.Logger.prepareVarsStructured
	suite.verifyLoggedJSONEntries(bufferLogger)
}

func (suite *BufferLoggerTestSuite) TestEmptyJSONEncoding() {
	bufferLogger, err := NewBufferLogger("test", "json", InfoLevel, nil)
	suite.Require().NoError(err, "Failed creating buffer logger")

	// get log entries
	logEntries, err := bufferLogger.GetLogEntries()
	suite.Require().NoError(err, "Failed to get log entries")

	// verify there's nothing there
	suite.Require().Len(logEntries, 0)
}

func (suite *BufferLoggerTestSuite) TestGetJSONWithNonJSONEncoding() {
	bufferLogger, err := NewBufferLogger("test", "console", InfoLevel, nil)
	suite.Require().NoError(err, "Failed creating buffer logger")

	// get log entries
	logEntries, err := bufferLogger.GetLogEntries()
	suite.Require().Error(err)
	suite.Require().Nil(logEntries)
}

func (suite *BufferLoggerTestSuite) verifyLoggedJSONEntries(bufferLogger *BufferLogger) {

	varsStructured := false
	varsGroupName := ""
	if bufferLogger.Logger.customEncoderConfig != nil &&
		bufferLogger.Logger.customEncoderConfig.JSON.VarGroupMode == VarGroupModeStructured {
		varsStructured = true
		varsGroupName = bufferLogger.Logger.customEncoderConfig.JSON.VarGroupName
	}

	// write a few entries
	bufferLogger.Logger.Debug("Unstructured %s", "debug")
	bufferLogger.Logger.DebugWith("Structured debug", "mode", "debug")
	bufferLogger.Logger.Info("Unstructured %s", "info")
	bufferLogger.Logger.InfoWith("Structured info", "mode", "info")
	bufferLogger.Logger.Warn("Unstructured %s", "warn")
	bufferLogger.Logger.WarnWith("Structured warn", "mode", "warn")
	bufferLogger.Logger.Error("Unstructured %s", "error")
	bufferLogger.Logger.ErrorWith("Structured error", "mode", "error")

	// get log entries
	logEntries, err := bufferLogger.GetLogEntries()
	suite.Require().NoError(err, "Failed to get log entries")

	// verify (debug should be filtered)
	suite.Require().Equal("Unstructured info", logEntries[0]["message"])
	suite.Require().Equal("info", logEntries[0]["level"])
	suite.Require().Equal("Structured info", logEntries[1]["message"])
	suite.Require().Equal("info", logEntries[1]["level"])

	if varsStructured {
		suite.Require().Equal(map[string]interface{}{"mode": "info"}, logEntries[1][varsGroupName])
	} else {
		suite.Require().Equal("info", logEntries[1]["mode"])

	}

	suite.Require().Equal("Unstructured warn", logEntries[2]["message"])
	suite.Require().Equal("warn", logEntries[2]["level"])
	suite.Require().Equal("Structured warn", logEntries[3]["message"])
	suite.Require().Equal("warn", logEntries[3]["level"])

	if varsStructured {
		suite.Require().Equal(map[string]interface{}{"mode": "warn"}, logEntries[3][varsGroupName])
	} else {
		suite.Require().Equal("warn", logEntries[3]["mode"])
	}

	suite.Require().Equal("Unstructured error", logEntries[4]["message"])
	suite.Require().Equal("error", logEntries[4]["level"])
	suite.Require().Equal("Structured error", logEntries[5]["message"])
	suite.Require().Equal("error", logEntries[5]["level"])
	if varsStructured {
		suite.Require().Equal(map[string]interface{}{"mode": "error"}, logEntries[5][varsGroupName])
	} else {
		suite.Require().Equal("error", logEntries[5]["mode"])
	}
}

type BufferLoggerPoolTestSuite struct {
	suite.Suite
}

func (suite *BufferLoggerPoolTestSuite) TestAllocation() {
	name := "name"
	encoding := "json"
	level := DebugLevel
	timeout := 1 * time.Second

	bufferLoggerPool, err := NewBufferLoggerPool(2, name, encoding, level, nil)
	suite.Require().NoError(err, "Failed creating buffer logger pool")

	// allocate first
	bufferLogger, err := bufferLoggerPool.Allocate(&timeout)
	suite.Require().NoError(err, "Failed allocating buffer logger pool")
	suite.Require().NotNil(bufferLogger)
	suite.Require().Equal(0, bufferLogger.Buffer.Len())
	bufferLogger.Logger.Info("Something")

	// allocate second
	bufferLogger, err = bufferLoggerPool.Allocate(&timeout)
	suite.Require().NoError(err, "Failed allocating buffer logger pool")
	suite.Require().NotNil(bufferLogger)
	suite.Require().Equal(0, bufferLogger.Buffer.Len())
	bufferLogger.Logger.Info("Another")
	suite.Require().NotEqual(0, bufferLogger.Buffer.Len())

	// allocate again - should fail
	nilBufferLogger, err := bufferLoggerPool.Allocate(&timeout)
	suite.Require().Error(err, "Expected to fail allocating")
	suite.Require().Nil(nilBufferLogger)

	// release second
	bufferLoggerPool.Release(bufferLogger)

	// allocate again - should succeed
	bufferLogger, err = bufferLoggerPool.Allocate(&timeout)
	suite.Require().NoError(err, "Failed allocating buffer logger pool")
	suite.Require().NotNil(bufferLogger)

	// allocated logger should be zero'd out
	suite.Require().Equal(0, bufferLogger.Buffer.Len())
}

// ============
// Benchmarking
// ============

func BenchmarkNewBufferLogger(b *testing.B) {
	loggerInstance := createLogger(b, nil)
	executeBenchmark(b, loggerInstance)
}

func BenchmarkNewBufferLoggerWithRedactor(b *testing.B) {
	loggerInstance := createLogger(b, NewRedactor(&bytes.Buffer{}))
	executeBenchmark(b, loggerInstance)
}

func BenchmarkNewBufferLoggerWithDisabledRedactor(b *testing.B) {
	redactor := NewRedactor(&bytes.Buffer{})
	redactor.SetDisabled(true)
	loggerInstance := createLogger(b, redactor)
	executeBenchmark(b, loggerInstance)
}

func BenchmarkNewBufferLoggerWithRedactorWithRedactions(b *testing.B) {
	redactor := NewRedactor(&bytes.Buffer{})
	redactor.AddRedactions([]string{"replaceme"})
	loggerInstance := createLogger(b, redactor)
	executeBenchmark(b, loggerInstance)
}

func BenchmarkNewBufferLoggerWithRedactorWithValueRedactions(b *testing.B) {
	redactor := NewRedactor(&bytes.Buffer{})
	redactor.AddValueRedactions([]string{"replaceme"})
	loggerInstance := createLogger(b, redactor)
	executeBenchmark(b, loggerInstance)
}

func createLogger(b *testing.B, redactor RedactingLogger) logger.Logger {
	bufferLogger, err := NewBufferLogger("test", "console", InfoLevel, redactor)
	if err != nil {
		b.FailNow()
		return nil
	}
	return bufferLogger.Logger
}

func executeBenchmark(b *testing.B, loggerInstance logger.Logger) {
	for i := 0; i < b.N; i++ {
		loggerInstance.InfoWith("Check", "password", "123456", "replaceme", "55")
	}
}

func TestBufferLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(BufferLoggerPoolTestSuite))
	suite.Run(t, new(BufferLoggerTestSuite))
}
