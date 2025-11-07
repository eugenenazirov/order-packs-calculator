# Algorithm Notes

## Problem Statement

Given a number of items `N` and a set of pack sizes `S = {s1 ... sk}`, find the combination that fulfils `N` exactly while minimising the total number of packs. If no combination exists, surface that fact quickly.

Constraints:

- `1 ≤ k ≤ 10`
- Pack sizes are positive integers (duplicates allowed but deduplicated internally)
- `N` may exceed 500 000 items

## High-Level Approach

The calculator implements a classic coin-change dynamic programming solution:

1. **Normalisation** – remove duplicates, verify bounds, and sort sizes ascending. This costs `O(k log k)`.
2. **DP Table** – allocate `dp[i]` storing the minimal packs needed to achieve `i`, plus `choice[i]` capturing the last pack chosen.
3. **Transition** – for each pack size `s`, iterate `i` from `s` to `N`, updating `dp[i]` when `dp[i - s] + 1` is cheaper.
4. **Reconstruction** – walk backwards from `N` using `choice` to recover counts per pack size.

The algorithm short-circuits with `ErrCannotFulfill` when `choice[N] == -1`.

## Complexity

Let `n = items` and `k = |packSizes|`.

- **Time:** `O(n × k)` – each table cell is visited once per pack size.
- **Memory:** `O(n)` – two integer slices of length `n + 1`.

For the worst-case edge case (`n = 500 000`, `k = 3`), the DP table holds ~4 MB of ints, which comfortably fits in memory while still completing well under the 2 s budget on modern hardware.

## Edge Case `[23, 31, 53] → 500 000`

1. During transitions, the DP discovers that `53` dominates most states due to its size.
2. Reconstruction walks:
   - `500000 - 53*9429 = 263`
   - `263` cannot be covered by `53`, so `31*7 = 217` reducing remainder to `46`.
   - `23*2 = 46` finishes the path.
3. Final distribution: `{53: 9429, 31: 7, 23: 2}`, total packs `9438`.

## Handling Impossible Inputs

If `N` is less than the smallest pack size, or the pack sizes are not coprime w.r.t. `N`, the DP table will leave `choice[N] == -1`. In that case the API returns:

```json
{
  "error": "Cannot pack exactly",
  "details": "cannot pack items exactly with the provided pack sizes",
  "suggestion": "Consider adding a pack size that divides N or adjust the order quantity"
}
```

Operators can then add more pack sizes via the UI or API.

## Optimisation Opportunities

- **Memory tuning:** Switch to `[]uint32` to halve DP footprint if necessary.
- **Pruning:** Track the greatest common divisor of pack sizes to reject impossible inputs earlier.
- **Parallelism:** Not required at current scale; the loop is CPU-friendly and fits within SLA.

The current implementation already satisfies the SLA and readability goals, so these remain future enhancements.
