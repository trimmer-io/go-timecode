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
	"testing"
)

type RateTestcase struct {
	Id       int
	rateNumA int
	RateDenA int
	rateNumB int
	RateDenB int
	AisLower bool
}

var (
	RateMinMaxTestcases []RateTestcase = []RateTestcase{
		RateTestcase{1, 25, 1, 24, 1, false},
		RateTestcase{2, 25, 1, 1, 1, false},
		RateTestcase{3, 25, 1, 1000, 1000, false},
		RateTestcase{4, 25, 1, 30000, 1001, true},
		RateTestcase{5, 30000, 1001, 25, 1, false},
		RateTestcase{6, 60000, 1001, 30000, 1001, false},
		RateTestcase{7, 30000, 1001, 60000, 1001, true},
	}
)

func TestTimecodeRateMin(t *testing.T) {
	for _, v := range RateMinMaxTestcases {
		a := NewRate(v.rateNumA, v.RateDenA)
		b := NewRate(v.rateNumB, v.RateDenB)
		c := MinRate(a, b)
		if v.AisLower {
			if !a.IsEqual(c) {
				t.Errorf("[Case #%.2d] Failed min test %d/%d != %d/%d", v.Id, c.rateNum, c.rateDen, a.rateNum, a.rateDen)
			}
		} else {
			if !b.IsEqual(c) {
				t.Errorf("[Case #%.2d] Failed min test %d/%d != %d/%d", v.Id, c.rateNum, c.rateDen, b.rateNum, b.rateDen)
			}
		}
	}
}

func TestTimecodeRateMax(t *testing.T) {
	for _, v := range RateMinMaxTestcases {
		a := NewRate(v.rateNumA, v.RateDenA)
		b := NewRate(v.rateNumB, v.RateDenB)
		c := MaxRate(a, b)
		if v.AisLower {
			if !b.IsEqual(c) {
				t.Errorf("[Case #%.2d] Failed max test %s != %s", v.Id, c, b)
			}
		} else {
			if !a.IsEqual(c) {
				t.Errorf("[Case #%.2d] Failed max test %s != %s", v.Id, c, a)
			}
		}
	}
}
