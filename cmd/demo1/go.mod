module github.com/rficu/rpr

go 1.16

replace github.com/rficu/rpr/pkg/rpr => ../../pkg/rpr

replace github.com/rficu/rpr/pkg/rtp => ../../pkg/rtp

replace github.com/rficu/rpr/pkg/connectivity => ../../pkg/connectivity

require github.com/rficu/rpr/pkg/rpr v0.0.0-00010101000000-000000000000

require (
	github.com/rficu/rpr/pkg/connectivity v0.0.0-00010101000000-000000000000
	github.com/wernerd/GoRTP v0.0.0-20191206100804-75c6a1c64532 // indirect
)
