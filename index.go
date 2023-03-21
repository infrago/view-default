package view_default

import (
	"github.com/infrago/infra"
	"github.com/infrago/view"
)

func Driver() view.Driver {
	return &defaultDriver{}
}

func init() {
	infra.Register("default", Driver())
}
