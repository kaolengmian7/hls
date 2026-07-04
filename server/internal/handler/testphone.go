package handler

import "regexp"

// testPhoneRe matches test accounts 900006000–900006299.
var testPhoneRe = regexp.MustCompile(`^900006(00\d|0[1-9]\d|1\d\d|2[0-9]\d)$`)

func isTestPhone(phone string) bool {
	return testPhoneRe.MatchString(phone)
}
