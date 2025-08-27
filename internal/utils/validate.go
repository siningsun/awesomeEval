package utils

import (
	"github.com/dlclark/regexp2"
)

// IsValidEmail checks if the email address is in a valid format:
func IsValidEmail(email string) bool {
	var emailRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	re := regexp2.MustCompile(emailRegex, regexp2.None)
	isValid, err := re.MatchString(email)
	if err != nil {
		return false
	}
	return isValid
}

// IsValidPassword checks if the password meets the required criteria:
func IsValidPassword(password string) bool {
	var passwordRegex = `^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[A-Za-z\d@$!%*?&]{8,}$`
	re := regexp2.MustCompile(passwordRegex, regexp2.None)
	isValid, err := re.MatchString(password)
	if err != nil {
		return false
	}
	return isValid
}

func IsValidMobile(mobile string) bool {
	var mobileRegex = `^1[3-9]\d{9}$`
	re := regexp2.MustCompile(mobileRegex, regexp2.None)
	isValid, err := re.MatchString(mobile)
	if err != nil {
		return false
	}
	return isValid
}

func IsValidVerificationCode(code string) bool {
	var codeRegex = `^\d{6}$`
	re := regexp2.MustCompile(codeRegex, regexp2.None)
	isValid, err := re.MatchString(code)
	if err != nil {
		return false
	}
	return isValid
}
