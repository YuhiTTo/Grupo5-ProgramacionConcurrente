package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type ResourceUsage struct {
	PeakHeapMB  float64
	AllocatedMB float64
	Mallocs     uint64
	GCCount     uint32
}

type ResourceProfileResult struct {
	Name           string
	Workers        int
	PeakHeapMB     float64
	AllocatedMB    float64
	AverageMallocs float64
	AverageGC      float64
}

func measureResourceUsage(run func()) ResourceUsage {
	runtime.GC()

	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	peakHeap := before.HeapAlloc

	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		var current runtime.MemStats

		for {
			select {

			case <-ticker.C:
				runtime.ReadMemStats(&current)

				if current.HeapAlloc > peakHeap {
					peakHeap = current.HeapAlloc
				}

			case <-done:
				return
			}
		}
	}()

	run()

	close(done)
	wg.Wait()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	if after.HeapAlloc > peakHeap {
		peakHeap = after.HeapAlloc
	}

	return ResourceUsage{
		PeakHeapMB: float64(peakHeap) /
			(1024 * 1024),

		AllocatedMB: float64(
			after.TotalAlloc-before.TotalAlloc,
		) / (1024 * 1024),

		Mallocs: after.Mallocs -
			before.Mallocs,

		GCCount: after.NumGC -
			before.NumGC,
	}
}

func profileSequentialResources(
	trainData []Sample,
	featureCount int,
	epochs int,
	learningRate float64,
	runs int,
) ResourceProfileResult {

	var peakHeapSum float64
	var allocatedSum float64
	var mallocSum uint64
	var gcSum uint64

	for run := 1; run <= runs; run++ {

		usage := measureResourceUsage(
			func() {
				model :=
					newLinearRegression(
						featureCount,
					)

				trainSequential(
					model,
					trainData,
					epochs,
					learningRate,
					false,
				)
			},
		)

		fmt.Printf(
			"Secuencial recurso %d/%d | Peak Heap: %.2f MB | Alloc: %.2f MB | GC: %d\n",
			run,
			runs,
			usage.PeakHeapMB,
			usage.AllocatedMB,
			usage.GCCount,
		)

		peakHeapSum += usage.PeakHeapMB
		allocatedSum += usage.AllocatedMB
		mallocSum += usage.Mallocs
		gcSum += uint64(usage.GCCount)
	}

	return ResourceProfileResult{
		Name:    "Secuencial",
		Workers: 0,

		PeakHeapMB: peakHeapSum / float64(runs),

		AllocatedMB: allocatedSum / float64(runs),

		AverageMallocs: float64(mallocSum) / float64(runs),

		AverageGC: float64(gcSum) / float64(runs),
	}
}

func profileConcurrentResources(
	trainData []Sample,
	featureCount int,
	epochs int,
	learningRate float64,
	workers int,
	runs int,
) ResourceProfileResult {

	var peakHeapSum float64
	var allocatedSum float64
	var mallocSum uint64
	var gcSum uint64

	for run := 1; run <= runs; run++ {

		usage := measureResourceUsage(
			func() {
				model :=
					newLinearRegression(
						featureCount,
					)

				trainConcurrent(
					model,
					trainData,
					epochs,
					learningRate,
					workers,
					false,
				)
			},
		)

		fmt.Printf(
			"%2d workers recurso %d/%d | Peak Heap: %.2f MB | Alloc: %.2f MB | GC: %d\n",
			workers,
			run,
			runs,
			usage.PeakHeapMB,
			usage.AllocatedMB,
			usage.GCCount,
		)

		peakHeapSum += usage.PeakHeapMB
		allocatedSum += usage.AllocatedMB
		mallocSum += usage.Mallocs
		gcSum += uint64(usage.GCCount)
	}

	return ResourceProfileResult{
		Name:    "Concurrente",
		Workers: workers,

		PeakHeapMB: peakHeapSum / float64(runs),

		AllocatedMB: allocatedSum / float64(runs),

		AverageMallocs: float64(mallocSum) / float64(runs),

		AverageGC: float64(gcSum) / float64(runs),
	}
}
