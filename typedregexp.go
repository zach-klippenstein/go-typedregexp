/*
Package typedregexp matches regular expressions into structs.

Regular expressions are specified as a template string (ala text/template), and a struct value
whose fields must all be strings. Each field on the struct must contain a valid regular expression.
The template string then has each reference to each field replaced with a capture group that matches
the corresponding sub-expression in the field.

POSIX regular expressions are not supported. regexp.CompilePOSIX can't handle named capture groups.

The returned TypedRegexp can be used to fill a struct.

	type Values struct {
		Name string
		Age  string
	}

	re, err := Compile("Hi, I'm {{.Name}}. I'm {{.Age}} years old!", Values{
		Name: `\w+`,
		Age:  `\d+`,
	})
	if err != nil {
		log.Fatal(err)
	}

	var values Values
	re.Find("Hi, I'm Sam. I'm 20 years old!", &values)

values is now:
	{Name:Sam Age:20}

See Examples for more features.
*/
package typedregexp

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"text/template"
)

// TypedRegexp is the representation of a compiled regular expression that fills a struct with
// the values of its submatches.
type TypedRegexp struct {
	pattern *regexp.Regexp

	/*
	 An instance of the struct that is passed into Compile whose fields are regexes.
	 Each regex is the value of the field passed into Compile, wrapped with the field's
	 capture group.

	 E.g. For the call
	 Compile("", struct{
	 	Name string
	 	Age  string
	 }{`\w+`, `\d+`}

	 this field would look like:
	 {
	 	Name: `(?P<Name>\w+)`,
	 	Age:  `(?P<Age>\d+)`,
	 }
	*/
	captureGroups interface{}

	// pattern may contain other capture groups/submatches. This field maps each field from captureGroups
	// to all it's submatch indices, as returned by pattern.SubexpNames().
	submatchIndicesByFieldName map[string][]int
}

// MustCompile is the same as Compile, but panics on error.
func MustCompile(pattern string, captureType interface{}) *TypedRegexp {
	re, err := Compile(pattern, captureType)
	if err != nil {
		panic(err)
	}
	return re
}

// Compile creates a TypedRegexp from a regular expression pattern string that uses the text/template
// package to refer to fields in a struct. captureGroups must be a struct with only exported string fields.
// Each field should contain the regex to match into that field.
func Compile(patternTemplate string, captureGroups interface{}) (*TypedRegexp, error) {
	ct := reflect.TypeOf(captureGroups)
	if ct.Kind() != reflect.Struct {
		return nil, fmt.Errorf("captureGroups must be a struct, is a %s", ct)
	}

	captureGroups, fieldNames, err := wrapFieldsInCaptureGroups(captureGroups)
	if err != nil {
		return nil, err
	}

	pattern, err := fillPatternTemplate(patternTemplate, captureGroups)
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("error parsing `%s`: %s", pattern, err)
	}

	indices := make(map[string][]int)
	for i, submatch := range re.SubexpNames() {
		for _, fieldName := range fieldNames {
			if fieldName == submatch {
				indices[fieldName] = append(indices[fieldName], i)
				break
			}
		}
	}

	return &TypedRegexp{
		captureGroups:              captureGroups,
		submatchIndicesByFieldName: indices,
		pattern:                    re,
	}, nil
}

/*
wrapFieldsInCaptureGroups returns a copy of captureGroups with each field
wrapped in a named capture group. The name of each capture group is the name of the field.
Returns an error if any field in captureGroups is not a setable string, or the pattern in a field
is not a valid Regexp.
*/
func wrapFieldsInCaptureGroups(captureGroups interface{}) (interface{}, []string, error) {
	v := reflect.ValueOf(captureGroups)
	t := v.Type()

	// Create a new instance of the struct to hold the wrapped values.
	dest := reflect.New(t).Elem()
	names := make([]string, t.NumField())

	for i := 0; i < v.NumField(); i++ {
		sourceField := v.Field(i)
		destField := dest.Field(i)
		name := t.Field(i).Name
		names[i] = name

		if sourceField.Kind() != reflect.String {
			return nil, nil, fmt.Errorf("fields must be strings, %s is a %s", name, sourceField.Type())
		}
		if !destField.CanSet() {
			return nil, nil, fmt.Errorf("fields must be setable, %s is read-only", name)
		}
		if sourceField.String() == "" {
			return nil, nil, fmt.Errorf("field %s is empty", name)
		}

		// Validate the sub-expression.
		pattern := sourceField.String()
		_, err := regexp.Compile(pattern)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing regex in field %s: %s", name, err)
		}

		pattern = fmt.Sprintf("(?P<%s>%s)", name, pattern)
		destField.SetString(pattern)
	}

	return dest.Interface(), names, nil
}

