1. **Optimize Redis Key Generation (`pkg/api/ratelimit/adapters/redis/redis.go`)**
   - Replace `fmt.Sprintf` calls with direct string concatenation and `strconv.FormatInt` to avoid reflection and heap allocation in hot rate-limiting paths.
   - Update `fixedWindowAllow` to replace `fmt.Sprintf("rl:dist:fixed:%s", key)` with `"rl:dist:fixed:" + key`.
   - Update `tokenBucketAllow` to replace `fmt.Sprintf("rl:dist:tb:%s", key)` with `"rl:dist:tb:" + key`.
   - Update `slidingWindowAllow` to replace `fmt.Sprintf("rl:dist:slide:%s", key)` with `"rl:dist:slide:" + key`.
   - Update `slidingWindowAllow` to replace `fmt.Sprintf("%d:%d", now, time.Now().UnixNano()%1000000)` with `strconv.FormatInt(now, 10) + ":" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)`.
   - Remove the `"fmt"` import from `pkg/api/ratelimit/adapters/redis/redis.go` and add `"strconv"`.

2. **Run tests**
   - Run tests targeting `pkg/api/ratelimit/adapters/redis` (noting from memory that these tests are located in `./pkg/algorithms/ratelimit/...` and `./pkg/servicemesh/ratelimit/...`) using `go test -v ./pkg/algorithms/ratelimit/... ./pkg/servicemesh/ratelimit/...`.

3. **Complete pre-commit steps**
   - Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.

4. **Create PR**
   - Use the `request_code_review` tool to create a PR as Bolt, formatting the title as `⚡ Bolt: Optimize Redis cache key generation` and the description with sections `💡 What`, `🎯 Why`, `📊 Impact`, and `🔬 Measurement`.
