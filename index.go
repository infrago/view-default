package view_default

import (
	"github.com/infrago/view"
)

func Driver() view.Driver {
	return &defaultDriver{}
}

func init() {
	view.Register("default", Driver())
}
