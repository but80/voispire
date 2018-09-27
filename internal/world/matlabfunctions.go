package world

const (
	kNFact = 9
)

type filterCoef struct {
	a [3]float64
	b [2]float64
}

var filterCoefficients = map[int]filterCoef{
	11: filterCoef{ // fs : 44100 (default)
		a: [3]float64{
			2.450743295230728,
			-2.06794904601978,
			0.59574774438332101,
		},
		b: [2]float64{
			0.0026822508007163792,
			0.0080467524021491377,
		},
	},
	12: filterCoef{ // fs : 48000
		a: [3]float64{
			2.4981398605924205,
			-2.1368928194784025,
			0.62187513816221485,
		},
		b: [2]float64{
			0.0021097275904709001,
			0.0063291827714127002,
		},
	},
	10: filterCoef{
		a: [3]float64{
			2.3936475118069387,
			-1.9873904075111861,
			0.5658879979027055,
		},
		b: [2]float64{
			0.0034818622251927556,
			0.010445586675578267,
		},
	},
	9: filterCoef{
		a: [3]float64{
			2.3236003491759578,
			-1.8921545617463598,
			0.53148928133729068,
		},
		b: [2]float64{
			0.0046331164041389372,
			0.013899349212416812,
		},
	},
	8: filterCoef{ // fs : 32000
		a: [3]float64{
			2.2357462340187593,
			-1.7780899984041358,
			0.49152555365968692,
		},
		b: [2]float64{
			0.0063522763407111993,
			0.019056829022133598,
		},
	},
	7: filterCoef{
		a: [3]float64{
			2.1225239019534703,
			-1.6395144861046302,
			0.44469707800587366,
		},
		b: [2]float64{
			0.0090366882681608418,
			0.027110064804482525,
		},
	},
	6: filterCoef{ // fs : 24000 and 22050
		a: [3]float64{
			1.9715352749512141,
			-1.4686795689225347,
			0.3893908434965701,
		},
		b: [2]float64{
			0.013469181309343825,
			0.040407543928031475,
		},
	},
	5: filterCoef{
		a: [3]float64{
			1.7610939654280557,
			-1.2554914843859768,
			0.3237186507788215,
		},
		b: [2]float64{
			0.021334858522387423,
			0.06400457556716227,
		},
	},
	4: filterCoef{ // fs : 16000
		a: [3]float64{
			1.4499664446880227,
			-0.98943497080950582,
			0.24578252340690215,
		},
		b: [2]float64{
			0.036710750339322612,
			0.11013225101796784,
		},
	},
	3: filterCoef{
		a: [3]float64{
			0.95039378983237421,
			-0.67429146741526791,
			0.15412211621346475,
		},
		b: [2]float64{
			0.071221945171178636,
			0.21366583551353591,
		},
	},
	2: filterCoef{ // fs : 8000
		a: [3]float64{
			0.041156734567757189,
			-0.42599112459189636,
			0.041037215479961225,
		},
		b: [2]float64{
			0.16797464681802227,
			0.50392394045406674,
		},
	},
	0: filterCoef{ // default
		a: [3]float64{
			0.0,
			0.0,
			0.0,
		},
		b: [2]float64{
			0.0,
			0.0,
		},
	},
}

// FilterForDecimate calculates the coefficients of low-pass filter and
// carries out the filtering. This function is only used for decimate().
func FilterForDecimate(x []float64, x_length, r int, y []float64) {
	c, ok := filterCoefficients[r]
	if !ok {
		c = filterCoefficients[0]
	}

	// Filtering on time domain.
	w := [3]float64{}
	var wt float64
	for i := 0; i < x_length; i++ {
		wt = x[i] + c.a[0]*w[0] + c.a[1]*w[1] + c.a[2]*w[2]
		y[i] = c.b[0]*wt + c.b[1]*w[0] + c.b[1]*w[1] + c.b[0]*w[2]
		w[2] = w[1]
		w[1] = w[0]
		w[0] = wt
	}
}

func decimate(x []float64, x_length, r int, y []float64) {
	tmp1 := make([]float64, x_length+kNFact*2)
	tmp2 := make([]float64, x_length+kNFact*2)

	for i := 0; i < kNFact; i++ {
		tmp1[i] = 2*x[0] - x[kNFact-i]
	}
	for i := kNFact; i < kNFact+x_length; i++ {
		tmp1[i] = x[i-kNFact]
	}
	for i := kNFact + x_length; i < 2*kNFact+x_length; i++ {
		tmp1[i] = 2*x[x_length-1] - x[x_length-2-(i-(kNFact+x_length))]
	}

	FilterForDecimate(tmp1, 2*kNFact+x_length, r, tmp2)
	for i := 0; i < 2*kNFact+x_length; i++ {
		tmp1[i] = tmp2[2*kNFact+x_length-i-1]
	}
	FilterForDecimate(tmp1, 2*kNFact+x_length, r, tmp2)
	for i := 0; i < 2*kNFact+x_length; i++ {
		tmp1[i] = tmp2[2*kNFact+x_length-i-1]
	}

	nout := x_length/r + 1
	nbeg := r - r*nout + x_length

	count := 0
	for i := nbeg; i < x_length+kNFact; i += r {
		y[count] = tmp1[i+kNFact-1]
		count++
	}
}
