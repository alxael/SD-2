package main

import (
	"math"
	mathrand "math/rand"
	"runtime"
	"strconv"
	"sync"
)

// This harness estimates the Lyapunov spectrum (lambda_1, lambda_2) of two
// classic piecewise-linear two-dimensional chaotic maps using the Benettin
// algorithm: the reference trajectory is evolved while two perturbation
// vectors are propagated through the analytic Jacobian and re-orthonormalised
// (Gram-Schmidt) every step to avoid overflow and directional collapse.
//
//   - Baker's map: dissipative-looking but area preserving on each branch;
//     the Jacobian is the constant matrix diag(2, 1/2), so the spectrum is
//     known exactly: lambda_1 = ln 2, lambda_2 = -ln 2.
//   - Gingerbreadman map: area preserving (|det J| = 1), so lambda_1 + lambda_2
//     should converge to 0. A positive lambda_1 confirms a chaotic initial
//     condition (as opposed to one trapped on a periodic island).
//
// The produced report stores the running estimates of both maps at log-spaced
// iteration counts so a single graph can overlay their convergence behaviour.

// chaoticMap bundles the dynamics, the linearisation, and an initial-condition
// sampler for a 2D map.
type chaoticMap struct {
	name     string
	step     func(x, y float64) (float64, float64)
	jacobian func(x, y float64) [2][2]float64
	sample   func(rng *mathrand.Rand) (float64, float64)
}

// baker's map on the unit square.
//
//	(x, y) -> (2x,     y/2    )  if x < 1/2
//	          (2x - 1, y/2 + 1/2) if x >= 1/2
func bakerStep(x, y float64) (float64, float64) {
	if x < 0.5 {
		return 2 * x, y / 2
	}
	return 2*x - 1, y/2 + 0.5
}

// The baker's Jacobian is constant everywhere except on the cut line x = 1/2,
// which a trajectory hits with probability zero.
func bakerJacobian(_, _ float64) [2][2]float64 {
	return [2][2]float64{
		{2, 0},
		{0, 0.5},
	}
}

// gingerbreadman map.
//
//	x_{n+1} = 1 - y_n + |x_n|
//	y_{n+1} = x_n
func gingerbreadmanStep(x, y float64) (float64, float64) {
	return 1 - y + math.Abs(x), x
}

// The only non-smoothness is the |x| fold at x = 0; away from it the Jacobian
// depends solely on sgn(x).
func gingerbreadmanJacobian(x, _ float64) [2][2]float64 {
	sign := 1.0
	if x < 0 {
		sign = -1.0
	}
	return [2][2]float64{
		{sign, -1},
		{1, 0},
	}
}

// uint32Scale normalises a uint32 state coordinate into [0, 1).
const uint32Scale = 1 << 32

// hashChaosRound advances the integer state exactly like chaosRound in hash.go
// (baker's map followed by the gingerbreadman's map) and additionally returns
// the Jacobian of that composed round in normalised coordinates. The Jacobian
// is the product J_G * J_B, evaluated via the branch the integer state selects:
//
//   - folded baker: J_B = diag(2, 1/2) on the low branch, diag(-2, -1/2) on the
//     complement branch (|det| = 1 either way);
//   - gingerbreadman: J_G = [[sgn, -1], [1, 0]] with sgn fixed by the high bit
//     of the baker output.
//
// Running the trajectory in exact uint32 arithmetic (rather than float64) avoids
// the doubling-map mantissa collapse, so the measured itinerary is the one the
// hash actually follows.
func hashChaosRound(x, y uint32) (uint32, uint32, [2][2]float64) {
	// baker's map
	var jb [2][2]float64
	if x < 1<<31 {
		x = x << 1
		y = y >> 1
		jb = [2][2]float64{{2, 0}, {0, 0.5}}
	} else {
		x = ^x << 1
		y = ^y>>1 | 1<<31
		jb = [2][2]float64{{-2, 0}, {0, -0.5}}
	}

	// gingerbreadman's map
	sign := 1.0
	absX := x
	if x >= 1<<31 {
		absX = -x
		sign = -1.0
	}
	jg := [2][2]float64{{sign, -1}, {1, 0}}

	return absX - y, x, matMul2(jg, jb)
}

// lyapunovResult holds the final spectrum estimate for a single random start.
type lyapunovResult struct {
	x0, y0           float64
	lambda1, lambda2 float64
}

