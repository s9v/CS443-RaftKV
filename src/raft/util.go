package raft

import "log"

// Debugging
const Debug = 0

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 1 {
                log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
		log.Printf(format, a...)
	}
	return
}

func CPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
                log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
		log.Printf(format, a...)
	}
	return
}
