package service

import "strconv"

func strPtr(s string) *string {
	return &s
}

func stringPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func nilableString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func int64PtrEqual(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func int64PtrToStrPtr(v *int64) *string {
	if v == nil {
		return nil
	}
	s := strconv.FormatInt(*v, 10)
	return &s
}

func nilableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}
