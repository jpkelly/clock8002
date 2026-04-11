/*
 * alsa-ltc — ALSA audio LTC timecode decoder with OSC output
 *
 * Captures audio from an ALSA device, decodes LTC (Linear Timecode)
 * using libltc, and sends the timecode as OSC messages to a specified
 * destination. Designed for use with clock-8001/8002.
 *
 * Usage: alsa-ltc [-v] <alsa-device> <OSC destination ip> <OSC port> [sample-rate] [fps]
 *        Use "-" for device to auto-detect on Raspberry Pi (bcm2835)
 *        -v enables verbose mode (activity dots, ALSA details, signal levels)
 *        sample-rate defaults to 44100 if not specified
 *        fps defaults to 25 if not specified
 *
 * Build: gcc -O2 -o alsa-ltc alsa-ltc.c -lasound -lltc
 * Dependencies: libasound2-dev, libltc-dev
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <unistd.h>
#include <time.h>
#include <arpa/inet.h>
#include <sys/socket.h>
#include <alsa/asoundlib.h>
#include <ltc.h>

#define DEFAULT_SAMPLE_RATE 44100
#define DEFAULT_FPS 25
#define CHANNELS    1
#define BUF_SIZE    1024
#define HEARTBEAT_INTERVAL_S 30
#define LTC_GAP_THRESHOLD_MS 1000

#ifndef ALSA_LTC_GIT_TAG
#define ALSA_LTC_GIT_TAG "dev"
#endif

#ifndef ALSA_LTC_GIT_COMMIT
#define ALSA_LTC_GIT_COMMIT "unknown"
#endif

#ifndef ALSA_LTC_BUILD_DATE
#define ALSA_LTC_BUILD_DATE "unknown"
#endif

static volatile int running = 1;

static void print_version(FILE *out, const char *prog) {
    fprintf(out,
            "%s version %s commit %s built %s\n",
            prog,
            ALSA_LTC_GIT_TAG,
            ALSA_LTC_GIT_COMMIT,
            ALSA_LTC_BUILD_DATE);
}

static void sighandler(int sig) {
    (void)sig;
    running = 0;
}

/* Minimal OSC message builder for a string argument.
 * OSC format: address (null-padded to 4-byte boundary),
 *             type tag string ",s\0\0",
 *             string argument (null-padded to 4-byte boundary).
 * Returns total message length, or -1 on error.
 */
static int osc_build_message(char *buf, int bufsize,
                             const char *address, const char *arg) {
    int pos = 0;

    /* Write address string, pad to 4-byte boundary */
    int alen = strlen(address) + 1;
    int apad = (4 - (alen % 4)) % 4;
    if (pos + alen + apad > bufsize) return -1;
    memcpy(buf + pos, address, alen);
    pos += alen;
    memset(buf + pos, 0, apad);
    pos += apad;

    /* Type tag string ",s" */
    const char *typetag = ",s";
    int tlen = strlen(typetag) + 1;
    int tpad = (4 - (tlen % 4)) % 4;
    if (pos + tlen + tpad > bufsize) return -1;
    memcpy(buf + pos, typetag, tlen);
    pos += tlen;
    memset(buf + pos, 0, tpad);
    pos += tpad;

    /* String argument, pad to 4-byte boundary */
    int slen = strlen(arg) + 1;
    int spad = (4 - (slen % 4)) % 4;
    if (pos + slen + spad > bufsize) return -1;
    memcpy(buf + pos, arg, slen);
    pos += slen;
    memset(buf + pos, 0, spad);
    pos += spad;

    return pos;
}

/* Auto-detect a suitable ALSA capture device.
 * Looks for a hardware capture device (hw: prefix), preferring USB audio
 * or bcm2835. Skips "null" and output-only devices.
 * Returns a malloc'd string with the device name, or NULL.
 */
