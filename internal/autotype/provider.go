package autotype

var (
	globalAutotype         Autotype
	defaultAutotypeFactory func() Autotype
)

func DefaultAutotype() Autotype {
	if globalAutotype != nil {
		return globalAutotype
	}
	if defaultAutotypeFactory != nil {
		return defaultAutotypeFactory()
	}
	return nil
}

func SetAutotype(a Autotype) {
	globalAutotype = a
}