func fillPatternTemplate(patternTemplate string, groupStruct interface{}) (string, error) {
	temp, err := template.New("").Parse(patternTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := temp.Execute(&buf, groupStruct); err != nil {
		return "", err
	}

	return buf.String(), nil
}

/*
Find sets the fields of values to the matches of the patterns specified for them in Compile.
values must be a pointer to the same type of struct as used in Compile.
Returns true if there is at least 1 field match.

Fields for which no match is found will not be modified, so you can specify default values by
just setting fields on values before passing it to this method.
*/
func (r *TypedRegexp) Find(s string, values interface{}) (found bool) {
	t := reflect.TypeOf(r.captureGroups)
	ptr := reflect.ValueOf(values)

	if ptr.Type() != reflect.PtrTo(t) {
		panic(fmt.Errorf("values must be %s, is %s", reflect.PtrTo(t), ptr.Type()))
	}

	submatches := r.pattern.FindStringSubmatch(s)
	if len(submatches) == 0 {
		return false
	}
	r.assignSubmatchesToStruct(submatches, &ptr)
	return true
}

// FindAll finds the first len(values) matches of the regex in s.
// valuesSlice must be a slice of structs (or pointers to structs) of the same type as passed to Compile.
// Nil elements are skipped.
// Returns the number of matches found. Note that not all valuesSlice[:n] may have been
// set from the regexp, if some of the matches didn't match on any of the fields.
//
// Calling FindAll with a slice of len 1 is basically same as passing a pointer to that struct
// to Find().
func (r *TypedRegexp) FindAll(s string, valuesSlice interface{}) (n int) {
	t := reflect.TypeOf(r.captureGroups)
	slice := reflect.ValueOf(valuesSlice)

	sliceType := slice.Type()
	expectedType := reflect.SliceOf(t)
	expectedPtrType := reflect.SliceOf(reflect.PtrTo(t))
	if sliceType != expectedType && sliceType != expectedPtrType {
		panic(fmt.Errorf("values must be %s or %s, is %s", expectedType, expectedPtrType, sliceType))
	}

	if slice.Len() == 0 {
		return 0
	}

	allSubmatches := r.pattern.FindAllStringSubmatch(s, slice.Len())
	n = len(allSubmatches)
	for submatchIndex, submatches := range allSubmatches {
		v := slice.Index(submatchIndex)

		if v.Kind() == reflect.Ptr && v.IsNil() {
			continue
		}

		r.assignSubmatchesToStruct(submatches, &v)
		//		slice.Index(submatchIndex).Set(v)
	}
	return
}

// assignSubmatchesToStruct uses the submatch mapping from r to set fields in v.
// v must be a struct of the type passed to Compile, or a pointer to such a struct.
// v may be mutated.
func (r *TypedRegexp) assignSubmatchesToStruct(submatches []string, v *reflect.Value) {
	t := reflect.TypeOf(r.captureGroups)

	if v.Kind() == reflect.Ptr {
		*v = v.Elem()
	}

	for i := 0; i < v.NumField(); i++ {
		name := t.Field(i).Name

		// Find the first non-empty submatch.
		fieldValue := r.findFirstNonEmptySubmatchForField(name, submatches)
		v.Field(i).SetString(fieldValue)
	}
}

func (r *TypedRegexp) findFirstNonEmptySubmatchForField(field string, submatches []string) string {
	submatchIndices := r.submatchIndicesByFieldName[field]
	for _, i := range submatchIndices {
		if i >= len(submatches) {
			return ""
		}

		value := submatches[i]
		if value != "" {
			return value
		}
	}
	return ""
}
