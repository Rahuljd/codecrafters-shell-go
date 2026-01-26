package main

import (
	"os"
	"strings"
)

// executableCache stores found executables to avoid repeated filesystem calls
var executableCache map[string]bool

// isExecAny checks if a file has execute permissions
func isExecAny(mode os.FileMode) bool {
	return mode.Perm()&0111 != 0
}

// findAllExes searches PATH for all executable files
func findAllExes() {
	executableCache = make(map[string]bool)
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))

	for _, p := range paths {
		entries, err := os.ReadDir(p)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				info, err := entry.Info()
				if err != nil {
					continue
				}

				// On Windows, also check for .exe, .bat, .cmd, .com extensions
				isExecutable := false
				if os.PathSeparator == '\\' {
					name := strings.ToLower(entry.Name())
					if strings.HasSuffix(name, ".exe") ||
						strings.HasSuffix(name, ".bat") ||
						strings.HasSuffix(name, ".cmd") ||
						strings.HasSuffix(name, ".com") {
						isExecutable = true
					}
				} else {
					// Unix: check execute bit
					isExecutable = isExecAny(info.Mode())
				}

				if isExecutable {
					executableCache[entry.Name()] = true
				}
			}
		}
	}
}

// getPathExecutables returns all executables found in PATH directories
func getPathExecutables() map[string]bool {
	if executableCache == nil {
		findAllExes()
	}
	return executableCache
}
