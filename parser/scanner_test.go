package parser

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/apparentlymart/go-rst"
)

const testScannerFilename = "test.rst"

func TestScanner(t *testing.T) {
	tests := []struct {
		Input string
		Want  []*Token
	}{
		{
			"",
			[]*Token{
				{
					Type:     EOF,
					Position: rst.Position{Line: 1, Column: 1},
				},
			},
		},
		{
			"\n",
			[]*Token{
				{
					Type:     BLANK,
					Data:     "",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 2, Column: 1},
				},
			},
		},
		{
			"hello",
			[]*Token{
				{
					Type:     LINE,
					Data:     "hello",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 2, Column: 1},
				},
			},
		},
		{
			"hello\nworld",
			[]*Token{
				{
					Type:     LINE,
					Data:     "hello",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "world",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			"hello\n    world",
			[]*Token{
				{
					Type:     LINE,
					Data:     "hello",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     INDENT,
					Data:     "    ",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "world",
					Position: rst.Position{Line: 2, Column: 5},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 3, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			"hello\n    world\n    foo\nbaz",
			[]*Token{
				{
					Type:     LINE,
					Data:     "hello",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     INDENT,
					Data:     "    ",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "world",
					Position: rst.Position{Line: 2, Column: 5},
				},
				{
					Type:     LINE,
					Data:     "foo",
					Position: rst.Position{Line: 3, Column: 5},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "baz",
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 5, Column: 1},
				},
			},
		},
		{
			// late indent
			"toplevel\n    nested-quote\n  quote",
			[]*Token{
				{
					Type:     LINE,
					Data:     "toplevel",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     INDENT,
					Data:     "    ",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "nested-quote",
					Position: rst.Position{Line: 2, Column: 5},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 3, Column: 3},
				},
				{
					Type:     LATE_INDENT,
					Data:     "  ",
					Position: rst.Position{Line: 3, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "quote",
					Position: rst.Position{Line: 3, Column: 3},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 4, Column: 1},
				},
			},
		},
		{
			"    world",
			[]*Token{
				{
					Type:     INDENT,
					Data:     "    ",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "world",
					Position: rst.Position{Line: 1, Column: 5},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 2, Column: 1},
				},
			},
		},
		{
			"    hello\n    world",
			[]*Token{
				{
					Type:     INDENT,
					Data:     "    ",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "hello",
					Position: rst.Position{Line: 1, Column: 5},
				},
				{
					Type:     LINE,
					Data:     "world",
					Position: rst.Position{Line: 2, Column: 5},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 3, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			"- push-indent\n  foo",
			[]*Token{
				{
					Type:     LINE,
					Data:     "- push-indent",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "foo",
					Position: rst.Position{Line: 2, Column: 3},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 3, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			"hello\n- push-indent\n  foo\nworld",
			[]*Token{
				{
					Type:     LINE,
					Data:     "hello",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "- push-indent",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "foo",
					Position: rst.Position{Line: 3, Column: 3},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "world",
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 5, Column: 1},
				},
			},
		},
		{
			":lazy-indent:\n    foo",
			[]*Token{
				{
					Type:     LINE,
					Data:     ":lazy-indent:",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "foo",
					Position: rst.Position{Line: 2, Column: 5},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 3, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			":lazy-indent:\n    foo\n    bar",
			[]*Token{
				{
					Type:     LINE,
					Data:     ":lazy-indent:",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "foo",
					Position: rst.Position{Line: 2, Column: 5},
				},
				{
					Type:     LINE,
					Data:     "bar",
					Position: rst.Position{Line: 3, Column: 5},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 4, Column: 1},
				},
			},
		},
		{
			"foo\n:lazy-indent:\n    foo\nbaz",
			[]*Token{
				{
					Type:     LINE,
					Data:     "foo",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LINE,
					Data:     ":lazy-indent:",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "foo",
					Position: rst.Position{Line: 3, Column: 5},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "baz",
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 5, Column: 1},
				},
			},
		},
		{
			":lazy-indent:\nfoo",
			[]*Token{
				{
					Type:     LINE,
					Data:     ":lazy-indent:",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "foo",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			":lazy-indent:",
			[]*Token{
				{
					Type:     LINE,
					Data:     ":lazy-indent:",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 2, Column: 1},
				},
			},
		},
		{
			":lazy-indent:\n\n",
			[]*Token{
				{
					Type:     LINE,
					Data:     ":lazy-indent:",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     DEDENT,
					Data:     "",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     BLANK,
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			"::\n    hello\n  world",
			[]*Token{
				{
					Type:     BLANK,
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LITERAL,
					Data:     "    hello",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     LITERAL,
					Data:     "  world",
					Position: rst.Position{Line: 3, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 4, Column: 1},
				},
			},
		},
		{
			"::\n    hello\n  world\nbaz",
			[]*Token{
				{
					Type:     BLANK,
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LITERAL,
					Data:     "    hello",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     LITERAL,
					Data:     "  world",
					Position: rst.Position{Line: 3, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "baz",
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 5, Column: 1},
				},
			},
		},
		{
			"  ::\n    hello\n  world",
			[]*Token{
				{
					Type:     INDENT,
					Data:     "  ",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     BLANK,
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LITERAL,
					Data:     "    hello",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     LINE,
					Data:     "world",
					Position: rst.Position{Line: 3, Column: 3},
				},
				{
					Type:     DEDENT,
					Position: rst.Position{Line: 4, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 4, Column: 1},
				},
			},
		},
		{
			"literal::\n    hello",
			[]*Token{
				{
					Type:     LINE,
					Data:     "literal:",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LITERAL,
					Data:     "    hello",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			"literal ::\n    hello",
			[]*Token{
				{
					Type:     LINE,
					Data:     "literal",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LITERAL,
					Data:     "    hello",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			"literal :: \n    hello",
			[]*Token{
				{
					Type:     LINE,
					Data:     "literal",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     LITERAL,
					Data:     "    hello",
					Position: rst.Position{Line: 2, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 3, Column: 1},
				},
			},
		},
		{
			"literal::",
			[]*Token{
				{
					Type:     LINE,
					Data:     "literal:",
					Position: rst.Position{Line: 1, Column: 1},
				},
				{
					Type:     EOF,
					Position: rst.Position{Line: 2, Column: 1},
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
		for _, wantToken := range test.Want {
			wantToken.Position.Filename = testScannerFilename
		}

		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			r := strings.NewReader(test.Input)
			scanner := NewScanner(r, testScannerFilename)
			got := make([]*Token, 0, 10)
			for {
				token := scanner.Read()
				got = append(got, token)
				if token.Type == EOF || token.Type == ERROR {
					break
				}

				// Special lines trigger the PushIndent and LazyIndent
				// feedback mechanisms for testing. In the real parser
				// the logic for detecting these is, of course, more complex.
				if token.Type == LINE {
					switch token.Data {
					case "- push-indent":
						scanner.PushIndent(2)
					case ":lazy-indent:":
						scanner.LazyIndent()
					}
				}
			}

			if !reflect.DeepEqual(got, test.Want) {
				t.Errorf(
					"\nincorrect tokens for %q\ngot:  %s\nwant: %s",
					test.Input,
					spewConfig.Sdump(got), spewConfig.Sdump(test.Want),
				)
			}
		})
	}
}
