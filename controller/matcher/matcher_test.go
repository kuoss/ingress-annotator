package matcher

import (
	"testing"

	"github.com/jmnote/tester"
	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	testCases := []struct {
		pattern    string
		objectName string
		want       bool
	}{
		{
			pattern:    ",,",
			objectName: "",
			want:       false,
		},
		{
			pattern:    "",
			objectName: "",
			want:       true,
		},
		{
			pattern:    "",
			objectName: "dev",
			want:       true,
		},
		{
			pattern:    "*",
			objectName: "",
			want:       true,
		},
		{
			pattern:    "*",
			objectName: "x",
			want:       true,
		},
		{
			pattern:    "*",
			objectName: "dev",
			want:       true,
		},
		{
			pattern:    "prod1",
			objectName: "prod2",
			want:       false,
		},
		{
			pattern:    "*-priv",
			objectName: "my-priv",
			want:       true,
		},
		{
			pattern:    "*-priv",
			objectName: "priv",
			want:       false,
		},
		{
			pattern:    "dev1,dev2",
			objectName: "dev2",
			want:       true,
		},
		{
			pattern:    "dev1,dev2",
			objectName: "dev3",
			want:       false,
		},
		{
			pattern:    "!dev*,prod*",
			objectName: "dev",
			want:       false,
		},
		{
			pattern:    "!dev*,prod*",
			objectName: "dev2",
			want:       false,
		},
		{
			pattern:    "!dev*,prod*",
			objectName: "prod",
			want:       true,
		},
		{
			pattern:    "!dev*,prod*",
			objectName: "prod2",
			want:       true,
		},
	}

	for i, tc := range testCases {
		t.Run(tester.Name(i, tc.pattern, tc.objectName), func(t *testing.T) {
			got := Match(tc.pattern, tc.objectName)
			assert.Equal(t, tc.want, got)
		})
	}
}
