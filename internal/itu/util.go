package itu

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const (
	randChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randIDSize = 8
)

var (
	lc        func(string) string                 = strings.ToLower
	uc        func(string) string                 = strings.ToUpper
	eq        func(string, string) bool           = strings.EqualFold
	split     func(string, string) []string       = strings.Split
	join      func([]string, string) string       = strings.Join
	idxr      func(string, rune) int              = strings.IndexRune
	trimS     func(string) string                 = strings.TrimSpace
	trimL     func(string, string) string         = strings.TrimLeft
	trimR     func(string, string) string         = strings.TrimRight
	cutPfx    func(string, string) (string, bool) = strings.CutPrefix
	hasPfx    func(string, string) bool           = strings.HasPrefix
	hasSfx    func(string, string) bool           = strings.HasSuffix
	repeat    func(string, int) string            = strings.Repeat
	atoi      func(string) (int, error)           = strconv.Atoi
	itoa      func(int) (string)                  = strconv.Itoa
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

var (
	eof error = io.EOF
	nilInstanceErr error = mkerr("Instance or receiver is nil; must initialize")
)