static char *detect_device(void) {
    void **hints = NULL;
    char *result = NULL;

    fprintf(stdout, "Detecting capture devices:\n");

    if (snd_device_name_hint(-1, "pcm", &hints) < 0)
        return NULL;

    for (int i = 0; hints[i] != NULL; i++) {
        char *name = snd_device_name_get_hint(hints[i], "NAME");
        char *desc = snd_device_name_get_hint(hints[i], "DESC");
        char *ioid = snd_device_name_get_hint(hints[i], "IOID");

        if (name == NULL)
            goto next;

        /* Skip null devices */
        if (strstr(name, "null") != NULL)
            goto next;

        /* Skip output-only devices */
        if (ioid != NULL && strstr(ioid, "Output") != NULL)
            goto next;

        fprintf(stdout, "Name: %s Desc: %s IO: %s\n",
                name ? name : "(null)",
                desc ? desc : "(null)",
                ioid ? ioid : "(null)");

        /* Match hw:CARD= devices — use plughw: for software conversion layer,
         * which is more resilient to USB device reset timing issues. */
        if (result == NULL && strncmp(name, "hw:CARD=", 8) == 0) {
            char plughw_name[256];
            snprintf(plughw_name, sizeof(plughw_name), "plug%s", name);
            fprintf(stdout, "Matched card: %s (using %s)\n", name, plughw_name);
            result = strdup(plughw_name);
        }

next:
        if (name) free(name);
        if (desc) free(desc);
        if (ioid) free(ioid);
    }

    snd_device_name_free_hint(hints);
    return result;
}

