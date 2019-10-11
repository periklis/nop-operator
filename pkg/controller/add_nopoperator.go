package controller

import (
	"github.com/periklis/nop-operator/pkg/controller/nopoperator"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, nopoperator.Add)
}
