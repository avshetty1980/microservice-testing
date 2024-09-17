package todo

type Stack struct {
}

func NewStack() *Stack {
	return &Stack{}
}

func (stack *Stack) IsEmpty() bool {
	return true
}
