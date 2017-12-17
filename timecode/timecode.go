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

// SMPTE ST 12-1-2014
// SMPTE ST 331-2011, value 81h
//
// see also
// http://andrewduncan.net/timecodes/
// http://www.bodenzord.com/archives/79
// https://documentation.apple.com/en/finalcutpro/usermanual/index.html#chapter=D%26section=6
//
// TODO
// - support SMPTE 24h bit

// Package timecode provides types and primitives to work with SMPTE ST 12-1
// timecodes at standard and user-defined edit rates. Currently only the DF
// flag is supported and timecodes cannot be negative.
//
// The package supports functions to convert between timecode, frame number
// and realtime durations as well as functions for timecode calculations.
// Drop-frame and non-drop-frame timecodes are correctly handled and all
// standard film, video and Television edit rates are supported. You may
// also use arbitrary user-defined edit rates down to 1ns precision with
// a timecode runtime of ~9 years.
//
// Timecode and edit rate are stored as a single 64bit integer for efficient
// timecode handling and comparisons.
package timecode // import "trimmer.io/go-timecode/timecode"

import (
	"database/sql/driver"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Timecode represents a duration at nanosecond precision similar to Golang's
// time.Duration. Unlike time.Duration Timecode cannot be negative.
//
// The 6 most significant bits are used to store an edit rate identifier required for
// offset calculations. The timecode's duration value occupies the 59 least significant
// bits which allows for expressing ~9 years of runtime.
type Timecode uint64

const (
	Invalid   Timecode = math.MaxUint64
	Zero      Timecode = 0
	Origin    string   = "00:00:00:00"
	rate_bits uint64   = 5  // = 16+16 frame rates (NDF+DF)
	time_bits uint64   = 59 // ~9 years at 1 ns granularity (needs 30bit)
	time_mask uint64   = (1<<time_bits - 1)
	Mask               = time_mask
)

// New creates a new timecode from a time.Duration and an edit rate. The duration
// is truncated to the edit rate's interval length before storage.
func New(d time.Duration, r Rate) Timecode {
	d = r.Truncate(d, 2)
	return Timecode(uint64(r.enum)<<time_bits | (uint64(d) & time_mask))
}

// IsValid indicates if a timecode is valid. Invalid timecodes have all bits set to 1.
func (t Timecode) IsValid() bool {
	return t != Invalid
}

// IsZero indicates of the duration part of the timecode is zero. Rate bits are
// not considered in this check.
func (t Timecode) IsZero() bool {
	return t.Uint64()&time_mask == 0
}

// SetFrame sets the timecode to a new frame number f and keeps the timecode's
// current rate.
func (t *Timecode) SetFrame(f int64) Timecode {
	r := t.Rate()
	*t = New(r.Duration(f), r)
	return *t
}

// SetRate sets a new edit rate r for the timecode and keeps the timecode's
// frame counter. Use this function to change the current edit rate or set
// set an initial edit rate after parsing a timecode with Parse() when the
// string did not contain a valid rate.
func (t *Timecode) SetRate(r Rate) Timecode {
	if t.Rate().enum == 0 || t.Rate().enum == df {
		s := int64(t.Duration() / time.Second)
		f := int64(t.Duration() % time.Second)
		frames := s*int64(r.fps) + f
		if r.IsDrop() {
			d := frames / int64(r.framesPer10Min)
			m := s / 60
			frames -= (m - d) * int64(r.dropFrames)
		}
		*t = New(r.Duration(frames), r)
		return *t
	}

	// keep current frame number and adjust time to new rate
	f := t.Frame()
	*t = New(0, r)
	return t.SetFrame(f)
}

func (t Timecode) Rate() Rate {
	rate, ok := rates[int(uint64(t)>>time_bits)]
	if !ok {
		rate = rates[0]
	}
	return rate
}

// Parse converts the string s to a timecode with optional rate. Without
// rate, the frame number is stored as raw nanosecond value. To reflect its
// actual duration you must call SetRate before calling any calculation
// functions. It's legal to parse and print timecodes without a rate.
//
// Well-formed timecodes must contain exactly four numeric segements of format
// 'hh:mm:ss:ff' where 'hh' denotes hours, 'mm' minutes, 'ss' seconds and 'ff'
// frames. Drop-frame timecodes use a semicolon ';' as the last separator
// between seconds and frame number.
//
// If s contains a '@' character, Parse treats the following substring as rate
// expression and uses ParseRate() to read it.
func Parse(s string) (Timecode, error) {

	if s == "" {
		s = Origin
	}

	isDF := strings.Contains(s, ";")
	hasRate := strings.Contains(s, "@")
	r := IdentityRate

	// treat DN and non-DF the same way (rate must be set after parsing)
	if isDF {
		s = strings.Replace(s, ";", ":", -1)
		r = IdentityRateDF
	}

	// strip and parse rate
	if hasRate {
		idx := strings.Index(s, "@")
		var err error
		r, err = ParseRate(s[idx+1:])
		if err != nil {
			return Invalid, err
		}
		s = s[:idx]

		// timecode is a frame counter, don't treat it as literal time!
		var frames int64
		for i, v := range strings.Split(s, ":") {
			t, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				// reject timecodes with invalid numbers
				return Invalid, fmt.Errorf("timecode: parsing timecode \"%s\": invalid syntax", s)
			}
			switch i {
			case 0:
				frames += int64(t) * 3600 * int64(r.fps)
			case 1:
				frames += int64(t) * 60 * int64(r.fps)
			case 2:
				frames += int64(t) * int64(r.fps)
			case 3:
				frames += int64(t)
			default:
				// reject timecodes longer than 4 segements
				return Invalid, fmt.Errorf("timecode: parsing timecode \"%s\": invalid syntax", s)
			}
		}

		// reverse the adjustment for drop frame timecodes
		if isDF {
			d := frames / int64(r.framesPer10Min)
			m := frames % int64(r.framesPer10Min)
			df := int64(r.dropFrames)
			frames = frames - 9*df*d - df*((m-df)/int64(r.framesPer10Min/10))
		}

		return New(r.Duration(frames), r), nil
	}

	// without rate we keep the frame number as nanosec part until a rate is set
	var d time.Duration
	for i, v := range strings.Split(s, ":") {
		t, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			// reject timecodes with invalid numbers
			return Invalid, fmt.Errorf("timecode: parsing timecode \"%s\": invalid syntax", s)
		}
		switch i {
		case 0:
			d += time.Duration(t) * time.Hour
		case 1:
			d += time.Duration(t) * time.Minute
		case 2:
			d += time.Duration(t) * time.Second
		case 3:
			d += time.Duration(t)
		default:
			// reject timecodes longer than 4 segements
			return Invalid, fmt.Errorf("timecode: parsing timecode \"%s\": invalid syntax", s)
		}
	}

	return New(d, r), nil
}

