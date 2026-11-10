package utils

import "testing"

func TestSet(t *testing.T) {
	set := NewSet[uint]()
	if SetContains(set, 0) {
		t.Error("set is not empty")
	}
	SetInsert(set, 0)
	if !SetContains(set, 0) {
		t.Error("set doesn't contain 0")
	}
	SetDelete(set, 0)
	if SetContains(set, 0) {
		t.Error("set is not empty")
	}
	for i := range 10 {
		SetInsert(set, uint(i))
	}
	for i := range 10 {
		if !SetContains(set, uint(i)) {
			t.Error("set doesn't contain", i)
		}
	}
	for range 10 {
		SetDelete(set, 0)
	}
	for i := range 9 {
		if !SetContains(set, uint(i+1)) {
			t.Error("set doesn't contain", i+1)
		}
		SetDelete(set, uint(i+1))
	}
	for i := range 10 {
		if SetContains(set, uint(i)) {
			t.Error("set contains", i)
		}
	}
}
