package utils

import (
	"fmt"
	"strings"
)

type FlagMap map[string]string

func (m *FlagMap) Set(value string) error {
	if *m == nil {
		*m = make(FlagMap)
	}
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) < 1 {
			return fmt.Errorf("invalid pair: %s", pair)
		}
		key := strings.TrimSpace(kv[0])
		value := ""
		if len(kv) == 2 {
			value = strings.TrimSpace(kv[1])
		}
		(*m)[key] = value
	}
	return nil
}

func (m *FlagMap) String() string {
	var pairs []string
	for k, v := range *m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}