// FromSMPTE unpacks the SMPTE timecode from tc and also considers the
// drop-frame bit. User bits are ignored right now.
func FromSMPTE(tc uint32, bits uint32) Timecode {
	h := uint64((tc>>28&0x03)*10 + (tc >> 24 & 0x0F))
	m := uint64((tc>>20&0x07)*10 + (tc >> 16 & 0x0F))
	s := uint64((tc>>12&0x07)*10 + (tc >> 8 & 0x0F))
	f := uint64((tc>>4&0x03)*10 + (tc & 0x0F))
	d := h*uint64(time.Hour) + m*uint64(time.Minute) + s*uint64(time.Second) + f
	t := Timecode(d & time_mask)
	if tc&0x40 > 0 {
		t |= df << time_bits
	}
	return t
}

// FromSMPTEwithRate unpacks the SMPTE timecode from tc, considering the
// drop-frame bit and uses rate as initial timecode rate.
func FromSMPTEwithRate(tc, bits uint32, rate float32) Timecode {
	t := FromSMPTE(tc, bits)
	if rate != 0 {
		t.SetRate(NewFloatRate(rate))
	}
	return t
}

// SMPTE returns a packed SMPTE timecode and user bits from the current
// timecode value.
func (t Timecode) SMPTE() (uint32, uint32) {
	rate := t.Rate()
	fps := int64(rate.fps)
	frame := t.adjustedFrame(rate)
	ff := frame % fps
	ss := frame / fps % 60
	mm := frame / (fps * 60) % 60
	hh := frame / (fps * 3600)
	tc := (hh/10)<<28 + hh%10<<24 + mm/10<<20 + mm%10<<16 + ss/10<<12 + ss%10<<8 + ff/10<<4 + ff%10
	if rate.IsDrop() {
		tc |= 0x40
	}
	return uint32(tc), 0
}

// String returns a string representation of the timecode as `hh:mm:ss:ff`.
// If the timecode uses a drop-frame edit rate, the last separator in the
// string is a semicolon `;`.
func (t Timecode) String() string {
	rate := t.Rate()
	frame := t.adjustedFrame(rate)
	fps := int64(rate.fps)
	ff := frame % fps
	ss := frame / fps % 60
	mm := frame / (fps * 60) % 60
	hh := frame / (fps * 3600)
	sep := ':'
	if t.Rate().IsDrop() {
		sep = ';'
	}
	return fmt.Sprintf("%02d:%02d:%02d%c%02d", hh, mm, ss, sep, ff)
}

// StringWithRate returns the timecode as string appended with the current
// rate after a separating `@` character.
func (t Timecode) StringWithRate() string {
	if t.Rate().enum == IdentityRate.enum {
		return t.String()
	}
	return fmt.Sprintf("%s@%s", t.String(), t.Rate().FloatString())
}

// Uint64 returns the raw timecode value as unsigned 64bit integer.
func (t Timecode) Uint64() uint64 {
	return uint64(t)
}

// Duration returns the duration part of the timecode.
func (t Timecode) Duration() time.Duration {
	return time.Duration((uint64(t) & time_mask))
}

