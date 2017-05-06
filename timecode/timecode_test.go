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
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func s(v int64) time.Duration {
	return time.Duration(v) * time.Second
}

func ms(v int64) time.Duration {
	return time.Duration(v) * time.Millisecond
}

func us(v int64) time.Duration {
	return time.Duration(v) * time.Microsecond
}

func ns(v int64) time.Duration {
	return time.Duration(v) * time.Nanosecond
}

type TimecodeTestcase struct {
	Id       string
	RateNum  int
	RateDen  int
	Time     time.Duration
	Offset   time.Duration
	Second   int64
	Frame    int64
	AsString string
}

func (tc *TimecodeTestcase) Check(t *testing.T, code Timecode) {
	if !code.IsValid() {
		t.Errorf("[Case #%s] Failed generating valid timecode", tc.Id)
	}
	r := NewRate(tc.RateNum, tc.RateDen)
	ms := code.Millisecond()
	expectedMs := int64((tc.Time + tc.Offset) / time.Millisecond)
	if cr := code.Rate(); !r.IsEqual(cr) {
		t.Errorf("[Case #%s] Wrong rate: expected=%s got=%s", tc.Id, r.RationalString(), cr.RationalString())
	}
	if ms != expectedMs {
		t.Errorf("[Case #%s] Wrong millisecond: expected=%d got=%d", tc.Id, expectedMs, ms)
	}
	s := code.Second()
	if s != tc.Second {
		t.Errorf("[Case #%s] Wrong second: expected=%d got=%d", tc.Id, tc.Second, s)
	}
	f := code.Frame()
	if f != tc.Frame {
		t.Errorf("[Case #%s] Wrong frame: expected=%d got=%d", tc.Id, tc.Frame, f)
	}
	fr := code.FrameAtRate(r)
	if fr != tc.Frame {
		t.Errorf("[Case #%s] Wrong frame with rate: expected=%d got=%d", tc.Id, tc.Frame, fr)
	}
	str := code.String()
	if str != tc.AsString {
		t.Errorf("[Case #%s] Wrong string: expected=%s got=%s", tc.Id, tc.AsString, str)
	}
}

