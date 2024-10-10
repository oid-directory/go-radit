package common

import (
	"bufio"
	"errors"
	//"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const (
	randChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randIDSize = 8
)

const (                                                                 
        // RFCURLPrefix contains the URI prefix for use with RFCs and   
        // Internet-Drafts referenced by various SMI registrations.     
        RFCURIPrefix = `https://datatracker.ietf.org/doc/html/`         
        RFCErrataPrefix = `https://www.rfc-editor.org/errata_search.php?eid=`
	IANAAssignmentsPrefix = `https://iana.org/assignments/`
)

var (
	lc        func(string) string                 = strings.ToLower
	uc        func(string) string                 = strings.ToUpper
	eq        func(string, string) bool           = strings.EqualFold
	split     func(string, string) []string       = strings.Split
	join      func([]string, string) string       = strings.Join
	sidx      func(string, string) int            = strings.Index
	idxr      func(string, rune) int              = strings.IndexRune
	trimS     func(string) string                 = strings.TrimSpace
	trimL     func(string, string) string         = strings.TrimLeft
	trimR     func(string, string) string         = strings.TrimRight
	cutPfx    func(string, string) (string, bool) = strings.CutPrefix
	hasPfx    func(string, string) bool           = strings.HasPrefix
	hasSfx    func(string, string) bool           = strings.HasSuffix
	repeat    func(string, int) string            = strings.Repeat
	atoi      func(string) (int, error)           = strconv.Atoi
	rplc      func(string, string, string) string = strings.ReplaceAll
	open      func(string) (*os.File, error)      = os.Open
	ctns      func(string, string) bool           = strings.Contains
	mkerr     func(string) error                  = errors.New
	isDigit   func(rune) bool                     = unicode.IsDigit
	isLower   func(rune) bool                     = unicode.IsLower
	isUpper   func(rune) bool                     = unicode.IsUpper
	newScan   func(io.Reader) *bufio.Scanner      = bufio.NewScanner
	newReader func(string) *strings.Reader        = strings.NewReader
)

var eof error = io.EOF

func trimNL(in string) string {
	nl := string(rune(10))
	return trimR(trimL(in, nl), nl)
}

func RemoveNL(in string) string {
	nl := string(rune(10))
	return rplc(in, nl, `\n`)
}

func CondenseWHSP(b string) string {
	b = trimS(b)

	var last bool // previous char was WHSP or HTAB.
	var bld strings.Builder

	for i := 0; i < len(b); i++ {
		c := rune(b[i])
		switch c {
		case rune(9), rune(32): // match either WHSP or horizontal tab
			if !last {
				last = true
				bld.WriteRune(rune(32)) // Add WHSP
			}
		default: // match all other characters
			if last {
				last = false
			}
			bld.WriteRune(c)
		}
	}

	return bld.String()
}

func ReadBytes(file string) ([]byte, error) {
	b, err := open(file)
	if err != nil {
		return nil, err
	}
	defer b.Close()

	byteV, _ := ioutil.ReadAll(b)
	return byteV, nil
}

func RandomID() string {
	id := make([]byte, randIDSize)
	for i := range id {
		id[i] = randChars[rand.Int63()%int64(len(randChars))]
	}
	return string(id)
}

func isAlnum(r rune) bool {
	return isLower(r) || isUpper(r) || isDigit(r)
}

func IsNumber(n string) bool {
	if len(n) == 0 {
		return false
	}

	for i := 0; i < len(n); i++ {
		if !isDigit(rune(n[i])) {
			return false
		}
	}
	return true
}

func checkOIDValues(o string, isp, spl []string) (err error) {
	// make sure things add up
	for ix := 0; ix < len(isp); ix++ {
		if len(isp) != len(spl) {
			err = mkerr("Length mismatch")
			break
		}

		if IsNumber(isp[ix]) {
			if isp[ix] != spl[ix] {
				err = mkerr("Arc mismatch")
				break
			}
		} else {
			if idx := idxr(isp[ix], '('); idx != -1 {
				if inner := trimR(isp[ix][idx+1:], `)`); !IsNumber(inner) {
					err = mkerr("ARC NAN: " + isp[ix])
					break
				} else if inner != spl[ix] {
					err = mkerr("NameAndNumberForm mismatch")
					break
				}
			}
		}
	}

	return
}
