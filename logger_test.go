/*
Copyright 2018 The Nuclio Authors.
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
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type LoggerTestSuite struct {
	suite.Suite
}

func (suite *LoggerTestSuite) TestRedactor() {
	output := &bytes.Buffer{}
	loggerInstance, err := NewNuclioZapCmd("redacted-test",
		InfoLevel,
		NewRedactor(output))
	loggerInstance.GetRedactor().AddValueRedactions([]string{"password"})
	loggerInstance.GetRedactor().AddRedactions([]string{"replaceme"})
	suite.Require().NoError(err, "Failed creating buffer logger")

	// log
	loggerInstance.InfoWith("Check", "password", "123456", "replaceme", "55")

	// verify (debug should be filtered)
	suite.Require().Contains(output.String(), "Check")
	suite.Require().NotContains(output.String(), "123456")
	suite.Require().NotContains(output.String(), "replaceme")
}

func (suite *LoggerTestSuite) TestPrepareVars() {
	zap := NuclioZap{}
	vars := []interface{}{
		"some", "thing",
		"something", "else",
	}
	encodedVars := zap.prepareVarsFlattened(vars)
	suite.Require().Equal("some=thing || something=else", encodedVars)

	structuredVars := zap.prepareVarsStructured(vars)
	suite.Require().Equal(map[string]interface{}{
		"some":      "thing",
		"something": "else",
	}, structuredVars)
}

func (suite *LoggerTestSuite) TestGetChild() {
	writer := &bytes.Buffer{}
	encoderConfig := NewEncoderConfig()
	encoderConfig.JSON.VarGroupName = "extra"
	encoderConfig.JSON.VarGroupMode = VarGroupModeStructured
	zap, err := NewNuclioZap("test", "json", encoderConfig, writer, writer, DebugLevel)
	suite.Require().NoError(err)
	childLogger := zap.GetChild("some-child")
	childLogger.InfoWith("Test", "some", "thing")
	suite.Require().Contains(writer.String(), `"extra":{"some":"thing"}`)
	suite.Require().Contains(writer.String(), `"name":"test.some-child"`)
}

func (suite *LoggerTestSuite) TestAddContextToVars() {
	zap, err := NewNuclioZap("test", "json", nil, &bytes.Buffer{}, &bytes.Buffer{}, DebugLevel)
	suite.Require().NoError(err)
	ctx := context.Background()

	requestID := "123456"
	contextID := "abcdef"
	ctx = context.WithValue(ctx, RequestIDKey, requestID)
	ctx = context.WithValue(ctx, ContextIDKey, contextID)

	vars := zap.addContextToVars(ctx, []interface{}{"some", "thing"})
	for _, expected := range []string{
		"requestID",
		requestID,
		"ctx",
		contextID,
	} {
		suite.Require().Contains(vars, expected)
	}
	suite.Require().Contains(vars, "some")
	suite.Require().Contains(vars, "thing")

	// validate it skips existing values
	existingRequestID := "987654"
	vars = zap.addContextToVars(ctx, []interface{}{"some", "thing", "requestID", existingRequestID})
	for _, expected := range []string{
		"requestID",
		existingRequestID,
		"ctx",
		contextID,
	} {
		suite.Require().Contains(vars, expected)
	}
}

func TestLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(LoggerTestSuite))
}