var (
	TimecodeCreateTestcases []TimecodeTestcase = []TimecodeTestcase{
		// 23.976 fps (note: this is non-drop frame, meaning the that there are
		//             actually 24 frames counted in time code, but the time code
		//             runs slower than wall-clock time)
		//
		TimecodeTestcase{"23_1", 24000, 1001, 0, 0, 0, 0, "00:00:00:00"},
		TimecodeTestcase{"23_2", 24000, 1001, Rate23976.Duration(1), 0, 0, 1, "00:00:00:01"},
		TimecodeTestcase{"23_3", 24000, 1001, Rate23976.Duration(6), 0, 0, 6, "00:00:00:06"},
		TimecodeTestcase{"23_4", 24000, 1001, Rate23976.Duration(24), 0, 1, 24, "00:00:01:00"},
		TimecodeTestcase{"23_5", 24000, 1001, Rate23976.Duration(60 * 24), 0, 60, 1440, "00:01:00:00"},
		TimecodeTestcase{"23_6", 24000, 1001, Rate23976.Duration(600 * 24), 0, 600, 14400, "00:10:00:00"},
		TimecodeTestcase{"23_7", 24000, 1001, Rate23976.Duration(14401), 0, 600, 14401, "00:10:00:01"},

		// 24fps
		TimecodeTestcase{"24_1", 24, 1, ms(0), 0, 0, 0, "00:00:00:00"},
		TimecodeTestcase{"24_2", 24, 1, Rate24.Duration(1), 0, 0, 1, "00:00:00:01"},
		TimecodeTestcase{"24_3", 24, 1, Rate24.Duration(6), 0, 0, 6, "00:00:00:06"},
		TimecodeTestcase{"24_4", 24, 1, Rate24.Duration(24), 0, 1, 24, "00:00:01:00"},
		TimecodeTestcase{"24_5", 24, 1, Rate24.Duration(60*24 + 1), 0, 60, 1441, "00:01:00:01"},
		TimecodeTestcase{"24_6", 24, 1, Rate24.Duration(600*24 + 1), 0, 600, 14401, "00:10:00:01"},

		// 25fps
		TimecodeTestcase{"25_1", 25, 1, ms(0), 0, 0, 0, "00:00:00:00"},
		TimecodeTestcase{"25_2", 25, 1, ms(40), 0, 0, 1, "00:00:00:01"},
		TimecodeTestcase{"25_3", 25, 1, ms(240), 0, 0, 6, "00:00:00:06"},
		TimecodeTestcase{"25_4", 25, 1, s(1), 0, 1, 25, "00:00:01:00"},
		TimecodeTestcase{"25_5", 25, 1, s(60) + ms(40), 0, 60, 1501, "00:01:00:01"},
		TimecodeTestcase{"25_6", 25, 1, s(600) + ms(40), 0, 600, 15001, "00:10:00:01"},

		// 29.97 fps (note: this is a drop-frame timecode with 29.97 actual frames per
		//            wall-clock second; to repair the resulting difference between
		//            timecode display and wall clock, the first 2 timecode values
		//            are skipped every minute, but not every tenth minute)
		//
		//            so 00:01:00:00 and 00:01:00:01 don't exist,
		//            but 00:10:00:00 and 00:10:00:01 do
		//
		TimecodeTestcase{"29_1", 30000, 1001, ms(0), 0, 0, 0, "00:00:00;00"},
		TimecodeTestcase{"29_2", 30000, 1001, Rate30DF.Duration(1), 0, 0, 1, "00:00:00;01"},
		TimecodeTestcase{"29_3", 30000, 1001, Rate30DF.Duration(6), 0, 0, 6, "00:00:00;06"},
		TimecodeTestcase{"29_4", 30000, 1001, Rate30DF.Duration(30), 0, 1, 30, "00:00:01;00"},
		TimecodeTestcase{"29_5", 30000, 1001, Rate30DF.Duration(1799), 0, 60, 1799, "00:00:59;29"},
		TimecodeTestcase{"29_6", 30000, 1001, Rate30DF.Duration(1800), 0, 60, 1800, "00:01:00;02"},
		TimecodeTestcase{"29_7", 30000, 1001, Rate30DF.Duration(17982), 0, 600, 17982, "00:10:00;00"},

		// 30 fps
		TimecodeTestcase{"30_1", 30, 1, ms(0), 0, 0, 0, "00:00:00:00"},
		TimecodeTestcase{"30_2", 30, 1, Rate30.Duration(1), 0, 0, 1, "00:00:00:01"},
		TimecodeTestcase{"30_3", 30, 1, Rate30.Duration(6), 0, 0, 6, "00:00:00:06"},
		TimecodeTestcase{"30_4", 30, 1, Rate30.Duration(30), 0, 1, 30, "00:00:01:00"},
		TimecodeTestcase{"30_5", 30, 1, Rate30.Duration(60*30 + 1), 0, 60, 1801, "00:01:00:01"},
		TimecodeTestcase{"30_6", 30, 1, Rate30.Duration(600*30 + 1), 0, 600, 18001, "00:10:00:01"},

		// 48 fps
		TimecodeTestcase{"48_1", 48, 1, ms(0), 0, 0, 0, "00:00:00:00"},
		TimecodeTestcase{"48_2", 48, 1, Rate48.Duration(1), 0, 0, 1, "00:00:00:01"},
		TimecodeTestcase{"48_3", 48, 1, Rate48.Duration(6), 0, 0, 6, "00:00:00:06"},
		TimecodeTestcase{"48_4", 48, 1, Rate48.Duration(48), 0, 1, 48, "00:00:01:00"},
		TimecodeTestcase{"48_5", 48, 1, Rate48.Duration(60*48 + 1), 0, 60, 2881, "00:01:00:01"},
		TimecodeTestcase{"48_6", 48, 1, Rate48.Duration(600*48 + 1), 0, 600, 28801, "00:10:00:01"},

		// 50 fps
		TimecodeTestcase{"50_1", 50, 1, ms(0), 0, 0, 0, "00:00:00:00"},
		TimecodeTestcase{"50_2", 50, 1, Rate50.Duration(1), 0, 0, 1, "00:00:00:01"},
		TimecodeTestcase{"50_3", 50, 1, Rate50.Duration(6), 0, 0, 6, "00:00:00:06"},
		TimecodeTestcase{"50_4", 50, 1, Rate50.Duration(50), 0, 1, 50, "00:00:01:00"},
		TimecodeTestcase{"50_5", 50, 1, Rate50.Duration(60*50 + 1), 0, 60, 3001, "00:01:00:01"},
		TimecodeTestcase{"50_6", 50, 1, Rate50.Duration(600*50 + 1), 0, 600, 30001, "00:10:00:01"},

		// 59.97 fps drop frame
		TimecodeTestcase{"59_1", 60000, 1001, ms(0), 0, 0, 0, "00:00:00;00"},
		TimecodeTestcase{"59_2", 60000, 1001, Rate60DF.Duration(1), 0, 0, 1, "00:00:00;01"},
		TimecodeTestcase{"59_3", 60000, 1001, Rate60DF.Duration(6), 0, 0, 6, "00:00:00;06"},
		TimecodeTestcase{"59_4", 60000, 1001, Rate60DF.Duration(60), 0, 1, 60, "00:00:01;00"},
		TimecodeTestcase{"59_5", 60000, 1001, Rate60DF.Duration(3599), 0, 60, 3599, "00:00:59;59"},
		TimecodeTestcase{"59_6", 60000, 1001, Rate60DF.Duration(3600), 0, 60, 3600, "00:01:00;04"},
		TimecodeTestcase{"59_7", 60000, 1001, Rate60DF.Duration(35964), 0, 600, 35964, "00:10:00;00"},

		// 60 fps
		TimecodeTestcase{"60_1", 60, 1, ms(0), 0, 0, 0, "00:00:00:00"},
		TimecodeTestcase{"60_2", 60, 1, Rate60.Duration(1), 0, 0, 1, "00:00:00:01"},
		TimecodeTestcase{"60_3", 60, 1, Rate60.Duration(6), 0, 0, 6, "00:00:00:06"},
		TimecodeTestcase{"60_4", 60, 1, Rate60.Duration(60), 0, 1, 60, "00:00:01:00"},
		TimecodeTestcase{"60_5", 60, 1, Rate60.Duration(60*60 + 1), 0, 60, 3601, "00:01:00:01"},
		TimecodeTestcase{"60_6", 60, 1, Rate60.Duration(600*60 + 1), 0, 600, 36001, "00:10:00:01"},

		// 100 fps
		TimecodeTestcase{"100_1", 100, 1, ms(0), 0, 0, 0, "00:00:00:00"},
		TimecodeTestcase{"100_2", 100, 1, Rate100.Duration(1), 0, 0, 1, "00:00:00:01"},
		TimecodeTestcase{"100_3", 100, 1, Rate100.Duration(6), 0, 0, 6, "00:00:00:06"},
		TimecodeTestcase{"100_4", 100, 1, Rate100.Duration(100), 0, 1, 100, "00:00:01:00"},
		TimecodeTestcase{"100_5", 100, 1, Rate100.Duration(60*100 + 1), 0, 60, 6001, "00:01:00:01"},
		TimecodeTestcase{"100_6", 100, 1, Rate100.Duration(600*100 + 1), 0, 600, 60001, "00:10:00:01"},

		// 120 fps
		TimecodeTestcase{"120_1", 120, 1, ms(0), 0, 0, 0, "00:00:00:00"},
		TimecodeTestcase{"120_2", 120, 1, Rate120.Duration(1), 0, 0, 1, "00:00:00:01"},
		TimecodeTestcase{"120_3", 120, 1, Rate120.Duration(6), 0, 0, 6, "00:00:00:06"},
		TimecodeTestcase{"120_4", 120, 1, Rate120.Duration(120), 0, 1, 120, "00:00:01:00"},
		TimecodeTestcase{"120_5", 120, 1, Rate120.Duration(60*120 + 1), 0, 60, 7201, "00:01:00:01"},
		TimecodeTestcase{"120_6", 120, 1, Rate120.Duration(600*120 + 1), 0, 600, 72001, "00:10:00:01"},
	}
)

