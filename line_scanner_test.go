package rst

import (
	"bufio"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestSplitRSTLines(t *testing.T) {
	tests := []struct {
		Input    string
		Expected []string
	}{
		{
			"",
			[]string{},
		},
		{
			"\n",
			[]string{
				"",
			},
		},
		{
			"Hello",
			[]string{
				"Hello",
			},
		},
		{
			"Hello\n",
			[]string{
				"Hello",
			},
		},
		{
			"Hello\nWorld",
			[]string{
				"Hello",
				"World",
			},
		},
		{
			"Hello \nWorld ",
			[]string{
				"Hello",
				"World",
			},
		},
		{
			" Hello   \n World   ",
			[]string{
				" Hello",
				" World",
			},
		},
		{
			"\tHello\t\n\tWorld\t",
			[]string{
				"\tHello",
				"\tWorld",
			},
		},
		{
			"Hello\v\f\nWorld\v\f",
			[]string{
				"Hello",
				"World",
			},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := strings.NewReader(test.Input)
			scanner := bufio.NewScanner(r)
			scanner.Split(splitRSTLines)
			got := make([]string, 0, len(test.Expected))
			for scanner.Scan() {
				got = append(got, scanner.Text())
			}
			if scanner.Err() != nil {
				t.Fatalf("got error: %s", scanner.Err())
			}
			if !reflect.DeepEqual(got, test.Expected) {
				t.Errorf(
					"incorrect output for %q\ngot:  %#v\nwant: %#v",
					test.Input, got, test.Expected,
				)
			}
		})
	}
}
