package autotype

type Autotype interface {
	Type(text string) error
}
