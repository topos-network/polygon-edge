package frost

type Frost struct {
}

func (fm *Frost) Initialize() error {
	return nil

}

func (fm *Frost) GenerateKeys() error {
	return nil
}

// NewForkManager is a constructor of ForkManager
func NewFrost() (*Frost, error) {

	fm := &Frost{}

	return fm, nil
}