func TestCreate(t *testing.T) {
	for _, v := range TimecodeCreateTestcases {
		tt := New(v.Time, NewRate(v.RateNum, v.RateDen))
		v.Check(t, tt)
	}
}

func TestParse(t *testing.T) {
	for _, v := range TimecodeCreateTestcases {
		tt, err := Parse(v.AsString)
		if err != nil {
			t.Errorf("[Case #%s] unexpected error: %v", v.Id, err)
		}
		tt.SetRate(NewRate(v.RateNum, v.RateDen))
		v.Check(t, tt)
	}
}

func TestParseWithFloatRate(t *testing.T) {
	for _, v := range TimecodeCreateTestcases {
		tt := New(v.Time, NewRate(v.RateNum, v.RateDen))
		s := tt.StringWithRate()
		expected := fmt.Sprintf("%s@%s", v.AsString, NewRate(v.RateNum, v.RateDen).FloatString())
		if s != expected {
			t.Errorf("[Case #%s] Wrong string with rate: expected=%s got=%s", v.Id, expected, s)
		}
		t2, err := Parse(s)
		if err != nil {
			t.Errorf("[Case #%s] unexpected error: %v", v.Id, err)
		}
		v.Check(t, t2)
	}
}

func TestParseWithRationalRate(t *testing.T) {
	for _, v := range TimecodeCreateTestcases {
		s := fmt.Sprintf("%s@%s", v.AsString, NewRate(v.RateNum, v.RateDen).RationalString())
		tt, err := Parse(s)
		if err != nil {
			t.Errorf("[Case #%s] unexpected error: %v", v.Id, err)
		}
		v.Check(t, tt)
	}
}

