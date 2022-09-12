package main

import (
	"regexp"
	"strings"
)

//test owner again 

type LookupOption int

const (
	LookupOptionDefault LookupOption = iota
	LookupOptionSubstring
	LookupOptionRegexp
)

func (lo LookupOption) String() string {
	return []string{"Default", "Substring", "Regexp"}[lo]
}

func LookupOptionConvert(input string) (lo LookupOption) {
	switch strings.ToLower(input) {
	case "default":
		lo = LookupOptionDefault
	case "substring":
		lo = LookupOptionSubstring
	case "regexp":
		lo = LookupOptionRegexp
	default:
		lo = LookupOptionDefault
	}
	return lo
}

type LookupRegex struct {
	Key    string
	Value  []string
	Regexp *regexp.Regexp // save the regexp value when LookupOption is Regexp
}

func lookupViaSubstring(input string, keyValueSlice []*LookupRegex) (result []string) {
	inputLowerCase := strings.ToLower(input)

	for _, iter := range keyValueSlice {
		if strings.Index(inputLowerCase, iter.Key) > -1 {
			return iter.Value
		}
	}
	return nil
}

func lookupViaRegexp(input string, regexpSlice []*LookupRegex) (result []string) {
	for _, iter := range regexpSlice {
		res := iter.Regexp.Find([]byte(input))
		if res != nil {
			return iter.Value
		}
	}
	return nil
}
