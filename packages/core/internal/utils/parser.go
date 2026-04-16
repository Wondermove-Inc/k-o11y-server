package utils

import "strings"

func ParseEdge(edge string) (cluster, namespace, workload string) {
	parts := strings.Split(edge, "$$")

	if len(parts) >= 3 {
		cluster = parts[0]
		namespace = parts[1]
		workload = parts[2]
	}

	return cluster, namespace, workload
}
