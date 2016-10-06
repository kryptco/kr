package krdclient

//	os.Args is missing hostname, use ps fallback instead on darwin
func currentCommand() *string {
	return nil
}
