package nucliozap

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type LoggerTestSuite struct {
	suite.Suite
}

func (suite *LoggerTestSuite) TestRedactor() {
	output := &bytes.Buffer{}
	redactor := NewRedactor(output)
	redactor.AddValueRedactions([]string{"password"})
	redactor.AddRedactions([]string{"replaceme"})
	loggerInstance, err := NewNuclioZapCmd("redacted-test",
		InfoLevel,
		redactor)
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

func TestLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(LoggerTestSuite))
}