// Second returns a properly rounded number of seconds covered by the
// timecode.
func (t Timecode) Second() int64 {
	// adjust for small rounding errors from periodic fractions
	// as found with almost all frame rate durations
	//
	//   24fps      41.666666ms
	//   30fps DF   33.366666ms
	//   30fps      33.333333ms
	//   48fps      20.833333ms
	//   60fps DF   16.683333ms
	//   60fps      16.666666ms
	//  120fps      8.333333ms
	//
	// return int64(t.Duration() / time.Second)
	return int64(math.Floor(float64(t.Duration())/float64(time.Second) + 0.001))
}

// Second returns the number of milliseconds covered by the timecode.
func (t Timecode) Millisecond() int64 {
	return int64(t.Duration() / time.Millisecond)
}

// Frame returns the frame sequence counter value corresponding to the
// timecode's duration at the timecode's edit rate. Note that this value
// will be wrong when the edit rate is unknown or unset, as is the case
// after parsing a timecode from string without setting the rate.
func (t Timecode) Frame() int64 {
	rate, ok := rates[int(uint64(t)>>time_bits)]
	if !ok {
		rate = rates[0]
	}
	return t.FrameAtRate(rate)
}

// FrameAtRate returns the frame sequence counter value corresponding to the
// timecode's duration at edit rate r.
func (t Timecode) FrameAtRate(r Rate) int64 {
	// when rate id is 0 the frame number within the current second
	// is stored as nanosecond value
	if r.enum == 0 || r.enum == df {
		f := int64(r.fps) * t.Second()
		f += int64(t.Duration() % time.Second)
		return f
	}

	// all other cases use nanosecond as time base for duration
	return int64(t.Duration() / r.FrameDuration())
}

func (t Timecode) adjustedFrame(r Rate) int64 {
	f := t.FrameAtRate(r)
	if !r.IsDrop() {
		return f
	}

	// for 29.97DF skip timecodes 0 and 1 of the first second
	// of every minute, except when the number of minutes
	// is divisible by ten (same for 59.97DF except skip 4 timecodes)
	d := f / int64(r.framesPer10Min)
	m := f % int64(r.framesPer10Min)
	df := int64(r.dropFrames)
	return f + 9*df*d + df*((m-df)/int64(r.framesPer10Min/10))
}

// Sub returns the difference between timecodes t and t2 in nanoseconds as
// time.Duration.
func (t Timecode) Sub(t2 Timecode) time.Duration {
	return t.Duration() - t2.Duration()
}

// Add returns a new timecode with current rate and duration d added to the
// current duration. Any negative result will be clipped to zero.
func (t Timecode) Add(d time.Duration) Timecode {
	d = t.Duration() + d
	if d < 0 {
		return New(0, t.Rate())
	}
	return New(d, t.Rate())
}

// AddFrames returns a new timecode adjusted by f frames relative to the
// edit rate. If f is positive, the new timecode is larger than the
// current one, if negative it will be smaller. Any negative result after
// adding will be clipped to zero.
func (t Timecode) AddFrames(f int64) Timecode {
	if f > t.Frame() {
		return New(0, t.Rate())
	}
	return New(t.Duration()+t.Rate().Duration(f), t.Rate())
}

// MarshalText implements the encoding.TextMarshaler interface for
// converting a timecode value to string. This implementation preserves
// the rate
func (t Timecode) MarshalText() ([]byte, error) {
	if t.IsValid() {
		return []byte(t.StringWithRate()), nil
	} else {
		return []byte{}, nil
	}
}

// UnmarshalText implements the encoding.TextMarshaler interface for
// reading a timecode values.
func (t *Timecode) UnmarshalText(data []byte) error {
	x, err := Parse(string(data))
	if err != nil {
		return err
	}
	*t = x
	return nil
}

// Scan implements sql.Scanner interface for converting database values
// to timecode so you can use type timecode.Timecode directly with ORMs
// or the sql package.
func (t *Timecode) Scan(value interface{}) error {
	var x Timecode
	var err error
	switch v := value.(type) {
	case int64:
		x = Timecode(v)
	case string:
		x, err = Parse(v)
	case []byte:
		x, err = Parse(string(v))
	case nil:
		x = Zero
	}
	if err != nil {
		return err
	}
	*t = x
	return nil
}

// Value implements sql driver.Valuer interface for converting timecodes
// to a database driver compatible type.
func (t Timecode) Value() (driver.Value, error) {
	return int64(t), nil
}

// ConvertTimecode implements schema.Converter function defined by the
// Gorilla schema package. To use this converter you need to register it
// via
//
//   dec := schema.NewDecoder()
//   dec.RegisterConverter(timecode.Timecode(0), timecode.ConvertTimecode)
//
// This will eventually becomes unnecessary once https://github.com/gorilla/schema/issues/57
// is fixed.
func ConvertTimecode(value string) reflect.Value {
	if t, err := Parse(value); err != nil {
		return reflect.ValueOf(t)
	}
	return reflect.Value{}
}
