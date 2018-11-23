package raft

import "log"

// Debugging
const Debug = 1

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
                log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
		log.Printf(format, a...)
	}
	return
}
