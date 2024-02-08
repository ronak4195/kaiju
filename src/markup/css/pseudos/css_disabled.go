package pseudos

import (
	"errors"
	"kaiju/markup/css/rules"
	"kaiju/markup/document"
)

func (p Disabled) Process(elm document.DocElement, value rules.SelectorPart) ([]document.DocElement, error) {
	return []document.DocElement{elm}, errors.New("not implemented")
}
