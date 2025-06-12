package host

import (
	"context"
	"fmt"
	"sync"

	"slices"

	"github.com/tetratelabs/wazero/api"
)

const (
	PageSize = 4096
)

// FreeBlock represents a free memory block within a page
type FreeBlock struct {
	Offset uint32 // Offset from page start
	Size   uint32 // Size of the free block
}

type AllocatorPage struct {
	StartAddress uint32
	FreeBlocks   []FreeBlock // Sorted by offset
}

type Allocator struct {
	Pages []AllocatorPage
	Mutex sync.Mutex
}

func NewAllocator() *Allocator {
	return &Allocator{
		Pages: []AllocatorPage{},
		Mutex: sync.Mutex{},
	}
}

func (a *Allocator) Alloc(ctx context.Context, module api.Module, size uint32) (uint32, error) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	if size == 0 || size > PageSize {
		return 0, fmt.Errorf("invalid allocation size: %d", size)
	}

	// Align to 8 bytes for better memory alignment
	alignedSize := (size + 7) & ^uint32(7)

	// Try to find a suitable free block in existing pages
	for i := range a.Pages {
		if addr, found := a.allocateInPage(&a.Pages[i], alignedSize); found {
			return addr, nil
		}
	}

	// No suitable block found, allocate a new page
	_, err := a.allocateNewPage(ctx, module)
	if err != nil {
		return 0, err
	}

	// Allocate in the new page
	page := &a.Pages[len(a.Pages)-1]
	addr, found := a.allocateInPage(page, alignedSize)
	if !found {
		return 0, fmt.Errorf("failed to allocate in new page")
	}

	return addr, nil
}

func (a *Allocator) Free(addr uint32) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	// Find which page this address belongs to
	for i := range a.Pages {
		page := &a.Pages[i]
		if addr >= page.StartAddress && addr < page.StartAddress+PageSize {
			a.freeInPage(page, addr)
			return
		}
	}
}

func (a *Allocator) allocateNewPage(ctx context.Context, module api.Module) (uint32, error) {
	// Call the WASM function to allocate a new page
	allocPage := module.ExportedFunction("alloc_page")
	results, err := allocPage.Call(ctx)
	if err != nil {
		return 0, err
	}

	if len(results) != 1 {
		return 0, fmt.Errorf("AllocPage returned %d results, expected 1", len(results))
	}

	pageAddr := uint32(results[0])

	// Create new page with one large free block
	newPage := AllocatorPage{
		StartAddress: pageAddr,
		FreeBlocks: []FreeBlock{
			{Offset: 0, Size: PageSize},
		},
	}

	a.Pages = append(a.Pages, newPage)
	return pageAddr, nil
}

func (a *Allocator) allocateInPage(page *AllocatorPage, size uint32) (uint32, bool) {
	// Find first free block that can fit the allocation
	for i, block := range page.FreeBlocks {
		if block.Size >= size {
			addr := page.StartAddress + block.Offset

			// Remove or split the free block
			if block.Size == size {
				// Remove the entire block
				page.FreeBlocks = slices.Delete(page.FreeBlocks, i, i+1)
			} else {
				// Split the block
				page.FreeBlocks[i] = FreeBlock{
					Offset: block.Offset + size,
					Size:   block.Size - size,
				}
			}

			return addr, true
		}
	}

	return 0, false
}

func (a *Allocator) freeInPage(page *AllocatorPage, addr uint32) {
	offset := addr - page.StartAddress

	// We need to determine the size of the allocation to free it
	// Since we don't track allocation sizes, we need to reconstruct
	// the free block. For simplicity, we'll assume the caller knows
	// the size, but since the interface doesn't provide it, we'll
	// need to track allocations.

	// For now, let's add a method that tracks allocation sizes
	// This is a limitation of the current interface

	// Find the size by looking at the next allocation or end of page
	size := a.findAllocationSize(page, offset)

	// Insert the free block in the correct position (sorted by offset)
	newBlock := FreeBlock{Offset: offset, Size: size}

	// Find insertion point
	insertPos := 0
	for i, block := range page.FreeBlocks {
		if block.Offset > offset {
			insertPos = i
			break
		}
		insertPos = i + 1
	}

	// Insert the block
	page.FreeBlocks = append(page.FreeBlocks[:insertPos],
		append([]FreeBlock{newBlock}, page.FreeBlocks[insertPos:]...)...)

	// Merge with adjacent blocks
	a.mergeAdjacentBlocks(page, insertPos)
}

func (a *Allocator) findAllocationSize(page *AllocatorPage, offset uint32) uint32 {
	// This is a simplified approach - in a real implementation,
	// we'd need to track allocation sizes or use a different strategy

	// Find the next free block or end of page
	nextFreeOffset := uint32(PageSize)
	for _, block := range page.FreeBlocks {
		if block.Offset > offset && block.Offset < nextFreeOffset {
			nextFreeOffset = block.Offset
		}
	}

	// Find the previous free block end
	prevFreeEnd := uint32(0)
	for _, block := range page.FreeBlocks {
		blockEnd := block.Offset + block.Size
		if blockEnd <= offset && blockEnd > prevFreeEnd {
			prevFreeEnd = blockEnd
		}
	}

	// The allocation starts at max(prevFreeEnd, offset) and ends at nextFreeOffset
	start := max(prevFreeEnd, offset)

	return nextFreeOffset - start
}

func (a *Allocator) mergeAdjacentBlocks(page *AllocatorPage, pos int) {
	if pos >= len(page.FreeBlocks) {
		return
	}

	// Merge with next block
	for pos < len(page.FreeBlocks)-1 {
		current := &page.FreeBlocks[pos]
		next := &page.FreeBlocks[pos+1]

		if current.Offset+current.Size == next.Offset {
			// Merge current with next
			current.Size += next.Size
			page.FreeBlocks = slices.Delete(page.FreeBlocks, pos+1, pos+2)
		} else {
			break
		}
	}

	// Merge with previous block
	for pos > 0 {
		current := &page.FreeBlocks[pos]
		prev := &page.FreeBlocks[pos-1]

		if prev.Offset+prev.Size == current.Offset {
			// Merge previous with current
			prev.Size += current.Size
			page.FreeBlocks = slices.Delete(page.FreeBlocks, pos, pos+1)
			pos--
		} else {
			break
		}
	}
}

// Additional helper method to get allocator statistics
func (a *Allocator) Stats() (totalPages int, totalFree uint32, totalAllocated uint32) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	totalPages = len(a.Pages)

	for _, page := range a.Pages {
		for _, block := range page.FreeBlocks {
			totalFree += block.Size
		}
	}

	totalAllocated = uint32(totalPages)*PageSize - totalFree
	return
}
