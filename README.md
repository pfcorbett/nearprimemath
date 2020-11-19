# nearprimemath
go program to factor large near prime numbers
Several years ago, I was teaching my young son the multiplication table.  He's quite mathematical, so I was looking for patterns I could show him.  In doing so,
I observed that all non-prime numbers take one of two forms: x^^2 - y^^2 or x(x+1) - y(y+1).  I'm not a mathematician by trade, but I've always enjoyed working with
numbers.  I tucked this away as an interesting fact that I had never been taught in my math education.
More recently, I was looking for a good problem to solve using go, and specifically using go's math/big package.  I realized that factoring large near primes has
been a challenge problem, with the RSA numbers being posed as a set of hard to factor near primes - multiples of two large primes.  I wondered if I could apply
the difference of squares technique to discovering the factors of a large nearprime.  (Jump to the conclusion, yes, but not efficiently enough to factor numbers as
large as the RSA numbers).  This has been an active area of mathematics and cryptography for decades, so I'm not expecting to break any new ground.  

In any case, I wrote the program nearprime.go to try this approach.  The basic approach is this:

We want to find x and y, x>0, x>y>=0, such that x^^2 - y^^2 = N, our target number to factor. We know that N is of the form x^^2 - y^^2, since N, being the product of
two large prime numbers, is odd.  All numbers of the form x(x+1) - y(y+1) are of course even.

N is known to be a near-prime, i.e. the product of two large prime numbers.  We start with x as the ceiling of N^^(1/2) (square root of N), and y = 0.  We also know
that when x is even, y is odd, and vice versa to produce an odd result.  Our approach is to increment x, set y to the next value so that x,y are E,O or O,E.  
Test if x^^2 - y^^2 == N.  Then advance y by two until the difference x^^2 - y^^2 is less than N.  Then advance x and y both by 1, which makes the error
positive again, and repeat by incrementing y.

Now, we optimize in a couple of ways.  We first note that for any x and y, x^^2 - y^^2 = N + e, where e is the "error".  Rearranging, x^^2 - y^^2 - N = e.  We can
initialize x and y as above and compute an initial error value e.  We are then going to iterate until we find an x y combination that yields e == 0.
This assists the performance as the math/big package uses a variable length slice []uint64 to store the magnitude (abs) of the number.  Keeping the math close to zero
will ensure that we are doing fewer operations in each math step.

Second, we note that we do not need to compute the squares each time.  Since a sequence of squares is easily calculated as a sum, we can easily determine the change
in e as x and y are incremented simply by adding and subtracting appropriately increasing values of x2delta (x squared delta) and y2delta (...).

The algorithm concludes when we find the right x and y that yields e==0.  In fact, I discover the x2delta and y2delta that yields e==0.  Then I convert those to the 
correct values of x and y.  Finally, the two factors are f1 = x-y and f2 = x+y.  In a final step, I multiply f1 x f2, and compare it to N as a verification check.

A third optimization in runtime is achieved by parallelizing the algorithm.  The approach I took here was to divide the search space into buckets, and partition
the buckets to twelve different threads to take advantage of the multithreading possible on my laptop, a 2018 MacBook Pro with 6 2.3GHz cores, each of which
supports two hyperthreads.

Let's see what we expect the runtime bounds to be.  Let the two factors be f1, f2 such that f1 <= f2.  Let's also assume that f2/f1 <= K, i.e. the two factors
are within a multiple of K of each other.  In practice, the challenge numbers have factors of the same decimal order of magnitude, therefore K = 10.
Now: 
  f2/f1 = (x+y)/(x-y) <= K
  x >= (K+1)y / (K-1)
  y <= (K-1)x/(K+1)
  
  N = x^^2 - y^^2 >= x^^2 - ((K-1)x/(K+1))^^2 = (1-((K-1)/(K+1))^^2)x^^2 = 4K/(K+1)^^2 * x^^2
  
  Let the solution value of x = r + d, where r is the initial value of x = ceiling(N^^(1/2))
  
  N >= 4K/(K+1)^^2 * (r+d)^^2
  r^^2 >= 4K/(K+1)^^2 * (r+d)^^2
  r >= 2K^^(1/2)/(K+1) * (r+d)
  (1-2K^^(1/2)/(K+1))*r >= (2K^^(1/2)/(K+1))*d
  (K-2K^^(1/2)+1)r >= 2K^^(1/2)d
  d <= (K^^(1/2)-1)^^2/2K^^(1/2)r
  
  Substituting 10 for K:
  d <= 0.74r
  
  Therefore, if the ratio of the two factors is bound by 10, then x will increment at most 0.74r times, where r ~= N^^(1/2).
  In the worst case, y = 9/11x = 9/11 * (1.74 N^^(1/2)) = 1.42 N^^(1/2).
  Since y increments by 1 when x increments, and by 2 otherwise, and since on each loop iteration y is incremented, we have a maximum of
  (0.74 + (1.42-0.74)/2) N^^(1/2) = 1.08 N^^(1/2) loop iterations.
  
  The overall complexity of the algorithm given a bounded ratio of the two factors is O(N^^(1/2)).
  
  Unfortunately, for a large value of N, e.g. the RSA 100 challenge number, which is the smallest RSA challenge number, and which has 100 decimal digits,
  the running time of this algorithm is much too high to be practical.  Other more mathematically powerful factoring solutions are known and implemented.
  
  
  
  