func matVec2(m [2][2]float64, v [2]float64) [2]float64 {
	return [2]float64{
		m[0][0]*v[0] + m[0][1]*v[1],
		m[1][0]*v[0] + m[1][1]*v[1],
	}
}

func matMul2(a, b [2][2]float64) [2][2]float64 {
	return [2][2]float64{
		{a[0][0]*b[0][0] + a[0][1]*b[1][0], a[0][0]*b[0][1] + a[0][1]*b[1][1]},
		{a[1][0]*b[0][0] + a[1][1]*b[1][0], a[1][0]*b[0][1] + a[1][1]*b[1][1]},
	}
}

func dot2(u, v [2]float64) float64 { return u[0]*v[0] + u[1]*v[1] }
func norm2(v [2]float64) float64   { return math.Sqrt(dot2(v, v)) }

// estimateFinalSpectrum runs the Benettin algorithm from (x0, y0) and returns
// the final (lambda_1, lambda_2) estimate. The boolean is false when the orbit
// escapes to infinity, which the gingerbreadman map permits for starts outside
// its bounded region.
func estimateFinalSpectrum(m chaoticMap, x0, y0 float64, transient, steps int) (lyapunovResult, bool) {
	const divergenceBound = 1e6

	x, y := x0, y0

	// Discard the transient so the estimate is taken on the attractor.
	for i := 0; i < transient; i++ {
		x, y = m.step(x, y)
		if diverged(x, y, divergenceBound) {
			return lyapunovResult{}, false
		}
	}

	// Orthonormal perturbation vectors spanning the tangent space.
	v1 := [2]float64{1, 0}
	v2 := [2]float64{0, 1}

	var sum1, sum2 float64
	for i := 1; i <= steps; i++ {
		j := m.jacobian(x, y)

		// Propagate perturbations through the linearised dynamics.
		v1 = matVec2(j, v1)
		v2 = matVec2(j, v2)

		// Gram-Schmidt re-orthonormalisation.
		n1 := norm2(v1)
		v1[0] /= n1
		v1[1] /= n1

		proj := dot2(v2, v1)
		v2[0] -= proj * v1[0]
		v2[1] -= proj * v1[1]
		n2 := norm2(v2)
		v2[0] /= n2
		v2[1] /= n2

		// Accumulate logarithmic growth rates.
		sum1 += math.Log(n1)
		sum2 += math.Log(n2)

		// Advance the reference trajectory.
		x, y = m.step(x, y)
		if diverged(x, y, divergenceBound) {
			return lyapunovResult{}, false
		}
	}

	return lyapunovResult{
		x0:      x0,
		y0:      y0,
		lambda1: sum1 / float64(steps),
		lambda2: sum2 / float64(steps),
	}, true
}

// diverged reports whether the orbit has left the bounded region or gone NaN.
func diverged(x, y, bound float64) bool {
	return math.IsNaN(x) || math.IsNaN(y) || math.Abs(x) > bound || math.Abs(y) > bound
}

// runRandomTrials samples `attempts` random initial conditions for the map and
// returns the final spectrum of every orbit that stayed bounded. Trials are
// distributed across all available cores; each worker owns an independent RNG
// so the run is deterministic for a given seed.
func runRandomTrials(m chaoticMap, attempts, transient, steps int, seed int64) []lyapunovResult {
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan int, workers)
	resultsCh := make(chan lyapunovResult, attempts)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := mathrand.New(mathrand.NewSource(seed + int64(workerID)))
			for range jobs {
				x0, y0 := m.sample(rng)
				if result, ok := estimateFinalSpectrum(m, x0, y0, transient, steps); ok {
					resultsCh <- result
				}
			}
		}(w)
	}

	go func() {
		for i := 0; i < attempts; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	results := make([]lyapunovResult, 0, attempts)
	for result := range resultsCh {
		results = append(results, result)
	}
	return results
}

