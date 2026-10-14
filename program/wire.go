package program

// Wire transfers the current source value through the baseline string contract.
// It returns the destination's assignment error. The caller decides when the
// source is ready and owns any resources used by either handle.
func Wire(destination Setter, source Getter) error {
	return destination.Set(source.Get())
}
