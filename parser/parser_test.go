package parser

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/apparentlymart/go-rst"
)

const testParserFilename = "test.rst"

func TestParseFragment(t *testing.T) {
	tests := []struct {
		Input string
		Want  *rst.Fragment
	}{
		{
			"",
			&rst.Fragment{},
		},
		{
			"*",
			&rst.Fragment{
				Body: rst.Body{
					&rst.BulletList{
						Items: []*rst.ListItem{
							{
								Body: nil,
							},
						},
					},
				},
			},
		},
		{
			"*\n*",
			&rst.Fragment{
				Body: rst.Body{
					&rst.BulletList{
						Items: []*rst.ListItem{
							{
								Body: nil,
							},
							{
								Body: nil,
							},
						},
					},
				},
			},
		},
	}

	spewConfig := &spew.ConfigState{
		Indent:                  "    ",
		SortKeys:                true,
		DisablePointerAddresses: true,
		DisableCapacities:       true,
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			r := strings.NewReader(test.Input)
			got := ParseFragment(r, testParserFilename)

			if !reflect.DeepEqual(got, test.Want) {
				t.Errorf(
					"\nincorrect result for %q\ngot:  %s\nwant: %s",
					test.Input,
					spewConfig.Sdump(got), spewConfig.Sdump(test.Want),
				)
			}
		})
	}

}
