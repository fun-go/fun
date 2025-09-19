package fun

import (
	"reflect"
	"strings"
)

type Tag struct {
	TagList map[string]string
}

func newTag(tag reflect.StructTag) *Tag {
	t := &Tag{
		TagList: map[string]string{},
	}
	pairs := strings.Split(strings.TrimSpace(tag.Get("fun")), ";")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		keyValue := strings.Split(pair, ":")
		if len(keyValue) == 1 {
			t.TagList[keyValue[0]] = ""
		} else {
			t.TagList[keyValue[0]] = keyValue[1]
		}
	}
	return t
}

func (tag *Tag) getTag(key string) (string, bool) {
	v, ok := tag.TagList[key]
	return v, ok
}
