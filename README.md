# Rate-limiting experiment

Using an example rate of 100 requests per minute, a safe implementation would be to only allow 1
request every 0.6 seconds using a `time.Ticker` (make a request, wait 0.6 seconds). However, we
chose something a little more complex (but not too complex) for more flexibility.

# The chosen rate limiter

The https://pkg.go.dev/golang.org/x/time/rate#Limiter from `golang.org/x/time/rate` is a good fit,
and implements the Token Bucket algorithm: https://en.wikipedia.org/wiki/Token_bucket.

**How to use:**

1. Instantiate `limiter := rate.NewLimiter(rate, burst)`, and share between goroutines
2. Before a rate-limited operation (like an outgoing API call), call `limiter.Wait(ctx)`
    * Note: the `limiter` is thread-safe

# Technical details and reasoning for parameters used

A `time.Ticker`'s primary drawback, as described in https://pkg.go.dev/time@go1.22.2#NewTicker:

> The ticker will adjust the time interval or drop ticks to make up for slow receivers.

It can end up being inefficient if too many ticks are dropped. In practice, it helps to allow a
burst of requests to prevent these issues:
* At the start, get all threads to send API requests since they are ready
* At points where we've taken longer to process API responses, but become ready again to catch up
  and make API requests

To allow this burst to happen while staying strictly under a target rate, the rate must be reduced
based on the burst size assuming maximum usage rate.
* Burst at start of period, then subsequent requests as soon as possible.

For example, permitting a burst of 5 while staying under 100 requests per minute involves a token
bucket refill rate of 95 requests per minute. A burst followed by maximum usage rate represents
5 + 95 = 100 requests in one minute, even if over a long period of time, the rate gets closer to the
token bucket refill rate of 95 requests per second. The logs give a more intuitive view on this
behaviour.

From a log timing perspective, 95 requests per second (request every 0.631578947368421 seconds) is a
bit awkward. However, we can see the initial burst of projects being processed at nearly 0s, then
1 project at 632ms, 1.263s, 1.895s, and so on until the end.

Log output from running the experiment on 100 projectIds
* Prefix is "time since start"
* The final rate of ~100 requests per minute highlights maximum usage rate
* Burst of 5 + 95 requests at 95/min = 100 requests processed in 1m

```
26.458µs Consumer 1 launched
114.791µs Consumer 3 launched
62.125µs Producer launched
31.625µs Consumer 2 launched
159.666µs ProjectIDs processed: 1
163.416µs ProjectIDs processed: 3
164.666µs ProjectIDs processed: 4
166.083µs ProjectIDs processed: 5
161.458µs ProjectIDs processed: 2
77.833µs Consumer 4 launched
99.25µs Consumer 5 launched
632.7995ms ProjectIDs processed: 6
1.264422041s ProjectIDs processed: 7
1.89885825s ProjectIDs processed: 8
2.527574791s ProjectIDs processed: 9
3.158758958s ProjectIDs processed: 10
6.3184045s ProjectIDs processed: 15
9.474893625s ProjectIDs processed: 20
12.635853875s ProjectIDs processed: 25
15.789964125s ProjectIDs processed: 30
18.94857725s ProjectIDs processed: 35
22.106118041s ProjectIDs processed: 40
25.264395333s ProjectIDs processed: 45
28.422054458s ProjectIDs processed: 50
31.579962541s ProjectIDs processed: 55
31.580040083s Producer done
34.73876425s ProjectIDs processed: 60
37.8954855s ProjectIDs processed: 65
41.05381275s ProjectIDs processed: 70
44.211729208s ProjectIDs processed: 75
47.36961725s ProjectIDs processed: 80
48.632804458s Done received (1)
50.52750625s ProjectIDs processed: 85
53.685145833s ProjectIDs processed: 90
56.843323083s ProjectIDs processed: 95
56.843375083s Done received (2)
58.738085833s Done received (3)
59.369656166s Done received (4)
1m0.001230125s ProjectIDs processed: 100
1m0.001280708s Done received (5)
1m0.001283041s Done. Actual rate: 100 / 1.000021 minutes = 99.997862 per minute
```

Log output from running the experiment on 200 projectIds
* Prefix is "time since start"
* The final rate of ~97.5 requests per minute highlights maximum usage rate, with the burst of 5 
requests being part of minute 1 and not minute 2
* Burst of 5 + 190 requests (at 95/min) = 195 requests processed in 2m
    * then last 5 requests (at 95/min) processed in 3.158s

```
36.375µs Producer launched
103.875µs Consumer 1 launched
119.292µs ProjectIDs processed: 1
120.834µs ProjectIDs processed: 2
121.959µs ProjectIDs processed: 3
122.834µs ProjectIDs processed: 4
123.792µs ProjectIDs processed: 5
154.75µs Consumer 2 launched
158.834µs Consumer 3 launched
163.875µs Consumer 4 launched
166.75µs Consumer 5 launched
632.669125ms ProjectIDs processed: 6
1.263647709s ProjectIDs processed: 7
1.90076125s ProjectIDs processed: 8
2.531289709s ProjectIDs processed: 9
3.159033875s ProjectIDs processed: 10
6.316824834s ProjectIDs processed: 15
9.474482417s ProjectIDs processed: 20
12.632858625s ProjectIDs processed: 25
15.789827084s ProjectIDs processed: 30
18.94830075s ProjectIDs processed: 35
22.106435459s ProjectIDs processed: 40
25.266130959s ProjectIDs processed: 45
28.422191959s ProjectIDs processed: 50
31.57983025s ProjectIDs processed: 55
34.737872834s ProjectIDs processed: 60
37.895953667s ProjectIDs processed: 65
41.053843375s ProjectIDs processed: 70
44.211736792s ProjectIDs processed: 75
47.369626875s ProjectIDs processed: 80
50.527777667s ProjectIDs processed: 85
53.685945625s ProjectIDs processed: 90
56.842891459s ProjectIDs processed: 95
1m0.001196s ProjectIDs processed: 100
1m3.160586667s ProjectIDs processed: 105
1m6.317009292s ProjectIDs processed: 110
1m9.474932667s ProjectIDs processed: 115
1m12.632593667s ProjectIDs processed: 120
1m15.790322542s ProjectIDs processed: 125
1m18.948576042s ProjectIDs processed: 130
1m22.106491459s ProjectIDs processed: 135
1m25.264375334s ProjectIDs processed: 140
1m28.422697334s ProjectIDs processed: 145
1m31.579831292s ProjectIDs processed: 150
1m34.73804075s ProjectIDs processed: 155
1m34.738185917s Producer done
1m37.90002575s ProjectIDs processed: 160
1m41.053776959s ProjectIDs processed: 165
1m44.211724667s ProjectIDs processed: 170
1m47.368758417s ProjectIDs processed: 175
1m48.000863917s Done received (1)
1m50.526823792s ProjectIDs processed: 180
1m53.685733375s ProjectIDs processed: 185
1m56.843301459s ProjectIDs processed: 190
2m0.001757667s ProjectIDs processed: 195
2m1.264375959s Done received (2)
2m1.895306917s Done received (3)
2m2.529194375s Done received (4)
2m3.159651167s ProjectIDs processed: 200
2m3.15971575s Done received (5)
2m3.159720917s Done. Actual rate: 200 / 2.052662 minutes = 97.434453 per minute
```