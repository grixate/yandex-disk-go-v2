package yadisk

import (
	"net/url"
	"strconv"
	"strings"
)

func addString(values url.Values, key, value string) {
	if value != "" {
		values.Set(key, value)
	}
}

func addCSV(values url.Values, key string, items []string) {
	if len(items) > 0 {
		values.Set(key, strings.Join(items, ","))
	}
}

func addInt(values url.Values, key string, v *int) {
	if v != nil {
		values.Set(key, strconv.Itoa(*v))
	}
}

func addInt64(values url.Values, key string, v int64) {
	if v != 0 {
		values.Set(key, strconv.FormatInt(v, 10))
	}
}

func addBool(values url.Values, key string, v *bool) {
	if v != nil {
		values.Set(key, strconv.FormatBool(*v))
	}
}
