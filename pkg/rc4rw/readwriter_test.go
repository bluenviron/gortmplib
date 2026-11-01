package rc4rw_test

import (
	"io"

	"github.com/bluenviron/gortmplib/pkg/rc4rw"
)

var _ io.ReadWriter = (*rc4rw.ReadWriter)(nil)
