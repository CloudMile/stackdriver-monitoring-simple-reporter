package utils

import "fmt"

func CPUValueFormatter(v interface{}) string {
	typed, _ := v.(float64)
	unit := "ms/s"

	if typed > 1000 {
		typed = typed / 1000
		unit = " s/s"
	}

	return fmt.Sprintf("+%6.2f%s", typed, unit)
}

func MemoryValueFormatter(v interface{}) string {
	typed, _ := v.(float64)
	unit := " B"

	// KB
	if typed > 1000 {
		typed = typed / 1024
		unit = "KB"
	}

	// MB
	if typed > 1000 {
		typed = typed / 1024
		unit = "MB"
	}

	// GB
	if typed > 1000 {
		typed = typed / 1024
		unit = "GB"
	}

	return fmt.Sprintf("+%8.2f%s", typed, unit)
}
