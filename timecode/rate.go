// Copyright (c) 2017 Alexander Eichhorn
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package timecode

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type Rate struct {
	// Id for standard edit rates. This value is set to 0 when a rate
	// is unknown and to R_MAX when the rate is user-defined.
	enum int
	// Nominal frame count per second by which the timecode using this rate will
	// assign timecode address labels.
	fps int
	// Numerator value of the effective edit rate at which the data stream
	// will advance in real-time (e.g. 25 for 25fps).
	rateNum int
	// Denominator value of the effective edit rate at which the data stream
	// will advance in real-time. For most edit rates this value will be 1,
	// drop-frame Television rates like 29.97, 59.94 and the special camera
	// capture rate 23.976 use 1001.
	rateDen int
	// Number of timecode address labels that will be dropped once per minute.
	dropFrames int
	// Effective number of actual frames per 10 minute time interval. This is
	// the same number as valid timecode address labels during that duration.
	framesPer10Min int
}

// Standard edit rates for non-drop-frame timecodes.
const (
	_       = iota // special: treat nanosecond component as frame number
	R_23976        // 24000,1001 (Note: this is NOT a drop-frame edit rate)
	R_24           // 24,1
	R_25           // 25,1
	R_30           // 30,1
	R_48           // 48,1
	R_50           // 50,1
	R_60           // 60,1
	R_96           // 96,1
	R_100          // 100,1
	R_120          // 120,1
	R_MAX   = 15   // special case: requires rateNum and rateDen to be set
)

// Standard edit rates for drop-frame timecodes.
const (
	df     = 16 + iota
	_      // undefined
	_      // undefined
	_      // undefined
	R_30DF // 30000,1001
	_      // undefined
	_      // undefined
	R_60DF // 60000,1001
)

// Common edit rate configurations you should use in your code when calling New()
var (
	InvalidRate    Rate = Rate{R_MAX, 0, 0, 0, 0, 0}
	OneFpsRate     Rate = Rate{0, 1, 1, 1, 0, 1 * 600}                             // == 1fps
	IdentityRate   Rate = Rate{0, 1000000000, 1000000000, 1, 0, 1000000000 * 600}  // == 1ns
	IdentityRateDF Rate = Rate{df, 1000000000, 1000000000, 1, 0, 1000000000 * 600} // == 1ns
	Rate23976      Rate = Rate{R_23976, 24, 24000, 1001, 0, 24 * 600}
	Rate24         Rate = Rate{R_24, 24, 24, 1, 0, 24 * 600}
	Rate25         Rate = Rate{R_25, 25, 25, 1, 0, 25 * 600}
	Rate30         Rate = Rate{R_30, 30, 30, 1, 0, 30 * 600}
	Rate30DF       Rate = Rate{R_30DF, 30, 30000, 1001, 2, 17982}
	Rate48         Rate = Rate{R_48, 48, 48, 1, 0, 48 * 600}
	Rate50         Rate = Rate{R_50, 50, 50, 1, 0, 50 * 600}
	Rate60         Rate = Rate{R_60, 60, 60, 1, 0, 60 * 600}
	Rate60DF       Rate = Rate{R_60DF, 60, 60000, 1001, 4, 35964}
	Rate96         Rate = Rate{R_96, 96, 96, 1, 0, 96 * 600}
	Rate100        Rate = Rate{R_100, 100, 100, 1, 0, 100 * 600}
	Rate120        Rate = Rate{R_120, 120, 120, 1, 0, 120 * 600}
)

var rates map[int]Rate = map[int]Rate{
	0:       IdentityRate,
	df:      IdentityRateDF,
	R_23976: Rate23976,
	R_24:    Rate24,
	R_25:    Rate25,
	R_30:    Rate30,
	R_30DF:  Rate30DF,
	R_48:    Rate48,
	R_50:    Rate50,
	R_60:    Rate60,
	R_60DF:  Rate60DF,
	R_96:    Rate96,
	R_100:   Rate100,
	R_120:   Rate120,
}

// NewRate creates a user-defined rate from rate numerator n and denominator d.
// If the rate is approximately close to a pre-defined standard rate, the
// standard rate's configuration including the appropriate enum id will be used.
func NewRate(n, d int) Rate {
	if n == 0 {
		n = 1
	}
	if d == 0 {
		d = 1
	}
	fps := float32(n) / float32(d)
	r := NewFloatRate(fps)
	if r.enum == R_MAX {
		return Rate{R_MAX, int(math.Ceil(float64(fps))), n, d, 0, int(fps * 600)}
	}
	return r
}

// NewFloatRate converts the float32 f to a rate. If the rate is approximately
// close to a pre-defined standard rate, the standard rate's configuration
// including the appropriate enum id will be used.
func NewFloatRate(f float32) Rate {
	switch {
	case 23.975 <= f && f < 23.997:
		return rates[R_23976]
	case f == 24:
		return rates[R_24]
	case f == 25:
		return rates[R_25]
	case 29.96 < f && f < 29.98:
		return rates[R_30DF]
	case f == 30:
		return rates[R_30]
	case f == 48:
		return rates[R_48]
	case f == 50:
		return rates[R_50]
	case 59.93 < f && f < 59.95:
		return rates[R_60DF]
	case f == 60:
		return rates[R_60]
	case f == 96:
		return rates[R_96]
	case f == 100:
		return rates[R_100]
	case f == 120:
		return rates[R_120]
	default:
		return Rate{R_MAX, int(f), int(f * 1000), 1000, 0, int(f) * 600}
	}
}

