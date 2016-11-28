package rst

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

const testParserFilename = "test.rst"

func TestParseFragment(t *testing.T) {
	tests := []struct {
		Input string
		Want  *Fragment
	}{
		{
			"",
			&Fragment{},
		},
		{
			"* foo",
			&Fragment{
				Body: Body{
					&BulletList{
						Items: []*ListItem{
							{
								Body: Body{
									&Paragraph{
										Text: Text{
											CharData("foo"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			"* foo\n* bar",
			&Fragment{
				Body: Body{
					&BulletList{
						Items: []*ListItem{
							{
								Body: Body{
									&Paragraph{
										Text: Text{
											CharData("foo"),
										},
									},
								},
							},
							{
								Body: Body{
									&Paragraph{
										Text: Text{
											CharData("bar"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			"    blockquote\n    baz",
			&Fragment{
				Body: Body{
					&BlockQuote{
						Quote: Body{
							&Paragraph{
								Text: Text{
									CharData("blockquote"),
									CharData("baz"),
								},
							},
						},
					},
				},
			},
		},
		{
			"    nested-blockquote\n  baz",
			&Fragment{
				Body: Body{
					&BlockQuote{
						Quote: Body{
							&BlockQuote{
								Quote: Body{
									&Paragraph{
										Text: Text{
											CharData("nested-blockquote"),
										},
									},
								},
							},
							&Paragraph{
								Text: Text{
									CharData("baz"),
								},
							},
						},
					},
				},
			},
		},
		{
			"    quote\n\n    -- attribution",
			&Fragment{
				Body: Body{
					&BlockQuote{
						Quote: Body{
							&Paragraph{
								Text: Text{
									CharData("quote"),
								},
							},
						},
						Attribution: Text{
							CharData("attribution"),
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