int main(int argc, char *argv[]) {
    snd_pcm_t *capture = NULL;
    LTCDecoder *decoder = NULL;
    short *audiobuf = NULL;
    int sock = -1;
    int rc;
    char *device = NULL;
    int device_allocated = 0;
    int exit_code = 1;  /* assume error; set to 0 only on clean shutdown */
    int verbose = 0;
    unsigned int sample_rate = DEFAULT_SAMPLE_RATE;
    unsigned int fps = DEFAULT_FPS;

    /* Parse optional -v flag */
    int argoff = 1;
    if (argc >= 2 && strcmp(argv[1], "--version") == 0) {
        print_version(stdout, argv[0]);
        return 0;
    }
    if (argc >= 2 && strcmp(argv[1], "-v") == 0) {
        verbose = 1;
        argoff = 2;
    }

    if (argc - argoff < 3 || argc - argoff > 5) {
        fprintf(stderr, "Usage:  %s [-v] <alsa-device> <OSC destination ip> <OSC port> [sample-rate] [fps]\n", argv[0]);
        fprintf(stderr, "  -v            Verbose mode: activity dots, ALSA details, signal levels\n");
        fprintf(stderr, "  sample-rate   Audio sample rate in Hz (default: %d)\n", DEFAULT_SAMPLE_RATE);
        fprintf(stderr, "  fps           LTC frame rate: 24, 25, or 30 (default: %d)\n", DEFAULT_FPS);
        fprintf(stderr, "  Use - for device for automatic detection on raspberry pi\n");
        fprintf(stderr, "  Use --version to display build information\n");
        return 1;
    }

    const char *osc_ip = argv[argoff + 1];
    int osc_port = atoi(argv[argoff + 2]);
    if (argc - argoff >= 4)
        sample_rate = (unsigned int)atoi(argv[argoff + 3]);
    if (argc - argoff >= 5)
        fps = (unsigned int)atoi(argv[argoff + 4]);

    signal(SIGINT, sighandler);
    signal(SIGTERM, sighandler);

    print_version(stdout, argv[0]);

    /* Device selection — with retry loop for USB audio boot timing.
     *
     * On Pi 5 with VL805 xHCI, the CM108 USB audio device becomes
     * unresponsive to SET_INTERFACE ~20s after enumeration if no active
     * USB traffic exists.  We must detect the device and start streaming
     * within that window.  Poll every 1s for up to 15s so we catch
     * the device the moment snd-usb-audio finishes probing.
     */
#define DETECT_TIMEOUT_S  15
#define DETECT_INTERVAL_S  1

    int use_autodetect = (strcmp(argv[argoff], "-") == 0);

    for (int attempt = 0; ; attempt++) {
        if (!running) return 0;

        if (use_autodetect) {
            device = detect_device();
            if (device == NULL) {
                if (attempt * DETECT_INTERVAL_S < DETECT_TIMEOUT_S) {
                    if (attempt == 0)
                        fprintf(stdout, "No capture device found, waiting for USB audio...\n");
                    sleep(DETECT_INTERVAL_S);
                    continue;
                }
                fprintf(stderr, "No suitable capture device found after %ds\n", DETECT_TIMEOUT_S);
                return 1;
            }
            device_allocated = 1;
        } else {
            device = argv[argoff];
        }

        /* Open ALSA capture */
        rc = snd_pcm_open(&capture, device, SND_PCM_STREAM_CAPTURE, 0);
        if (rc < 0) {
            fprintf(stderr, "cannot open audio device %s (%s)\n", device, snd_strerror(rc));
            if (device_allocated) { free(device); device = NULL; device_allocated = 0; }
            if (use_autodetect && attempt * DETECT_INTERVAL_S < DETECT_TIMEOUT_S) {
                sleep(DETECT_INTERVAL_S);
                continue;
            }
            return 1;
        }
        fprintf(stdout, "audio interface opened\n");

        /* Print ALSA card info for diagnostics (always) */
        {
            snd_pcm_info_t *pcm_info;
            snd_pcm_info_alloca(&pcm_info);
            if (snd_pcm_info(capture, pcm_info) == 0) {
                fprintf(stdout, "Card: %s\n", snd_pcm_info_get_name(pcm_info));
                fprintf(stdout, "Device: %s (subdevice %d)\n",
                        snd_pcm_info_get_subdevice_name(pcm_info),
                        snd_pcm_info_get_subdevice(pcm_info));
            }
        }

        break;  /* successfully opened */
    }

    /* Configure hardware parameters */
    snd_pcm_hw_params_t *hw_params;

    rc = snd_pcm_hw_params_malloc(&hw_params);
    if (rc < 0) {
        fprintf(stderr, "cannot allocate hardware parameter structure (%s)\n", snd_strerror(rc));
        goto cleanup;
    }
    fprintf(stdout, "hw_params allocated\n");

    rc = snd_pcm_hw_params_any(capture, hw_params);
    if (rc < 0) {
        fprintf(stderr, "cannot initialize hardware parameter structure (%s)\n", snd_strerror(rc));
        snd_pcm_hw_params_free(hw_params);
        goto cleanup;
    }
    fprintf(stdout, "hw_params initialized\n");

    rc = snd_pcm_hw_params_set_access(capture, hw_params, SND_PCM_ACCESS_RW_INTERLEAVED);
    if (rc < 0) {
        fprintf(stderr, "cannot set access type (%s)\n", snd_strerror(rc));
        snd_pcm_hw_params_free(hw_params);
        goto cleanup;
    }
    fprintf(stdout, "hw_params access setted\n");

    rc = snd_pcm_hw_params_set_format(capture, hw_params, SND_PCM_FORMAT_S16_LE);
    if (rc < 0) {
        fprintf(stderr, "cannot set sample format (%s)\n", snd_strerror(rc));
        snd_pcm_hw_params_free(hw_params);
        goto cleanup;
    }
    fprintf(stdout, "hw_params format setted\n");

    unsigned int rate = sample_rate;
    rc = snd_pcm_hw_params_set_rate_near(capture, hw_params, &rate, 0);
    if (rc < 0) {
        fprintf(stderr, "cannot set sample rate (%s)\n", snd_strerror(rc));
        snd_pcm_hw_params_free(hw_params);
        goto cleanup;
    }
    if (rate != sample_rate)
        fprintf(stdout, "hw_params rate: requested %u, got %u\n", sample_rate, rate);
    else
        fprintf(stdout, "hw_params rate setted\n");

    rc = snd_pcm_hw_params_set_channels(capture, hw_params, CHANNELS);
    if (rc < 0) {
        fprintf(stderr, "cannot set channel count (%s)\n", snd_strerror(rc));
        snd_pcm_hw_params_free(hw_params);
        goto cleanup;
    }
    fprintf(stdout, "hw_params channels setted\n");

    rc = snd_pcm_hw_params(capture, hw_params);
    if (rc < 0) {
        fprintf(stderr, "cannot set parameters (%s)\n", snd_strerror(rc));
        snd_pcm_hw_params_free(hw_params);
        goto cleanup;
    }
    fprintf(stdout, "hw_params setted\n");

    if (verbose) {
        snd_pcm_uframes_t buf_sz, per_sz;
        unsigned int periods;
        snd_pcm_hw_params_get_buffer_size(hw_params, &buf_sz);
        snd_pcm_hw_params_get_period_size(hw_params, &per_sz, NULL);
        snd_pcm_hw_params_get_periods(hw_params, &periods, NULL);
        fprintf(stdout, "  buffer_size=%lu period_size=%lu periods=%u\n",
                (unsigned long)buf_sz, (unsigned long)per_sz, periods);
    }

    snd_pcm_hw_params_free(hw_params);
    fprintf(stdout, "hw_params freed\n");

    rc = snd_pcm_prepare(capture);
    if (rc < 0) {
        fprintf(stderr, "cannot prepare audio interface for use (%s)\n", snd_strerror(rc));
        goto cleanup;
    }
    fprintf(stdout, "audio interface prepared\n");

    /* Allocate audio buffer */
    audiobuf = malloc(BUF_SIZE * CHANNELS * sizeof(short));
    if (audiobuf == NULL) {
        fprintf(stderr, "cannot allocate audio buffer\n");
        goto cleanup;
    }
    fprintf(stdout, "buffer allocated\n");

    /* Initialize LTC decoder */
    decoder = ltc_decoder_create(sample_rate * fps / BUF_SIZE, 32);
    if (decoder == NULL) {
        fprintf(stderr, "cannot create LTC decoder\n");
        goto cleanup;
    }
    fprintf(stdout, "LTC decoder initialized: sample rate: %d, fps: %d / %d\n",
            rate, fps, BUF_SIZE);

    /* Create UDP socket for OSC output */
    sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        fprintf(stderr, "socket() failed\n");
        goto cleanup;
    }

    /* Enable broadcast if destination is broadcast address */
    int broadcast = 1;
    if (setsockopt(sock, SOL_SOCKET, SO_BROADCAST, &broadcast, sizeof(broadcast)) < 0) {
        fprintf(stderr, "setsockopt() failed\n");
        goto cleanup;
    }

    struct sockaddr_in dest;
    memset(&dest, 0, sizeof(dest));
    dest.sin_family = AF_INET;
    dest.sin_port = htons(osc_port);
    dest.sin_addr.s_addr = inet_addr(osc_ip);

    /* Main capture loop */
    /* If we reach the main loop, setup succeeded — a clean SIGTERM exit is success */
    exit_code = 0;

    char prev_tc[16] = "";
    ltc_off_t total_samples = 0;
    int consecutive_errors = 0;
    time_t last_heartbeat = time(NULL);
    unsigned long heartbeat_frames = 0;
    unsigned long heartbeat_ltc_frames = 0;
    short peak_level = 0;
    struct timespec last_ltc_ts = {0, 0};
    int ltc_active = 0;  /* set after first decode */

    while (running) {
        snd_pcm_sframes_t frames = snd_pcm_readi(capture, audiobuf, BUF_SIZE);
        if (frames < 0) {
            consecutive_errors++;

            /* Check PCM state — DISCONNECTED means the USB device is gone */
            snd_pcm_state_t pcm_state = snd_pcm_state(capture);
            if (pcm_state == SND_PCM_STATE_DISCONNECTED || frames == -ENODEV) {
                fprintf(stderr, "USB audio device disconnected, exiting for systemd restart\n");
                exit_code = 1;
                goto cleanup;
            }

            fprintf(stderr, "read from audio interface failed (%s) [%d] pcm_state=%s\n",
                    snd_strerror(frames), consecutive_errors,
                    snd_pcm_state_name(pcm_state));
            /* USB devices may need a moment to settle — sleep before prepare */
            sleep(1);
            if (snd_pcm_prepare(capture) < 0) {
                if (consecutive_errors >= 10) {
                    /* Truly unrecoverable — exit so systemd can restart */
                    fprintf(stderr, "audio device unrecoverable after %d errors, exiting\n",
                            consecutive_errors);
                    exit_code = 1;
                    goto cleanup;
                }
                /* prepare failed but haven't hit limit yet — keep trying */
            }
            continue;
        }
        consecutive_errors = 0;
        heartbeat_frames += frames;

        /* Track peak audio level for verbose heartbeat */
        if (verbose) {
            for (snd_pcm_sframes_t i = 0; i < frames; i++) {
                short s = audiobuf[i] < 0 ? -audiobuf[i] : audiobuf[i];
                if (s > peak_level) peak_level = s;
            }
        }

        /* Periodic heartbeat (always on; verbose adds signal level) */
        {
            time_t now = time(NULL);
            if (now - last_heartbeat >= HEARTBEAT_INTERVAL_S) {
                fprintf(stdout, "\n[heartbeat] %lu frames, %lu ltc decoded, %d errors, device %s",
                        heartbeat_frames, heartbeat_ltc_frames, consecutive_errors, device);
                if (verbose)
                    fprintf(stdout, ", peak=%d", peak_level);
                fprintf(stdout, "\n");
                fflush(stdout);
                peak_level = 0;
                last_heartbeat = now;
            }
        }

        ltc_decoder_write_s16(decoder, audiobuf, frames, total_samples);
        total_samples += frames;

        LTCFrameExt frame;
        while (ltc_decoder_read(decoder, &frame)) {
            heartbeat_ltc_frames++;

            /* LTC gap detection (always on) */
            struct timespec now_ts;
            clock_gettime(CLOCK_MONOTONIC, &now_ts);
            if (ltc_active) {
                long gap_ms = (now_ts.tv_sec - last_ltc_ts.tv_sec) * 1000
                            + (now_ts.tv_nsec - last_ltc_ts.tv_nsec) / 1000000;
                if (gap_ms > LTC_GAP_THRESHOLD_MS) {
                    fprintf(stdout, "\n[gap] no LTC decoded for %ldms\n", gap_ms);
                    fflush(stdout);
                }
            }
            last_ltc_ts = now_ts;
            ltc_active = 1;

            SMPTETimecode tc;
            ltc_frame_to_time(&tc, &frame.ltc, 1);

            char tc_str[16];
            snprintf(tc_str, sizeof(tc_str), "%02d:%02d:%02d:%02d",
                     tc.hours, tc.mins, tc.secs, tc.frame);

            /* Only send if timecode changed */
            if (strcmp(tc_str, prev_tc) != 0) {
                strncpy(prev_tc, tc_str, sizeof(prev_tc) - 1);
                prev_tc[sizeof(prev_tc) - 1] = '\0';

                char osc_buf[256];
                int len = osc_build_message(osc_buf, sizeof(osc_buf),
                                            "/clock/ltc", tc_str);
                if (len > 0) {
                    ssize_t sent = sendto(sock, osc_buf, len, 0,
                                          (struct sockaddr *)&dest, sizeof(dest));
                    if (sent < 0) {
                        fprintf(stderr, "Failed to send OSC packet!\n");
                    }
                }

                if (verbose) {
                    putchar('.');
                    fflush(stdout);
                }
            }
        }
    }

cleanup:
    fprintf(stdout, "\n");
    if (audiobuf) {
        free(audiobuf);
        fprintf(stdout, "buffer freed\n");
    }
    if (capture) {
        snd_pcm_drop(capture);
        snd_pcm_close(capture);
        fprintf(stdout, "audio interface closed\n");
    }
    if (decoder)
        ltc_decoder_free(decoder);
    if (sock >= 0)
        close(sock);
    if (device_allocated && device)
        free(device);

    return exit_code;
}
