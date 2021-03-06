package nucliozap

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type LoggerTestSuite struct {
	suite.Suite
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
