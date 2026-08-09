package main

import (
	"reflect"
	"strings"
	"testing"
	"unicode"

	ctxapp "github.com/intivai/backend/internal/context/application"
	cvapp "github.com/intivai/backend/internal/cv/application"
	iamapp "github.com/intivai/backend/internal/iam/application"
	jobapp "github.com/intivai/backend/internal/job/application"
	scrapp "github.com/intivai/backend/internal/screening/application"
)

// DTO contract: every exported field on API result types must have a
// snake_case json tag. Prevents the capitalized-key leak (M1 bug).
func TestDTOJsonTags(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(iamapp.AuthContext{}),
		reflect.TypeOf(iamapp.RegisterOrgResult{}),
		reflect.TypeOf(iamapp.AuthenticateResult{}),
		reflect.TypeOf(iamapp.CreateUserResult{}),
		reflect.TypeOf(jobapp.JobResult{}),
		reflect.TypeOf(cvapp.CVResult{}),
		reflect.TypeOf(cvapp.CVListItem{}),
		reflect.TypeOf(cvapp.CVDetail{}),
		reflect.TypeOf(scrapp.ApplicationResult{}),
		reflect.TypeOf(ctxapp.ContextResult{}),
		reflect.TypeOf(ctxapp.PromptResult{}),
	}
	for _, typ := range types {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			tag := field.Tag.Get("json")
			if tag == "" {
				t.Errorf("%s.%s: missing json tag", typ.Name(), field.Name)
				continue
			}
			name := strings.Split(tag, ",")[0]
			if name == "" || name == "-" {
				t.Errorf("%s.%s: empty json name", typ.Name(), field.Name)
				continue
			}
			if strings.ToLower(name) != name {
				t.Errorf("%s.%s: json tag %q not snake_case", typ.Name(), field.Name, name)
			}
			for _, r := range name {
				if unicode.IsUpper(r) {
					t.Errorf("%s.%s: json tag %q contains uppercase", typ.Name(), field.Name, name)
					break
				}
			}
		}
	}
}
