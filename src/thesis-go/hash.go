package main

import (
	"encoding/binary"
	"math/bits"
	"runtime"
	"sync"
)

const stateSize = 32
const stateStep = 16
const bitrate = stateStep * 4 // 64 bytes

// tree construction parameters
const leafBlocks = 256                // bitrate-blocks per leaf -> 16 KB per leaf (fits L1)
const leafSize = leafBlocks * bitrate // 16384 bytes
const treeFanout = 8                  // children per internal node
const nodeExtraRounds = 4             // extra chaos rounds after a node absorbs its children

// domain separation constants injected into the capacity portion of fresh sponges
const domainLeaf uint32 = 0x4C454146 // "LEAF"
const domainNode uint32 = 0x4E4F4445 // "NODE"

type State struct {
	data [32]uint32
}

func (state *State) diffuse() {
	for half := 0; half < 2; half++ {
		base := half * stateStep
		for index := 0; index < stateStep; index++ {
			next := (index + 1) % stateStep
			state.data[base+index] ^= bits.RotateLeft32(state.data[base+next], 13)
		}
	}
	for half := 0; half < 2; half++ {
		base := half * stateStep
		for index := stateStep - 1; index >= 0; index-- {
			prev := (index + stateStep - 1) % stateStep
			state.data[base+index] ^= bits.RotateLeft32(state.data[base+prev], 11)
		}
	}
	for index := 0; index < stateStep; index++ {
		state.data[index] ^= bits.RotateLeft32(state.data[stateStep+index], 7)
	}
	for index := 0; index < stateStep; index++ {
		state.data[stateStep+index] ^= bits.RotateLeft32(state.data[index], 19)
	}
}

func chaosRound(x, y uint32) (uint32, uint32) {
	// baker's map
	if x < 1<<31 {
		x = x << 1
		y = y >> 1
	} else {
		x = ^x << 1
		y = ^y>>1 | 1<<31
	}

	// gingerbreadman's map
	absX := x
	if x >= 1<<31 {
		absX = -x
	}
	return absX - y, x
}

func (state *State) chaos() {
	for index := 0; index < stateStep; index++ {
		bIndex := index + stateStep

		x := state.data[index]
		y := state.data[bIndex]

		for round := 0; round < 16; round++ {
			x, y = chaosRound(x, y)
		}

		state.data[index] = x
		state.data[bIndex] = y

		target := (index+1)%stateStep + stateStep
		state.data[target] ^= bits.RotateLeft32(x, 9)
		state.data[target] ^= bits.RotateLeft32(y, 17)
	}
	state.diffuse()
}

func (state *State) squeeze() []byte {
	out := make([]byte, stateStep*4)
	for index := range stateStep {
		binary.BigEndian.PutUint32(out[index*4:], state.data[index])
	}
	return out
}

func pad(data []byte, size int) []byte {
	// 10*1 padding: append 0x01, zeros, 0x01 to fill to size boundary
	padLen := size - 1 - (len(data) % size)
	if padLen < 1 {
		padLen += size
	}
	padded := make([]byte, len(data)+1+padLen)
	copy(padded, data)
	padded[len(data)] = 0x01
	padded[len(padded)-1] = 0x01
	return padded
}

// newDomainState returns a fresh sponge state with domain-separation constants
// XORed into the capacity portion (so they cannot be cancelled by absorbed input).
func newDomainState(domain uint32, level uint32) State {
	var s State
	s.data[stateStep] = domain
	s.data[stateStep+1] = level
	return s
}

// absorbInto absorbs a bitrate-aligned byte slice into an existing state.
func absorbInto(state *State, data []byte) {
	blockValues := make([]uint32, stateStep)
	for blockStart := 0; blockStart < len(data); blockStart += bitrate {
		block := data[blockStart : blockStart+bitrate]
		for i := range stateStep {
			blockValues[i] = binary.BigEndian.Uint32(block[i*4 : i*4+4])
		}
		for i := range stateStep {
			state.data[i] ^= blockValues[i]
		}
		state.chaos()
	}
}

// hashLeaf produces a leaf-level sponge state by absorbing a chunk of input.
func hashLeaf(chunk []byte) State {
	state := newDomainState(domainLeaf, 0)
	absorbInto(&state, chunk)
	return state
}

// hashNode produces an internal-node state by absorbing the serialized states
// of its children, then running a few extra chaos rounds.
func hashNode(children []State, level uint32) State {
	state := newDomainState(domainNode, level)
	buf := make([]byte, len(children)*stateSize*4)
	for i, c := range children {
		off := i * stateSize * 4
		for j, w := range c.data {
			binary.BigEndian.PutUint32(buf[off+j*4:], w)
		}
	}
	absorbInto(&state, buf)
	for r := 0; r < nodeExtraRounds; r++ {
		state.chaos()
	}
	return state
}

// parallelMap runs work(i) for i in [0,n) across a worker pool sized to GOMAXPROCS.
// The pool size affects only scheduling, not the values produced.
func parallelMap(n int, work func(i int)) {
	if n <= 0 {
		return
	}
	workers := runtime.GOMAXPROCS(0)
	if workers > n {
		workers = n
	}
	var wg sync.WaitGroup
	jobs := make(chan int, workers*2)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				work(i)
			}
		}()
	}
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}

func hash(input []byte, outputSize int, sectionCount int) []byte {
	_ = sectionCount // tree shape is determined by input length and fixed constants

	padded := pad(input, bitrate)

	// leaves: fixed-size chunks of leafSize bytes (the last leaf may be smaller
	// but is always a multiple of bitrate due to padding).
	numLeaves := (len(padded) + leafSize - 1) / leafSize
	if numLeaves < 1 {
		numLeaves = 1
	}

	leaves := make([]State, numLeaves)
	parallelMap(numLeaves, func(i int) {
		start := i * leafSize
		end := start + leafSize
		if end > len(padded) {
			end = len(padded)
		}
		leaves[i] = hashLeaf(padded[start:end])
	})

	// tree reduction: absorb fanout children per node, run extra chaos rounds,
	// repeat level by level until a single root state remains.
	level := uint32(1)
	nodes := leaves
	for len(nodes) > 1 {
		groupCount := (len(nodes) + treeFanout - 1) / treeFanout
		next := make([]State, groupCount)
		currentLevel := level
		current := nodes
		parallelMap(groupCount, func(i int) {
			start := i * treeFanout
			end := start + treeFanout
			if end > len(current) {
				end = len(current)
			}
			next[i] = hashNode(current[start:end], currentLevel)
		})
		nodes = next
		level++
	}

	finalState := nodes[0]

	intermediateRounds := 9 + (len(input)%37+outputSize%31)%11
	for round := 0; round < intermediateRounds; round++ {
		finalState.chaos()
	}

	// squeeze phase
	output := make([]byte, 0, outputSize)
	for len(output) < outputSize {
		output = append(output, finalState.squeeze()...)
		if len(output) < outputSize {
			finalState.chaos()
		}
	}
	return output[:outputSize]
}
