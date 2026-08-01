# piClock Chassis Inventory

Authoritative record of physical chassis, their burn-in status, and shipping
disposition.

> **A chassis may be shipped only if `Burn-in = PASSED` and
> `Disposition = IN INVENTORY`.**

Blank cells mean *not yet known*. Do not fill them with guesses — capture the
real value from the unit.

## Fleet inventory

| Fleet | Serial             | Model / Rev  | RAM | eth0 MAC            | EEPROM      | Dongle port | Burn-in     | Disposition | Shippable |
| ----- | ------------------ | ------------ | --- | ------------------- | ----------- | ----------- | ----------- | ----------- | --------- |
| 1     |                    |              |     |                     |             |             |             |             |           |
| 2     | `4239e42b6042be00` | Pi 5 Rev 1.1 | 1GB | `88:a2:9e:ca:b5:8d` | 26 May 2026 | `1-1.1`     | NOT STARTED | IN TEST     | NO        |
| 3     |                    |              |     |                     |             |             |             |             |           |
| 4     | `2de21e75d52b0660` | Pi 5 Rev 1.1 | 1GB | `88:a2:9e:8a:ff:93` | 26 May 2026 | `1-1.1`     | NOT STARTED | IN TEST     | NO        |
| 5     |                    |              |     |                     |             |             |             |             |           |
| ?     | `022b7797c799c277` | Pi 5 Rev 1.0 | 2GB | `2c:cf:67:9f:3d:6b` | 8 Dec 2025  | `1-1.3`     | NOT STARTED | IN TEST     | NO        |

Open questions on the table:

- **Fleet 2** is inferred, not confirmed — the serial was reporting at the time
  fleet 2 was said to be booted.
- **`022b7797c799c277`** has no known fleet number. It may be fleet 1, 3 or 5.
  It is the only 2 GB and only Rev 1.0 chassis seen so far, making it the fleet
  outlier.
- **Fleets 1, 3, 5** have never been inspected.

## Status vocabulary

| Field           | Values                                                                  |
| --------------- | ----------------------------------------------------------------------- |
| **Burn-in**     | `NOT STARTED` / `IN PROGRESS` / `PASSED (date, spec ver)` / `FAILED (date, reason)` |
| **Disposition** | `IN INVENTORY` / `SHIPPED (date, to)` / `IN TEST` / `RMA` / `QUARANTINE` |
| **Shippable**   | `YES` only when Burn-in = `PASSED` **and** Disposition = `IN INVENTORY`  |

## Identification rules

Cards are moved between chassis routinely. Therefore:

- **Serial is the only stable chassis identifier.** IP addresses and hostnames
  travel with the SD card, not the chassis — never use them to identify a unit.
- **Never record an SD card as belonging to a chassis.** The card is the
  *software* under test; the chassis is where *hardware performance* attaches.
- A test result is the tuple `(software, chassis serial, timestamp)`.
- Re-read the serial from the running unit at test time rather than trusting any
  stored mapping.

Capture chassis identity with:

```sh
grep -E '^(Serial|Revision|Model)' /proc/cpuinfo
tr -d '\0' < /proc/device-tree/model
awk '/MemTotal/{printf "%.1f GB\n", $2/1048576}' /proc/meminfo
cat /sys/class/net/eth0/address
```

## Per-chassis history

### `2de21e75d52b0660` — fleet 4

Pi 5 Model B Rev 1.1 (`a04171`), 1.0 GB (986 MB).
EEPROM 26 May 2026; config `BOOT_UART=1`, `BOOT_ORDER=0xf1`.

- **2026-07-31** — 5 h 54 m continuous LTC soak (362 samples) on
  `trixie-v1.3.15`: zero stalls, zero restarts, zero errors.
  *Informal, pre-spec — does not count as burn-in.*
  Archive: `ltc-soak-fleet4-BOARD-2de21e75d52b0660-1GB-Rev1.1-362samples-5h54m-clean.csv`

### `4239e42b6042be00` — fleet 2 (inferred)

Pi 5 Model B Rev 1.1 (`a04171`), 1.0 GB.
EEPROM 26 May 2026; config `BOOT_UART=1`, `BOOT_ORDER=0xf1`.

- **2026-07-31** — 7/7 clean 240 s production-condition reboot cycles on
  `trixie-v1.3.15`; display confirmed up every cycle.
  *Informal, pre-spec — does not count as burn-in.*
  Archive: `ltc-cycle-prod-cond-BOARD-4239e42b6042be00-1GB-Rev1.1-7of7-clean.csv`

### `022b7797c799c277` — fleet unknown

Pi 5 Model B Rev 1.0 (`b04170`), 2.0 GB.
EEPROM 8 Dec 2025 (reports update available); config `BOOT_UART=1`,
`POWER_OFF_ON_HALT=0`, `BOOT_ORDER=0xf461`.

- **2026-07-31** — 16 h 50 m continuous soak on the `v1.3.1` card: zero
  restarts, zero errors, USB dongle never re-enumerated. Plus 5/5 clean 30 s
  reboot cycles. *Informal, pre-spec — does not count as burn-in.*

## Cross-chassis observations

The dongle's hub port tracks board revision: both Rev 1.1 chassis present it at
`1-1.1`, the Rev 1.0 chassis at `1-1.3`. Wiring is identical across units, so
this most likely reflects a Rev 1.0 vs Rev 1.1 routing difference on the USB
carrier rather than a different physical connector being used. The hub is
Single-TT with ganged power on all chassis.

## Software builds under test

Listed here for reference only — builds are **not** tied to any chassis.

| Build            | Identity                                                            | Notes                                                                     |
| ---------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `v1.3.1`         | commit `16302b4`; `sdl-clock` sha256 `0f32cd15…`                    | SDL2 only (`veandco/go-sdl2`), Go 1.17. `alsa-ltc.service` **has** the `ExecStopPost` USB re-authorize self-heal. |
| `trixie-v1.3.15` | commit `1d5a174`; `sdl-clock` sha256 `5520cd7e…`, `alsa-ltc` `14506407…` | SDL2 **and** SDL3 (`Zyko0/go-sdl3`, bundled libs), Go 1.24. **No** `ExecStopPost`; adds `ExecStartPre`. 19 commits past `v1.3.7`. |

## Results with unknown chassis

The 2026-07-29/30 wedge testing (~55 % wedge rate, arms A–D and A2) has **no
chassis serial recorded** in any archived capture. This gap is why
`board_serial` is now a mandatory column on every result row.