// ParseRate converts the string s to a rate. The string is treated as a
// rate enumeration index when its value is an integer, as floating point
// rate when s parses as float32 or as rational rate otherwise.
//
// If the pased float or rational rate is approximately close to a pre-defined
// standard rate, the standard rate's configuration including the appropriate
// enum id will be used.
func ParseRate(s string) (Rate, error) {
	// try parsing as index
	if i, err := strconv.Atoi(s); err == nil {
		switch {
		case i <= R_MAX:
			fallthrough
		case i == R_30DF || i == R_60DF:
			return rates[i], nil
		default:
			return NewFloatRate(float32(i)), nil
		}
	}

	// try parsing as float
	if f, err := strconv.ParseFloat(s, 32); err == nil {
		return NewFloatRate(float32(f)), nil
	}

	// try parsing as rational
	if fields := strings.Split(s, "/"); len(fields) == 2 {
		a, _ := strconv.Atoi(fields[0])
		b, err := strconv.Atoi(fields[1])
		if err == nil && b > 0 {
			return NewFloatRate(float32(a) / float32(b)), nil
		}
	}

	return InvalidRate, fmt.Errorf("timecode: parsing rate \"%s\": invalid syntax", s)
}

// IsZero indicates if the rate equals IdentityRate. This may be used to check if
// a timecode has no associated rate using Timecode.Rate().IsZero().
func (r Rate) IsZero() bool {
	return r.IsEqual(IdentityRate)
}

// IsValid indicates if a rate may be used in calculations. Rates with a denominator
// of zero would lead to division by zero panics, rates with a numerator of zero
// are undefined.
func (r Rate) IsValid() bool {
	return r.rateNum > 0 && r.rateDen > 0
}

// IsDrop indicates if the rate refers to a drop-frame timecode.
func (r Rate) IsDrop() bool {
	return r.enum&0x10 > 0
}

// IndexString returns the enumeration for a standard timecode as string.
func (r Rate) IndexString() string {
	return strconv.Itoa(r.enum)
}

// Fraction returns the rate's numerator and denominator.
func (r Rate) Fraction() (int, int) {
	return r.rateNum, r.rateDen
}

// RationalString returns the rate as rational string of form 'numerator/denominator'.
func (r Rate) RationalString() string {
	return strings.Join([]string{
		strconv.Itoa(r.rateNum),
		strconv.Itoa(r.rateDen),
	}, "/")
}

// Float returns the rate as floating point number.
func (r Rate) Float() float32 {
	switch r.rateDen {
	case 0:
		return 0
	case 1:
		return float32(r.rateNum)
	default:
		return float32(r.rateNum) / float32(r.rateDen)
	}
}

// FloatString returns the rate as floating point string with maximum precision
// of 3 digits.
func (r Rate) FloatString() string {
	switch r.rateDen {
	case 0:
		return "0.0"
	case 1:
		return strconv.FormatFloat(float64(r.rateNum), 'f', 1, 32)
	default:

		return strconv.FormatFloat(float64(r.rateNum)/float64(r.rateDen), 'f', 3, 32)
	}
}

func (r Rate) MarshalText() ([]byte, error) {
	return []byte(r.FloatString()), nil
}

func (r *Rate) UnmarshalText(data []byte) error {
	d := string(data)
	switch d {
	case "", "-", "--", "NaN", "unknown":
		*r = IdentityRate
		return nil
	default:
		if rr, err := ParseRate(d); err != nil {
			return err
		} else {
			*r = rr
			return nil
		}
	}
}

// FrameDuration returns the duration of a single frame at the edit rate.
func (r Rate) FrameDuration() time.Duration {
	if r.rateNum == 0 {
		return time.Nanosecond
	}
	return time.Duration(1000000000 * float64(r.rateDen) / float64(r.rateNum))
}

// Duration returns the duration of f frames at the edit rate.
func (r Rate) Duration(f int64) time.Duration {
	if r.rateNum == 0 {
		return 0
	}
	d := time.Duration(float64(f) * 1000000000 * float64(r.rateDen) / float64(r.rateNum))
	return r.Truncate(d, 2)
}

// Frames returns the number of frames matching duration d at the edit rate.
func (r Rate) Frames(d time.Duration) int64 {
	return int64(d / r.FrameDuration())
}

// Truncate clips duration d to the edit rate's interval length, while internally
// rounding to precision digits.
func (r Rate) Truncate(d time.Duration, precision int) time.Duration {
	i := int64(d)
	n := int64(r.FrameDuration())
	if x := i % n; x > n/2 {
		return time.Duration(i + n - x)
	} else if x > n/int64(precision) {
		return time.Duration(i - x)
	} else {
		return d
	}
}

func (r Rate) TruncateFloat(d float64, precision int) time.Duration {
	pow := math.Pow(10, float64(precision))
	rd := r.FrameDuration()
	val := pow * float64(d) / float64(rd)
	_, div := math.Modf(val)
	var round float64
	if d > 0 {
		if div >= 0.5 {
			round = math.Ceil(val)
		} else {
			round = math.Floor(val)
		}
	} else {
		if div >= 0.5 {
			round = math.Floor(val)
		} else {
			round = math.Ceil(val)
		}
	}
	return time.Duration(round/pow) * rd
}

// MinRate returns the rate with smaller frame duration.
func MinRate(a, b Rate) Rate {
	if a.FrameDuration() > b.FrameDuration() {
		return a
	}
	return b
}

// MaxRate returns the rate with larger frame duration.
func MaxRate(a, b Rate) Rate {
	if a.FrameDuration() < b.FrameDuration() {
		return a
	}
	return b
}

// IsSmaller returns true if rate b has a larger frame duration than rate r.
func (r Rate) IsSmaller(b Rate) bool {
	return r.FrameDuration() < b.FrameDuration()
}

// IsEqual returns true when rate r and b have equal frame duration.
func (r Rate) IsEqual(b Rate) bool {
	return r.FrameDuration() == b.FrameDuration()
}
