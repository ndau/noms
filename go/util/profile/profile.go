// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package profile

import (
	"io"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/attic-labs/kingpin"

	"github.com/ndau/noms/go/d"
)

var (
	cpuProfile   string
	memProfile   string
	blockProfile string
)

func RegisterProfileFlags(app *kingpin.Application) {
	// Must reset globals because under test this can get called multiple times.
	cpuProfile = ""
	memProfile = ""
	blockProfile = ""
	app.Flag("cpuprofile", "write cpu profile to file").StringVar(&cpuProfile)
	app.Flag("memprofile", "write memory profile to file").StringVar(&memProfile)
	app.Flag("blockprofile", "write block profile to file").StringVar(&blockProfile)
}

// MaybeStartProfile checks the -blockProfile, -cpuProfile, and -memProfile flag and, for each that is set, attempts to start gathering profiling data into the appropriate files. It returns an object with one method, Stop(), that must be called in order to flush profile data to disk before the process terminates.
func MaybeStartProfile() interface {
	Stop()
} {
	p := &prof{}
	if blockProfile != "" {
		f, err := os.Create(blockProfile)
		d.PanicIfError(err)
		runtime.SetBlockProfileRate(1)
		p.bp = f
	}
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		d.PanicIfError(err)
		pprof.StartCPUProfile(f)
		p.cpu = f
	}
	if memProfile != "" {
		f, err := os.Create(memProfile)
		d.PanicIfError(err)
		p.mem = f
	}
	return p
}

type prof struct {
	bp  io.WriteCloser
	cpu io.Closer
	mem io.WriteCloser
}

func (p *prof) Stop() {
	if p.bp != nil {
		pprof.Lookup("block").WriteTo(p.bp, 0)
		p.bp.Close()
		runtime.SetBlockProfileRate(0)
	}
	if p.cpu != nil {
		pprof.StopCPUProfile()
		p.cpu.Close()
	}
	if p.mem != nil {
		pprof.WriteHeapProfile(p.mem)
		p.mem.Close()
	}
}
