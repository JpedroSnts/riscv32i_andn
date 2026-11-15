package barramento

// Barramento liga a CPU à memória e aos dispositivos.
// Se o acesso falhar (endereço inexistente, fora do limite etc.),
// as funções devolvem um erro em vez de um valor inventado.
type Barramento interface {
	Read32(addr uint32) (uint32, error)
	Write32(addr uint32, val uint32) error
}
