package typedregexp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type CaptureGroups struct {
	Name      string
	S0m3th1ng string
}

type InvalidFieldType struct {
	Age int
}

type UnsetableField struct {
	name string
}

func TestWrapFieldsInCaptureGroups(t *testing.T) {
	var (
		groupStruct interface{}
		names       []string
		err         error
	)

	// Empty fields
	groupStruct, names, err = wrapFieldsInCaptureGroups(CaptureGroups{})
	assert.EqualError(t, err, "field Name is empty")

	// Valid patterns
	groupStruct, names, err = wrapFieldsInCaptureGroups(CaptureGroups{
		Name:      `\w+`,
		S0m3th1ng: `\d+`,
	})
	require.NoError(t, err)
	assert.Equal(t, CaptureGroups{
		Name:      `(?P<Name>\w+)`,
		S0m3th1ng: `(?P<S0m3th1ng>\d+)`,
	}, groupStruct)
	assert.Equal(t, []string{"Name", "S0m3th1ng"}, names)

	// Invalid pattern
	groupStruct, names, err = wrapFieldsInCaptureGroups(CaptureGroups{
		Name:      `(`,
		S0m3th1ng: `)`,
	})
	assert.EqualError(t, err, "error parsing regex in field Name: error parsing regexp: missing closing ): `(`")

	// Invalid structs
	groupStruct, names, err = wrapFieldsInCaptureGroups(InvalidFieldType{})
	assert.EqualError(t, err, "fields must be strings, Age is a int")
	groupStruct, names, err = wrapFieldsInCaptureGroups(UnsetableField{})
	assert.EqualError(t, err, "fields must be setable, name is read-only")
}

func TestFillPatternTemplate(t *testing.T) {
	var (
		pattern     string
		err         error
		groupStruct = CaptureGroups{
			Name:      `(?P<Name>\w+)`,
			S0m3th1ng: `(?P<S0m3th1ng>\d+)`,
		}
	)

	// Valid template.
	pattern, err = fillPatternTemplate("{{.Name}}|{{.S0m3th1ng}}", groupStruct)
	require.NoError(t, err)
	assert.Equal(t, `(?P<Name>\w+)|(?P<S0m3th1ng>\d+)`, pattern)

	// Alternatives.
	pattern, err = fillPatternTemplate("a {{.Name}}|b {{.Name}}", groupStruct)
	require.NoError(t, err)
	assert.Equal(t, `a (?P<Name>\w+)|b (?P<Name>\w+)`, pattern)

	// Invalid template.
	pattern, err = fillPatternTemplate("{{.Name}|{{.S0m3th1ng}}", groupStruct)
	assert.EqualError(t, err, "template: :1: unexpected \"}\" in operand")
}

func TestFind_FirstAlternative(t *testing.T) {
	re, err := Compile("a {{.Name}}|b {{.Name}}", CaptureGroups{
		Name:      `\w+`,
		S0m3th1ng: `\d+`,
	})
	require.NoError(t, err)

	var values CaptureGroups

	found := re.Find("a foo", &values)
	assert.True(t, found)
	assert.Equal(t, "foo", values.Name)
}

func TestFind_SecondAlternative(t *testing.T) {
	re, err := Compile("a {{.Name}}|b {{.Name}}", CaptureGroups{
		Name:      `\w+`,
		S0m3th1ng: `\d+`,
	})
	require.NoError(t, err)

	var values CaptureGroups

	found := re.Find("b foo", &values)
	assert.True(t, found)
	assert.Equal(t, "foo", values.Name)
}

func TestFind_NoMatch(t *testing.T) {
	re, err := Compile("a {{.Name}}|b {{.Name}}", CaptureGroups{
		Name:      `\w+`,
		S0m3th1ng: `\d+`,
	})
	require.NoError(t, err)

	var values CaptureGroups

	found := re.Find("foo", &values)
	assert.False(t, found)
	assert.Equal(t, "", values.Name)
}

func TestFind_PartialFields(t *testing.T) {
	re, err := Compile("{{.Name}}( {{.S0m3th1ng}})?", CaptureGroups{
		Name:      `\w+`,
		S0m3th1ng: `\w+`,
	})
	require.NoError(t, err)

	var values CaptureGroups

	found := re.Find("foo", &values)
	assert.True(t, found)
	assert.Equal(t, "foo", values.Name)
	assert.Equal(t, "", values.S0m3th1ng)
}

func TestFind_WrongType(t *testing.T) {
	re, err := Compile("{{.Name}}", CaptureGroups{
		Name:      `\w+`,
		S0m3th1ng: `\d+`,
	})
	require.NoError(t, err)

	// Pointer to wrong type
	func() {
		defer func() {
			err := recover().(error)
			assert.EqualError(t, err, "values must be *typedregexp.CaptureGroups, is *typedregexp.InvalidFieldType")
		}()

		re.Find("foo", &InvalidFieldType{})
	}()

	// Non-pointer
	func() {
		defer func() {
			err := recover().(error)
			assert.EqualError(t, err, "values must be *typedregexp.CaptureGroups, is typedregexp.CaptureGroups")
		}()

		re.Find("foo", CaptureGroups{})
	}()
}
