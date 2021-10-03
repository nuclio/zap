/*
Copyright 2021 The Nuclio Authors.
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
	"fmt"
	"io"
	"regexp"
)

type RedactingLogger interface {

	// SetOutput sets redactor output
	SetOutput(io.Writer)

	// GetOutput returns redactor writer
	GetOutput() io.Writer

	// Write writes to output
	Write(p []byte) (n int, err error)

	// AddRedactions redacts simple strings
	AddRedactions(redactions []string)

	// AddValueRedactions redacts key:[value] or key=[value] kind of strings
	AddValueRedactions(valueRedactions []string)

	// SetDisabled turns logger redaction on/off
	SetDisabled(disable bool)
}

type Redactor struct {
	disabled               bool
	output                 io.Writer
	redactions             [][]byte
	valueRedactions        [][]byte
	valueRedactionsRegexps []regexp.Regexp
	replacement            []byte
	valueReplacement       []byte

	// must have same signature as io.Writer Write
	redactFunc func(p []byte) (n int, err error)
}

func NewRedactor(output io.Writer) *Redactor {
	redactor := &Redactor{
		output:                 output,
		redactions:             [][]byte{},
		valueRedactions:        [][]byte{},
		valueRedactionsRegexps: []regexp.Regexp{},
		replacement:            []byte("*****"),
		valueReplacement:       []byte(`$1"[redacted]"`),
		disabled:               false,
	}
	redactor.redactFunc = redactor.redactEnabled
	return redactor
}

func (r *Redactor) SetOutput(output io.Writer) {
	r.output = output
}

func (r *Redactor) GetOutput() io.Writer {
	return r.output
}

func (r *Redactor) GetRedactions() [][]byte {
	return r.redactions
}

func (r *Redactor) SetDisabled(disable bool) {
	r.disabled = disable
	if disable {
		r.redactFunc = r.redactDisabled
	} else {
		r.redactFunc = r.redactEnabled
	}
}

func (r *Redactor) AddValueRedactions(valueRedactions []string) {
	for _, valueRedaction := range valueRedactions {
		r.valueRedactions = append(r.valueRedactions, []byte(valueRedaction))
	}
	r.valueRedactions = r.removeDuplicates(r.valueRedactions)
	r.prepareReplacements()
}

func (r *Redactor) AddRedactions(redactions []string) {
	var nonEmptyRedactions []string

	for _, redaction := range redactions {
		if redaction != "" {
			nonEmptyRedactions = append(nonEmptyRedactions, redaction)
		}
	}

	for _, nonEmptyRedaction := range nonEmptyRedactions {
		r.redactions = append(r.redactions, []byte(nonEmptyRedaction))
	}
	r.redactions = r.removeDuplicates(r.redactions)
}

func (r *Redactor) Write(p []byte) (n int, err error) {
	return r.redactFunc(p)
}

func (r *Redactor) Enable() {
	r.disabled = false
}

func (r *Redactor) Disable() {
	r.disabled = true
}

func (r *Redactor) prepareReplacements() {

	// redact values of either strings of type `valueRedaction=[value]` or `valueRedaction: [value]`
	// w/wo single/double quotes
	// golang regex doesn't support lookarounds, so we will check things manually
	matchKeyWithSeparatorTemplate := `\\*[\'"]?(?i)%s\\*[\'"]?\s*[=:]\s*`
	matchValue := `\'[^\']*?\'|\"[^\"]*\"|\S*`

	for _, redactionField := range r.valueRedactions {
		matchKeyWithSeparator := fmt.Sprintf(matchKeyWithSeparatorTemplate, redactionField)
		r.valueRedactionsRegexps = append(r.valueRedactionsRegexps,
			*regexp.MustCompile(fmt.Sprintf(`(%s)(%s)`, matchKeyWithSeparator, matchValue)),
		)
	}

}

func (r *Redactor) redactDisabled(p []byte) (n int, err error) {
	return r.output.Write(p)
}

func (r *Redactor) redactEnabled(p []byte) (n int, err error) {
	redactedPrint := r.redact(p)
	n, err = r.output.Write(redactedPrint)
	if err != nil {
		return
	}
	if n != len(redactedPrint) {
		err = io.ErrShortWrite
		return
	}

	// HACK: let the caller know we wrote the original length of the text
	// To prevent caller explode while validating the length of the written text (redaction might change the length)
	return len(p), err
}

func (r *Redactor) redact(inputToRedact []byte) []byte {

	// replace key=value or key: value
	for _, valueRedactionsRegexp := range r.valueRedactionsRegexps {
		inputToRedact = valueRedactionsRegexp.ReplaceAll(inputToRedact, r.valueReplacement)
	}

	// replace the simple string redactions
	for _, redactionField := range r.redactions {
		inputToRedact = bytes.ReplaceAll(inputToRedact, redactionField, r.replacement)
	}

	return inputToRedact
}

func (r *Redactor) removeDuplicates(elements [][]byte) [][]byte {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[string(elements[v])] = true
	}

	// Place all keys from the map into a slice.
	var result [][]byte
	for key := range encountered {
		result = append(result, []byte(key))
	}
	return result
}
