package sms

type SmsSender interface {
	Send(phone string, code string) error
}
