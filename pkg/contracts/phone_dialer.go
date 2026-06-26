package contracts

// PhoneDialer — необов'язкова функція дзвінка через Asterisk.
// Якщо AMI не налаштований — повертається nil, ендпоїнти /dial відповідають 503.
type PhoneDialer interface {
	DialPhone(phone string) (callID string, err error)
	HangupCall(callID string)
	IsDialerConnected() bool
}
