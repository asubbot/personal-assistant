package main

import "pa/cmd/pa/wire"

// Type aliases keep EP-027/EP-029 tests stable while the composition root lives in wire (EP-042).
type (
	paApplication    = wire.Application
	paInfrastructure = wire.Infrastructure
)
