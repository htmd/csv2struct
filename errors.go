package csv2struct

type IncorrectFileErr struct {
	message string
}

func NewIncorrectFileErr(message string) IncorrectFileErr {
	return IncorrectFileErr{
		message: message,
	}
}

func (e IncorrectFileErr) Error() string {
	return e.message
}
