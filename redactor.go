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
	"fmt"
	"io"
	"regexp"
	"strings"
)

type RedactingLogger interface {
	GetRedactor() *Redactor
	GetOutput() io.Writer
}

type Redactor struct {
	disabled               bool
	output                 io.Writer
	redactions             []string
	valueRedactions        []string
	replacementString      string
	valueReplacementString string
}

func NewRedactor(output io.Writer) *Redactor {
	return &Redactor{
		output:                 output,
		redactions:             []string{},
		valueRedactions:        []string{},
		replacementString:      "*****",
		valueReplacementString: `"[redacted]"`,
		disabled:               false,
	}
}

func (r *Redactor) GetOutput() io.Writer {
	return r.output
}

func (r *Redactor) AddValueRedactions(valueRedactions []string) {
	r.valueRedactions = append(r.valueRedactions, valueRedactions...)
	r.valueRedactions = r.removeDuplicates(r.valueRedactions)
}

func (r *Redactor) GetRedactions() []string {
	return r.redactions
}

func (r *Redactor) AddRedactions(redactions []string) {
	var nonEmptyRedactions []string

	for _, redaction := range redactions {
		if redaction != "" {
			nonEmptyRedactions = append(nonEmptyRedactions, redaction)
		}
	}

	r.redactions = append(r.redactions, nonEmptyRedactions...)
	r.redactions = r.removeDuplicates(r.redactions)
}

func (r *Redactor) Write(p []byte) (n int, err error) {
	redactedPrint := r.redact(string(p))
	n, err = r.output.Write([]byte(redactedPrint))
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

func (r *Redactor) Enable() {
	r.disabled = false
}

func (r *Redactor) Disable() {
	r.disabled = true
}

func (r *Redactor) redact(input string) string {
	if r.disabled {
		return input
	}

	redacted := input

	// golang regex doesn't support lookarounds, so we will check things manually
	matchKeyWithSeparatorTemplate := `\\*[\'"]?(?i)%s\\*[\'"]?\s*[=:]\s*`
	matchValue := `\'[^\']*?\'|\"[^\"]*\"|\S*`

	// redact values of either strings of type `valueRedaction=[value]` or `valueRedaction: [value]`
	// w/wo single/double quotes
	for _, redactionField := range r.valueRedactions {
		matchKeyWithSeparator := fmt.Sprintf(matchKeyWithSeparatorTemplate, redactionField)
		re := regexp.MustCompile(fmt.Sprintf(`(%s)(%s)`, matchKeyWithSeparator, matchValue))
		redacted = re.ReplaceAllString(redacted, fmt.Sprintf(`$1%s`, r.valueReplacementString))
	}

	// replace the simple string redactions
	for _, redactionField := range r.redactions {
		redacted = strings.ReplaceAll(redacted, redactionField, r.replacementString)
	}

	return redacted
}

func (r *Redactor) removeDuplicates(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	var result []string
	for key := range encountered {
		result = append(result, key)
	}
	return result
}
