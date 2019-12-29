package vkg

import (
	"fmt"
	"log"
)

type Allocation struct {
	Offset uint64
	Size   uint64
}

func (a *Allocation) String() string {
	return fmt.Sprintf("[%d %d]", a.Offset, a.Size)
}

type IAllocator interface {
	Free(a *Allocation)
	Allocate(size uint64, align uint64) *Allocation
}

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

func (p *LinearAllocator) Allocate(size uint64, align uint64) *Allocation {
	log.Printf("%v", p.allocs)
	if len(p.allocs) == 0 {
		//There is nothing allocated, allocate here
		if size <= p.Size {
			p.allocs = make([]*Allocation, 0)
			na := &Allocation{Offset: 0, Size: size}
			p.allocs = append(p.allocs, na)
			return na
		} else {
			log.Println("Cant allocate", size)
			// If this pool isn't large enough return nil
			return nil
		}
	} else {
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

				log.Printf("%d %d", l, h)

				if h-l >= size {
					// FIXME: this should examine all possible allocation options and choose the best
					// Found an inter alloc allocation
					log.Printf("Found inter allocation %d %d %d", i, h-l, size)
					na := &Allocation{Offset: l, Size: size}

					p.allocs = append(p.allocs[:i+1], append([]*Allocation{na}, p.allocs[i+1:]...)...)
					return na
				}

			}
		}
		l := p.allocs[len(p.allocs)-1]
		nl := makeAlignUp(l.Offset+l.Size, align)
		log.Printf("Last %d %d", p.Size-nl, size)
		if p.Size-nl >= size {
			// Can we allocate from here to the end?
			na := &Allocation{Offset: nl, Size: size}
			p.allocs = append(p.allocs, na)
			return na
		} else {
			// if not then return nil
			return nil
		}
	}
	return nil
}

func (p *LinearAllocator) String() string {
	return fmt.Sprintf("%v", p.allocs)
}
