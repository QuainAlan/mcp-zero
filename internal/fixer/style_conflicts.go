package fixer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StyleConflictPair represents a pair of files that conflict due to different naming styles
type StyleConflictPair struct {
	GoZeroStyle string // go_zero style: snake_case (e.g., service_context.go)
	GoZeroFlat  string // gozero style: flat (e.g., servicecontext.go)
}

// knownStyleConflicts lists all known file pairs that can conflict between go_zero and gozero styles
var knownStyleConflicts = []StyleConflictPair{
	{GoZeroStyle: "service_context.go", GoZeroFlat: "servicecontext.go"},
	// Add more known conflicts here if discovered
}

// CleanupStyleConflicts removes conflicting files based on the chosen style
// This prevents duplicate type declarations when switching between go_zero and gozero styles
func CleanupStyleConflicts(projectPath string, style string) error {
	// Determine which style's files to keep
	keepGoZeroStyle := (style == "go_zero")

	// Walk through project directories looking for conflicts
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Ignore "no such file" errors - we might have deleted the file during cleanup
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Check each known conflict pair
		for _, conflict := range knownStyleConflicts {
			goZeroFile := filepath.Join(path, conflict.GoZeroStyle)
			gozeroFile := filepath.Join(path, conflict.GoZeroFlat)

			// Check if both files exist (conflict!)
			goZeroExists := fileExists(goZeroFile)
			gozeroExists := fileExists(gozeroFile)

			if goZeroExists && gozeroExists {
				// Both exist - remove the one we don't want
				var fileToRemove string
				if keepGoZeroStyle {
					fileToRemove = gozeroFile
				} else {
					fileToRemove = goZeroFile
				}

				// Double-check file exists before removing
				if fileExists(fileToRemove) {
					if err := os.Remove(fileToRemove); err != nil {
						return fmt.Errorf("failed to remove conflicting file %s: %w", fileToRemove, err)
					}
				}
			}
		}

		return nil
	})

	return err
}

// DetectExistingStyle detects which naming style is currently used in the project
// Returns "go_zero" or "gozero", or empty string if cannot determine
func DetectExistingStyle(projectPath string) string {
	for _, conflict := range knownStyleConflicts {
		// Check in common locations
		commonDirs := []string{
			filepath.Join(projectPath, "internal", "svc"),
			filepath.Join(projectPath, "internal", "handler"),
			filepath.Join(projectPath, "internal", "logic"),
		}

		for _, dir := range commonDirs {
			goZeroFile := filepath.Join(dir, conflict.GoZeroStyle)
			gozeroFile := filepath.Join(dir, conflict.GoZeroFlat)

			if fileExists(goZeroFile) {
				return "go_zero"
			}
			if fileExists(gozeroFile) {
				return "gozero"
			}
		}
	}

	return ""
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// SuggestStyleBasedOnExisting suggests which style to use based on existing files
// Returns the detected style, or the provided default if no existing style detected
func SuggestStyleBasedOnExisting(projectPath string, defaultStyle string) string {
	if existingStyle := DetectExistingStyle(projectPath); existingStyle != "" {
		return existingStyle
	}
	return defaultStyle
}

// ValidateNoStyleConflicts checks if there are any style conflicts in the project
// Returns an error if conflicts are found
func ValidateNoStyleConflicts(projectPath string) error {
	var conflicts []string

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		for _, conflict := range knownStyleConflicts {
			goZeroFile := filepath.Join(path, conflict.GoZeroStyle)
			gozeroFile := filepath.Join(path, conflict.GoZeroFlat)

			if fileExists(goZeroFile) && fileExists(gozeroFile) {
				relPath, _ := filepath.Rel(projectPath, path)
				conflicts = append(conflicts, fmt.Sprintf("%s: both %s and %s exist",
					relPath, conflict.GoZeroStyle, conflict.GoZeroFlat))
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("style conflicts detected:\n%s", strings.Join(conflicts, "\n"))
	}

	return nil
}
