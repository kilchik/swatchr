package main

type insufficientDiskSpaceError struct {
}

func (e insufficientDiskSpaceError) Error() string {
	return "insufficient disk space"
}

type alreadyExistsError struct {
}

func (e alreadyExistsError) Error() string {
	return "movie already added"
}
