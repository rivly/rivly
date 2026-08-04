package stack

type Kind int

const (
	KindInvalid Kind = iota
	KindNotFound
	KindConflict
	KindRejected
	KindUnreachable
)

type Error struct {
	Kind    Kind
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func invalid(message string) error {
	return &Error{Kind: KindInvalid, Message: message}
}

func notFound(message string) error {
	return &Error{Kind: KindNotFound, Message: message}
}

func conflict(message string) error {
	return &Error{Kind: KindConflict, Message: message}
}

func rejected(message string) error {
	return &Error{Kind: KindRejected, Message: message}
}

func unreachable(message string) error {
	return &Error{Kind: KindUnreachable, Message: message}
}
