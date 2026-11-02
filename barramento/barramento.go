package barramento

type Barramento interface {
	Read32 (addr uint32) uint32
	Write32 (addr uint32, val uint32)
}
