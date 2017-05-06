go-timecode
===========

[![Build Status](https://travis-ci.org/echa/go-timecode.svg?branch=master)](https://travis-ci.org/echa/go-timecode)
[![GoDoc](https://godoc.org/github.com/echa/go-timecode/timecode?status.svg)](https://godoc.org/github.com/echa/go-timecode/timecode)


go-timecode is a [Go](http://golang.org/) library for SMPTE ST 12-1-2014 timecodes.

Features
--------
- correct drop-frame and non-drop-frame support
- all standard film, video and TV edit rates 23.976, 24, 25, 29.97, 30, 48, 50, 59.94, 60, 96, 100, 120
- arbitrary user-defined edit rates down to 1ns precision with a timecode runtime of ~9 years
- conversion between timecode, frame number and realtime
- timecode and frame calculations
- timecode & rate fit into a single 64bit integer for efficient binary storage
- parses and outputs SMPTE ST 12-1 timecode with DF flag
- different output methods to include and parse edit rate with timecode strings


Many timecode libraries treat timecode values as literal wall-clock time which they are not. Instead, timecodes are mere address labels for edit units in a sequence of video frames or audio samples. With a different edit rate, the same frame in a sequence has a different timecode address.

It's probably the reason why drop-frame timecodes are often misunderstood. I guess it's better to call them skip-timecodes because that's what's happening. At 29.97fps each edit unit has a duration of 33.3666ms, 30 such edit units last for 1.001s. Hence, a 29.97DF timecode of `00:00:01;00` actually means 1.001s instead of 1s. Now this makes it obvious that the time-part of the timecode runs a bit slower than realtime. To compensate for this speed issue, a drop-frame timecode *skips some address labels* now and then. To be precise, it skips frame labels `??:??:??;00` and `??:??:??;01` every minute but not every 10th minute. That's all.


Documentation
-------------

- [API Reference](http://godoc.org/github.com/echa/go-timecode/timecode)
- [FAQ](https://github.com/echa/go-timecode/wiki/FAQ)

Installation
------------

Install go-timecode using the "go get" command:

    go get github.com/echa/go-timecode/timecode

The Go distribution is go-timecode's only dependency.

Examples
--------

```
import "github.com/echa/go-timecode/timecode"

tc := timecode.Parse("00:00:59;29")
tc.SetRate(timecode.Rate30DF)
tc.Add(2*time.Minute)
fmt.Println("Frame number at TC", tc, "is", tc.Frame())

```


Contributing
------------

See [CONTRIBUTING.md](https://github.com/echa/go-timecode/blob/master/.github/CONTRIBUTING.md).


License
-------

go-timecode is available under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).

