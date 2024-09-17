package todo_test

import (
	"avshetty1980/microservice-testing/todo"
	"testing"

	"github.com/stretchr/testify/suite"
)

func (s *StackSuite) TestEmpty() {

	stack := todo.NewStack()

	s.True(stack.IsEmpty())
}

type StackSuite struct {
	suite.Suite
}

func TestStackSuite(t *testing.T) {
	suite.Run(t, new(StackSuite))
}