// estimateHashChaosSpectrum runs the Benettin algorithm on the composed
// baker -> gingerbreadman round used by hash.go, starting from the integer
// state (x0, y0). The trajectory is advanced in exact uint32 arithmetic; the
// reported initial condition is normalised into [0, 1).
func estimateHashChaosSpectrum(x0, y0 uint32, transient, steps int) lyapunovResult {
	x, y := x0, y0

	// Discard the transient.
	for i := 0; i < transient; i++ {
		x, y, _ = hashChaosRound(x, y)
	}

	// Orthonormal perturbation vectors spanning the tangent space.
	v1 := [2]float64{1, 0}
	v2 := [2]float64{0, 1}

	var sum1, sum2 float64
	for i := 1; i <= steps; i++ {
		var j [2][2]float64
		x, y, j = hashChaosRound(x, y)

		// Propagate perturbations through the linearised dynamics.
		v1 = matVec2(j, v1)
		v2 = matVec2(j, v2)

		// Gram-Schmidt re-orthonormalisation.
		n1 := norm2(v1)
		v1[0] /= n1
		v1[1] /= n1

		proj := dot2(v2, v1)
		v2[0] -= proj * v1[0]
		v2[1] -= proj * v1[1]
		n2 := norm2(v2)
		v2[0] /= n2
		v2[1] /= n2

		sum1 += math.Log(n1)
		sum2 += math.Log(n2)
	}

	return lyapunovResult{
		x0:      float64(x0) / uint32Scale,
		y0:      float64(y0) / uint32Scale,
		lambda1: sum1 / float64(steps),
		lambda2: sum2 / float64(steps),
	}
}

// runHashChaosTrials samples `attempts` random uint32 initial states for the
// composed hash round and returns the final spectrum of each, distributed
// across all cores with per-worker deterministic RNGs.
func runHashChaosTrials(attempts, transient, steps int, seed int64) []lyapunovResult {
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan int, workers)
	resultsCh := make(chan lyapunovResult, attempts)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := mathrand.New(mathrand.NewSource(seed + int64(workerID)))
			for range jobs {
				resultsCh <- estimateHashChaosSpectrum(rng.Uint32(), rng.Uint32(), transient, steps)
			}
		}(w)
	}

	go func() {
		for i := 0; i < attempts; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	results := make([]lyapunovResult, 0, attempts)
	for result := range resultsCh {
		results = append(results, result)
	}
	return results
}

func generateChaoticMapsTest() {
	const (
		bakerAttempts          = 50000
		gingerbreadmanAttempts = 50000
		hashAttempts           = 50000
		transient              = 1000
		steps                  = 100_000
	)

	baker := chaoticMap{
		name:     "baker",
		step:     bakerStep,
		jacobian: bakerJacobian,
		// The baker's map lives on the unit square.
		sample: func(rng *mathrand.Rand) (float64, float64) {
			return rng.Float64(), rng.Float64()
		},
	}

	gingerbreadman := chaoticMap{
		name:     "gingerbreadman",
		step:     gingerbreadmanStep,
		jacobian: gingerbreadmanJacobian,
		// Sample the box [-10, 10]^2; escaping starts are discarded by
		// estimateFinalSpectrum.
		sample: func(rng *mathrand.Rand) (float64, float64) {
			return -10 + rng.Float64()*20, -10 + rng.Float64()*20
		},
	}

	bakerResults := runRandomTrials(baker, bakerAttempts, transient, steps, 1)
	gingerbreadmanResults := runRandomTrials(gingerbreadman, gingerbreadmanAttempts, transient, steps, 2)
	hashResults := runHashChaosTrials(hashAttempts, transient, steps, 3)

	writeChaoticMapResults("test-chaotic-maps-baker", bakerResults)
	writeChaoticMapResults("test-chaotic-maps-gingerbreadman", gingerbreadmanResults)
	writeChaoticMapResults("test-chaotic-maps-hash", hashResults)
}

// writeChaoticMapResults stores the random-start spectrum for a single map.
func writeChaoticMapResults(reportName string, results []lyapunovResult) {
	writer, file, err := generateCsvReportFile(reportName)
	if err != nil {
		return // fail silently, matching the other harnesses
	}
	defer file.Close()
	defer writer.Flush()

	writer.Write([]string{"x0", "y0", "lambda1", "lambda2"})

	for _, r := range results {
		writer.Write([]string{
			strconv.FormatFloat(r.x0, 'f', 6, 64),
			strconv.FormatFloat(r.y0, 'f', 6, 64),
			strconv.FormatFloat(r.lambda1, 'f', 6, 64),
			strconv.FormatFloat(r.lambda2, 'f', 6, 64),
		})
	}
}
