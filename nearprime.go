package main

import (
	"fmt"
	"math/big"
	//"sync"
)

// Find the factors of a large near prime number.
const workchunksize = 1000000
const numworkers = 6

var np big.Int
var outdone chan struct{}
var inx chan big.Int

func main() {
	// First initialize np to the value of the target near prime.  The largest of these is the RSA100 number.
	//np.SetString("799", 10) // ./nearprime  0.00s user 0.00s system 1% cpu 0.262 total
	//np.SetString("37479454157", 10) // ./nearprime  0.01s user 0.00s system 3% cpu 0.265 total on MBPro
	//np.SetString("17684351495169528919", 10) // ./nearprime2  125.96s user 0.08s system 99% cpu 2:06.13 total
	np.SetString("11148760720422040092407", 10) // ./nearprime2  768.33s user 0.54s system 100% cpu 12:48.74 total
	//np.SetString("1522605027922533360535618378132637429718068114961380688657908494580122963258952897654000350692006139", 10)

	// Now find the smallest value x such that x squared is greater than the near prime.
	var sq, x big.Int
	fmt.Println("Near Prime:", &np)
	x.Sqrt(&np)

	sq.Mul(&x, &x)
	switch sq.Cmp(&np) {
	case -1:
		// sqrt is a floor function, convert result to ceiling
		x.Add(&x, big.NewInt(1))
		fmt.Println("Starting x:", &x)
	case 0:
		// Unlikely, but the near prime is a perfect square
		fmt.Println("First  factor:", &x)
		fmt.Println("Second factor:", &x)
		return
	case 1:
		panic("Should not be positive")
	}

	// At this point, we are going to look for the two distinct factors of the target np (near prime number).  We know this number has two distinct factors.
	// Since it also is given that it is the product of two large primes, it is obviously odd.  Observe that all composite numbers are of the form x^^2 - y^^2
	// or x(x+1) - y(y+1) for x > y and y >= 0.  We can rule out composite numbers of the second type since they are all even numbers.  So, we are looking
	// for a combination of x and y such that x^^2 - y^^2 == np.  We note that one of x and y must be even and the other odd for the resulting np to be odd.
	// We also note that x > np^^(1/2), which we calculated above.  x can be any value above that.  Also, we realize that since np is not a perfect square, we
	// can increment x by 1, and at each step increment y by 1 or 2.  We increment y by 1 when we increment x.  This flips y from E to O or O to E in reverse
	// phase to x.  This will not overshoot, as
	// (x+1)^^2 - x^^2 - ((y+1)^^2 - y^^2) = 2x - 2y > 0
	// since x is always greater than y.  For large targets, x is much greater than y.  We otherwise can increment y by two until we undershoot the target number
	// or hit it exactly when we find the factors.  We don't actually need to compute the factors until we have x^^2 - y^^2 == np.
	// We can also avoid computing the squares by simply adding in the correct value of x2delta and y2delta, which are the increments of the sequence of squares,
	// and themselves are a simple arithmatic sequence, and then updating those values.  At the end, we directly compute x and y from x2delta and y2delta respectively.
	// Then we compute the factors simply as x+y and x-y.
	//
	// The complexity of this algorithm as a function of the size of the target is linear over the range of y values checked, since each loop iteration is effectively incrementing y by either
	// 1 or 2 (depending on whether x is also incremented in that loop iteration or not).
	// To calculate a bound on runtime, first let the ratio of the two factors be k.  Since in practice the two factors are of the same order of magnitude (same number of digits)
	// the maximum range of k < 10.  The actual k may be less than that.  We will determine the upper bound on runtime in terms of K, the maximum value of k that we expect from a
	// solution f1*f2=N, where N is the target nearprime to be factored (np in the code).  Therefore x+y < K(x-y), i.e. x/y > (K+1)/(K-1).  x starts at r=N^^(1/2) and y essentially
	// at 0 (in the code we start the iteration of y at 1 or 2 depending on whether the intial value of x is odd or even, since we rule out the case where N is a square in the preliminary section),
	//  and then y increases potentially up to the limit of (K-1)/(K+1)x for the final value of x.  Let x = r + d, where r is the ceiling of the square root of N i.e. the starting value for x in
	// the iteration, and d is the amount x is incremented by to get the final solution that factors N.
	// Therefore, N = x^^2 - y^^2 = (x+y)(x-y) >= (x + (K-1)/(K+1)x)(x - (K-1)/(K+1)x) = 4Kx^^2/(K+1)^^2.  Taking the square root, r >= 2K^^(1/2)/(K+1) * x
	// Substituting x = r+d, and solving for d yields d <= (k^^(1/2)-1)^^2/2k^^(1/2) * r.
	// A practical value of K can be determined  by constraining the two factors to be within a decimal order of magnitude of each other, i.e. K=10.  This yields a limit on d <= 0.74r.
	// x = r + d <= 1.74r.  y <= (K-1)/(K+1)x = 1.42r.
	// Therefore, at the limit, x is incremented .73N^^(1/2) times, while y is incremented from 0 to 1.42N^^(1/2).  Since y is incremented by 1 each time x is incremented and by 2 otherwise, we
	// have at the limit N^^(1/2) * (0.73 + (1.42-0.73)/2) = 1.07N^^(1/2) loop iterations.
	//
	// Of course, if the factors are allowed to differ more widely, the runtime will increase as a function of
	// the actual ratio k of the two factors.  For widely differing factors, sieve techniques could discover the factors more quickly as they favor imbalanced factors.
	// E.g. for odd factors, the worst case for this algorithm
	// is O(N) iterations, as the resulting factors are N/3 and 3, which take N/3-N^^(1/2) + (N^^(1/2)-3)/2 ~= N/3 iterations to discover using this approach.
	//

	inx = make(chan big.Int, numworkers)
	outdone = make(chan struct{})

	bigincr := big.NewInt(workchunksize)

	for i := 0; i < numworkers; i++ {
		go npworker(workchunksize, i)
	}

	six := big.NewInt(6)
	var t big.Int
	for {
		if workdone() {
			break
		}
		if np.Cmp(t.Mul(&x, six)) < 0 {
			<-outdone
			break
		}
		var y big.Int
		y.Set(&x)
		inx <- y
		x.Add(&x, bigincr)
	}
	close(inx)
}

