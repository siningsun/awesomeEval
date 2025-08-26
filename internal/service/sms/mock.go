package sms

import "fmt"

type MockSmsService struct{}

func (s *MockSmsService) Send(phone string, code string) error {
	fmt.Printf("send %s to phone %s\n", code, phone)
	return nil
}
