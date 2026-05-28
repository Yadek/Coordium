//go:build windows

package main

import "syscall"

// без этого cmd.exe выводит русские буквы кракозябрами
func setupConsole() {
	const cpUTF8 = 65001
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	kernel32.NewProc("SetConsoleOutputCP").Call(uintptr(cpUTF8))
	kernel32.NewProc("SetConsoleCP").Call(uintptr(cpUTF8))
}
