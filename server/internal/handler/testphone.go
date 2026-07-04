package handler

import "regexp"

// testPhoneRe matches test accounts 900006000–900006299.
var testPhoneRe = regexp.MustCompile(`^900006[0-2]\d{2}$`)

func isTestPhone(phone string) bool {
	return testPhoneRe.MatchString(phone)
}
