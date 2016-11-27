package parser

import (
	"bufio"
)

// splitRSTLines is a SplitFunc for bufio.Scanner that frames "lines" from
// an RST document. This is similar to the built-in ScanLines implementation,
// but it additionally trims off trailing whitespace from lines.
func splitRSTLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	advance, token, err = bufio.ScanLines(data, atEOF)

	if token != nil {
	Loop:
		for {
			if len(token) == 0 {
				break
			}

			switch token[len(token)-1] {
			case 8, 9, 32, 12, 11:
				token = token[:len(token)-1]
			default:
				break Loop
			}
		}
	}

	return advance, token, err
}
