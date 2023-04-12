package main

func Must[R any](r R, e error) R {
	if e != nil {
		panic(e)
	}
	return r
}
