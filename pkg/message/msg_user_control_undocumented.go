package message

import "github.com/bluenviron/gortmplib/pkg/rawmessage"

// UserControlUndocumented is an undocumented user control message.
type UserControlUndocumented struct{}

func (m *UserControlUndocumented) unmarshal(_ *rawmessage.Message) error {
	return nil
}

func (m UserControlUndocumented) marshal() (*rawmessage.Message, error) {
	panic("unimplemented")
}