func workdone() bool {
	select {
	case <-outdone:
		return true
	default:
		return false
	}
}

func npworker(workchunk uint64, i int) {
	one := big.NewInt(1)
	two := big.NewInt(2)
	four := big.NewInt(4)

	drain := false
	for x := range inx {
		if drain {
			continue // drain the channel
		}
		if workdone() {
			break
		}
		//fmt.Println("x =", &x, "thread =", i)
		var t, y, x2delta, y2delta big.Int

		t.Mul(&x, &x)
		t.Sub(&t, &np)
		y.Sqrt(&t) // Floor of square root of difference x squared - N

		if x.Bit(0) == 1 {
			y.SetBit(&y, 0, 0)
		} else {
			if y.Bit(0) == 0 {
				y.Sub(&y, one)
			}
		}

		x2delta.Add(&x, x2delta.Add(&x, one))
		y2delta.Add(&y, y2delta.Add(&y, one))
		y.Mul(&y, &y)
		t.Sub(&t, &y)

		var i uint64
	loop:
		for i = 0; i < workchunk; i++ {
			switch t.Sign() {
			case 1:
				// Normal case, need to increase y squared to (y+2) squared
				t.Sub(t.Sub(t.Sub(&t, &y2delta), &y2delta), two)
				y2delta.Add(&y2delta, four)
			case -1:
				// Second most common case, increment both x and y by 1, preserving EO or OE relationship between them
				t.Add(&t, &x2delta)
				t.Sub(&t, &y2delta)
				x2delta.Add(&x2delta, two)
				y2delta.Add(&y2delta, two)
			case 0:
				// Eureka, we found the factors.  Reconstruct x and y, then contruct f1 and f2
				var f1, f2 big.Int
				x2delta.Sub(&x2delta, one)
				x2delta.Rsh(&x2delta, 1)
				y2delta.Sub(&y2delta, one)
				y2delta.Rsh(&y2delta, 1)
				f1.Sub(&x2delta, &y2delta)
				f2.Add(&x2delta, &y2delta)
				fmt.Println("x: ", &x2delta)
				fmt.Println("y: ", &y2delta)
				fmt.Println("First factor is:", &f1)
				fmt.Println("Second factor is:", &f2)
				fmt.Println("Verifier =", f1.Mul(&f1, &f2))
				close(outdone)
				drain = true
				break loop
			}
		}
	}
}