func TestParseWithIndexRate(t *testing.T) {
	for _, v := range TimecodeCreateTestcases {
		s := fmt.Sprintf("%s@%s", v.AsString, NewRate(v.RateNum, v.RateDen).IndexString())
		tt, err := Parse(s)
		if err != nil {
			t.Errorf("[Case #%s] unexpected error: %v", v.Id, err)
		}
		v.Check(t, tt)
	}
}

func TestPassthrough(t *testing.T) {
	for _, v := range TimecodeCreateTestcases {
		tt, err := Parse(v.AsString)
		if err != nil {
			t.Errorf("[Case #%s] unexpected error: %v", v.Id, err)
		}
		s := tt.String()
		if v.AsString != s {
			t.Errorf("[Case #%s] Wrong passthrough: %s/%s", v.Id, v.AsString, s)
		}
	}
}

type TimecodeMarshal struct {
	T Timecode `json:"timecode"`
}

func TestMarshal(t *testing.T) {
	for _, v := range TimecodeCreateTestcases {
		m := TimecodeMarshal{
			T: New(v.Time, NewRate(v.RateNum, v.RateDen)),
		}
		b, err := json.Marshal(m)
		if err != nil {
			t.Errorf("[Case #%s] Marshal failed: %s", v.Id, err)
		}
		c := TimecodeMarshal{}
		if err = json.Unmarshal(b, &c); err != nil {
			t.Errorf("[Case #%s] Unmarshal failed: %s", v.Id, err)
		}
		c.T.SetRate(m.T.Rate())
		v.Check(t, c.T)
	}
}

var (
	TimecodeOffsetTestcases []TimecodeTestcase = []TimecodeTestcase{
		TimecodeTestcase{"25_1", 25, 1, ms(40), ms(40), 0, 2, "00:00:00:02"},                                    // 40ms = 1 frame
		TimecodeTestcase{"25_2", 25, 1, ms(1000), ms(40), 1, 26, "00:00:01:01"},                                 // 25 ms < 1 frame
		TimecodeTestcase{"25_3", 25, 1, ms(240), ms(1000), 1, 31, "00:00:01:06"},                                // 1s
		TimecodeTestcase{"25_4", 24, 1, Rate24.Duration(78471), Rate24.Duration(9), 3270, 78480, "00:54:30:00"}, // 00:54:29:15 + 9f
	}
)

func TestOffset(t *testing.T) {
	for _, v := range TimecodeOffsetTestcases {
		r := NewRate(v.RateNum, v.RateDen)
		tt := New(v.Time, r).Add(v.Offset)
		v.Check(t, tt)
	}
}
