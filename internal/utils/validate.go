package utils

import "regexp"

// IsValidEmail checks if the email address is in a valid format:
func IsValidEmail(email string) bool {
	var emailRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}

// IsValidPassword checks if the password meets the required criteria:
func IsValidPassword(password string) bool {
	var passwordRegex = `^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[A-Za-z\d@$!%*?&]{8,}$`
	re := regexp.MustCompile(passwordRegex)
	return re.MatchString(password)
}

func IsValidMobile(mobile string) bool {
	var mobileRegex = `^1[3-9]\d{9}$`
	re := regexp.MustCompile(mobileRegex)
	return re.MatchString(mobile)
}

func IsValidVerificationCode(code string) bool {
	var codeRegex = `^\d{6}$`
	re := regexp.MustCompile(codeRegex)
	return re.MatchString(code)
}
