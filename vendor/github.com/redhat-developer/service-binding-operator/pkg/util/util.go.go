package util

func MergeMaps(dest map[string]string, src map[string]string) map[string]string {
	for k, v := range src {
		dest[k] = v
	}
	return dest
}
