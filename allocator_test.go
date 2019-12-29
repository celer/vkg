package vkg

import (
	"log"
	"testing"
)

func TestAlign(t *testing.T) {
	if makeAlignUp(12, 3) != 12 {
		t.Fail()
	}

	if makeAlignUp(10, 3) != 12 {
		t.Fail()
	}

}

func TestAllocator(t *testing.T) {

	a := PoolAllocator{Size: 1024, Align: 1}

	ra := a.Allocate(2048)
	if ra != nil {
		t.Error("Failed first allocation")
	}

	log.Printf("%v ", a.allocs)

	ra = a.Allocate(512)
	fa := ra
	if ra == nil {
		t.Error("Failed 2nd allocation")
	}

	ra = a.Allocate(768)
	if ra != nil {
		t.Error("Failed 3rd allocation")
	}

	ra = a.Allocate(500)
	k := ra
	if ra == nil {
		t.Error("Failed 4th allocation")
	}

	ra = a.Allocate(50)
	if ra != nil {
		t.Error("Failed 5th allocation")
	}

	ra = a.Allocate(5)
	if ra == nil {
		t.Error("Failed 6th allocation")
	}

	ra = a.Allocate(20)
	if ra != nil {
		t.Error("Failed 7th allocation")
	}

	a.Free(k)
	log.Printf("Free %s", a.String())
	ra = a.Allocate(500)
	if ra == nil {
		t.Error("Failed 8th allocation")
	}

	a.Free(fa)
	log.Printf(a.String())
	ra = a.Allocate(20)
	if ra == nil {
		t.Error("Failed 9th allocation")
	}

	ra = a.Allocate(40)
	if ra == nil {
		t.Error("Failed 10th allocation")
	}

	ra = a.Allocate(12)
	if ra == nil {
		t.Error("Failed 11th allocation")
	}
	ra = a.Allocate(500)
	if ra != nil {
		t.Error("Failed 12th allocation")
	}
	ra = a.Allocate(5)
	if ra == nil {
		t.Error("Failed 13th allocation")
	}
	log.Printf(a.String())
}
