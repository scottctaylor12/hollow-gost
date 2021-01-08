package donut

import (
	"bytes"
	"syscall"
	"time"
	"unsafe"
)

func WaitReadBytes(handle syscall.Handle, tempBytes *[]byte, done *uint32) (err error) {

	var overlapped syscall.Overlapped

	var counter int
	var finished bool

	// Start reading from the pipe in another thread. That thread will block.
	go syncReadFile(handle, (uintptr)(unsafe.Pointer(&(*tempBytes)[0])), uintptr(len(*tempBytes)), done, &overlapped, &finished, &err)

	// Wait until ReadFile has stopped blocking or until timeout
	for finished == false && counter < 5000 {

		time.Sleep(50 * time.Millisecond)

		counter += 50
	}
	return err

}

func ReadFromPipes(stdout syscall.Handle, stdoutBytes *[]byte, stderr syscall.Handle, stderrBytes *[]byte) (err error) {

	tempBytes := make([]byte, 8192)

	// Read STDOUT
	if stdout != 0 {

		for {
			var stdOutDone uint32

			err = WaitReadBytes(stdout, &tempBytes, &stdOutDone)

			if int(stdOutDone) == 0 {
				break
			}

			tempBytes = bytes.Trim(tempBytes, "\x00")
			for _, b := range tempBytes {
				*stdoutBytes = append(*stdoutBytes, b)
			}
			tempBytes = make([]byte, 8192)

			if err != nil {

				if err.Error() != "The pipe has been ended." {
					break
				}

			}

		}
	}

	// Read STDERR
	if stderr != 0 {

		for {
			var stdErrDone uint32

			err = WaitReadBytes(stderr, &tempBytes, &stdErrDone)

			if int(stdErrDone) == 0 {
				break
			}

			tempBytes = bytes.Trim(tempBytes, "\x00")
			for _, b := range tempBytes {
				*stderrBytes = append(*stderrBytes, b)
			}
			tempBytes = make([]byte, 8192)

			if err != nil {

				if err.Error() != "The pipe has been ended." {
					break
				}

			}

		}
	}

	return err
}
