package vkg

import (
	"fmt"
	"log"
)

// Allocation is an allocation of some hunk of memory
type Allocation struct {
	// Offset in memory to this object
	Offset uint64
	// Size of the allocated memory
	Size uint64

	// The item that was allocated
	Object IAllocatedItem
}

// String for stringer interface
func (a *Allocation) String() string {
	return fmt.Sprintf("{Offset:%d Size:%d Object:%v}", a.Offset, a.Size, a.Object)
}

// LinearAllocator is a basic linear allocator for
// memory, it simply allocates blocks of memory
// and will return the first allocation that
// that fits the request, it's not ideal, but works.
type LinearAllocator struct {
	Size   uint64
	allocs []*Allocation
}

func makeAlignUp(a uint64, align uint64) uint64 {
	m := a % align
	if m == 0 {
		return a
	}
	a = (a - m) + align
	return a
}

func (p *LinearAllocator) DestroyContents() {
	for _, alloc := range p.allocs {
		alloc.Object.Destroy()
	}
}

func (p *LinearAllocator) Allocations() []*Allocation {
	return p.allocs
}

// Free the specified allocation
func (p *LinearAllocator) Free(fa *Allocation) {
	fi := -1
	for i, a := range p.allocs {
		if a == fa {
			fi = i
		}
	}
	if fi != -1 {
		p.allocs = append(p.allocs[:fi], p.allocs[fi+1:]...)
	}
}

// Allocate a new hunk of memory
func (p *LinearAllocator) Allocate(size uint64, align uint64) *Allocation {
	if len(p.allocs) == 0 {
		//There is nothing allocated, allocate here
		if size <= p.Size {
			p.allocs = make([]*Allocation, 0)
			na := &Allocation{Offset: 0, Size: size}
			p.allocs = append(p.allocs, na)
			return na
		}
		// If this pool isn't large enough return nil
		return nil
	}
	// We can insert at the head of the block
	if p.allocs[0].Offset > size {
		na := &Allocation{Offset: 0, Size: size}
		p.allocs = append([]*Allocation{na}, p.allocs...)
		return na
	}

	for i := 0; i < len(p.allocs); i++ {
		c := p.allocs[i]
		if i+1 < len(p.allocs) {
			n := p.allocs[i+1]

			l := makeAlignUp(c.Offset+c.Size, align)
			h := n.Offset

			if h-l >= size {
				// FIXME: this should examine all possible allocation options and choose the best
				// Found an inter alloc allocation
				na := &Allocation{Offset: l, Size: size}

				p.allocs = append(p.allocs[:i+1], append([]*Allocation{na}, p.allocs[i+1:]...)...)
				return na
			}

		}
	}
	l := p.allocs[len(p.allocs)-1]
	nl := makeAlignUp(l.Offset+l.Size, align)
	if p.Size-nl >= size {
		// Can we allocate from here to the end?
		na := &Allocation{Offset: nl, Size: size}
		p.allocs = append(p.allocs, na)
		return na
	}
	// if not then return nil
	return nil
}

func (p *LinearAllocator) LogDetails() {
	for _, alloc := range p.allocs {
		log.Printf("\t %v", alloc)
	}
}

// String for stringer interface
func (p *LinearAllocator) String() string {
	return fmt.Sprintf("%v", p.allocs)
}
