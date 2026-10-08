package liberrors

// ErrReaderUnpublished is an error that can be returned by a reader.
type ErrReaderUnpublished struct{}

// Error implements the error interface.
func (e ErrReaderUnpublished) Error() string {
	return "stream unpublished"
}
