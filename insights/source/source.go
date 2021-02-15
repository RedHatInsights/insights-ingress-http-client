package source

import "io"

// Source An object for storing data of multiple types
type Source struct {
	ID       string
	Type     string
	Filename string
	Contents io.Reader
}
