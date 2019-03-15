package exchange

import (
	"github.com/aghape/core"
	"github.com/aghape/core/resource"
)

// Container is an interface, any exporting/importing backends needs to implement this
type Container interface {
	NewReader(*Resource, *core.Context) (Rows, error)
	NewWriter(*Resource, *core.Context) (Writer, error)
}

// Rows is an interface, backends need to implement this in order to read data from it
type Rows interface {
	Header() []string
	ReadRow() (*resource.MetaValues, error)
	Next() bool
	Total() uint
}

// Writer is an interface, backends need to implement this in order to write data
type Writer interface {
	WriteHeader() error
	WriteRow(interface{}) (*resource.MetaValues, error)
	Flush() error
}
