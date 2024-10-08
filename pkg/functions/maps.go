package functions

import "strings"

// MapSet sets a key, value in a map. If a map is nil, it firsts initializes the map
func mapSet[K comparable, V any](m *map[K]V, key K, value V) {
	if *m == nil {
		*m = make(map[K]V)
	}
	(*m)[key] = value
}

// MapSet sets a key, value in a map. If a map is nil, it firsts initializes the map
func MapSet[T any](m *map[string]T, key string, value T) {
	mapSet(m, key, value)
}

// MapContains checks if `destination` contains all keys from `source`
func MapContains[T comparable](destination map[string]T, source map[string]T) bool {
	if len(destination) == 0 && len(source) == 0 {
		return true
	}

	for k, v := range source {
		if destination[k] != v {
			return false
		}
	}
	return true
}

func MapEqual[K comparable, V comparable](first map[K]V, second map[K]V) bool {
	if len(first) != len(second) {
		return false
	}

	for k := range first {
		if second[k] != first[k] {
			return false
		}
	}
	return true
}

func MapHasKey[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}

func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func MapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func MapFilterWithPrefix[K string, V any](m map[K]V, keyPrefix string) map[K]V {
	result := make(map[K]V, len(m)/2)
	for k, v := range m {
		if strings.HasPrefix(string(k), keyPrefix) {
			result[k] = v
		}
	}

	return result
}

func MapFilter[K string, V any](m map[K]V, filters ...func(k K, v V) bool) map[K]V {
	result := make(map[K]V, len(m))
	for k, v := range m {
		shouldAdd := true
		for _, f := range filters {
			if !f(k, v) {
				shouldAdd = false
				break
			}
		}
		if shouldAdd {
			result[k] = v
		}
	}
	return result
}

func MapJoin[K comparable, V any](first *map[K]V, second map[K]V) {
	for k, v := range second {
		mapSet(first, k, v)
	}
}
