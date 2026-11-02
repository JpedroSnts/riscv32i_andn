package cpu

import "riscv/barramento"

type CPU struct {
	Regs[32] uint32
	PC uint32
	Barramento barramento.Barramento
}

func New(b barramento.Barramento) * CPU {
	return &CPU{Barramento: b}
}
