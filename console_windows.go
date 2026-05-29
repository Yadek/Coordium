//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// Без этого cmd.exe выводит русские буквы кракозябрами и не понимает ANSI-цвета.
func setupConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")

	const cpUTF8 = 65001
	kernel32.NewProc("SetConsoleOutputCP").Call(uintptr(cpUTF8))
	kernel32.NewProc("SetConsoleCP").Call(uintptr(cpUTF8))

	// включаем обработку ANSI-кодов цвета (Windows 10+)
	const enableVTProcessing = 0x0004
	getStdHandle := kernel32.NewProc("GetStdHandle")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")

	stdout, _, _ := getStdHandle.Call(^uintptr(10)) // STD_OUTPUT_HANDLE = -11
	var mode uint32
	getConsoleMode.Call(stdout, uintptr(unsafe.Pointer(&mode)))
	setConsoleMode.Call(stdout, uintptr(mode|enableVTProcessing))
}
