/*
 * alsa-ltc — ALSA audio LTC timecode decoder with OSC output
 *
 * Captures audio from an ALSA device, decodes LTC (Linear Timecode)
 * using libltc, and sends the timecode as OSC messages to a specified
 * destination. Designed for use with clock-8001/8002.
 *
 * Usage: alsa-ltc <alsa-device> <OSC destination ip> <OSC port>
 *        Use "-" for device to auto-detect on Raspberry Pi (bcm2835)
 *
 * Build: gcc -O2 -o alsa-ltc alsa-ltc.c -lasound -lltc
 * Dependencies: libasound2-dev, libltc-dev
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <unistd.h>
#include <arpa/inet.h>
#include <sys/socket.h>
#include <alsa/asoundlib.h>
#include <ltc.h>

#define SAMPLE_RATE 48000
#define FPS         25
#define CHANNELS    1
#define BUF_SIZE    1024

static volatile int running = 1;

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

/* Open, configure, and prepare an ALSA capture device.
 * Returns a ready-to-use capture handle, or NULL on failure.
 * On failure, the device is always closed before returning.
 */
static snd_pcm_t *setup_capture(const char *device) {
    snd_pcm_t *capture = NULL;
    snd_pcm_hw_params_t *hw_params = NULL;
    int rc;

    rc = snd_pcm_open(&capture, device, SND_PCM_STREAM_CAPTURE, 0);
    if (rc < 0) {
        fprintf(stderr, "cannot open audio device %s (%s)\n", device, snd_strerror(rc));
        return NULL;
    }
    fprintf(stdout, "audio interface opened\n");

    rc = snd_pcm_hw_params_malloc(&hw_params);
    if (rc < 0) {
        fprintf(stderr, "cannot allocate hardware parameter structure (%s)\n", snd_strerror(rc));
        goto fail;
    }
    fprintf(stdout, "hw_params allocated\n");

    rc = snd_pcm_hw_params_any(capture, hw_params);
    if (rc < 0) {
        fprintf(stderr, "cannot initialize hardware parameter structure (%s)\n", snd_strerror(rc));
        goto fail;
    }
    fprintf(stdout, "hw_params initialized\n");

    rc = snd_pcm_hw_params_set_access(capture, hw_params, SND_PCM_ACCESS_RW_INTERLEAVED);
    if (rc < 0) {
        fprintf(stderr, "cannot set access type (%s)\n", snd_strerror(rc));
        goto fail;
    }
    fprintf(stdout, "hw_params access setted\n");

    rc = snd_pcm_hw_params_set_format(capture, hw_params, SND_PCM_FORMAT_S16_LE);
    if (rc < 0) {
        fprintf(stderr, "cannot set sample format (%s)\n", snd_strerror(rc));
        goto fail;
    }
    fprintf(stdout, "hw_params format setted\n");

    unsigned int rate = SAMPLE_RATE;
    rc = snd_pcm_hw_params_set_rate_near(capture, hw_params, &rate, 0);
    if (rc < 0) {
        fprintf(stderr, "cannot set sample rate (%s)\n", snd_strerror(rc));
        goto fail;
    }
    fprintf(stdout, "hw_params rate setted\n");

    rc = snd_pcm_hw_params_set_channels(capture, hw_params, CHANNELS);
    if (rc < 0) {
        fprintf(stderr, "cannot set channel count (%s)\n", snd_strerror(rc));
        goto fail;
    }
    fprintf(stdout, "hw_params channels setted\n");

    rc = snd_pcm_hw_params(capture, hw_params);
    if (rc < 0) {
        fprintf(stderr, "cannot set parameters (%s)\n", snd_strerror(rc));
        goto fail;
    }
    fprintf(stdout, "hw_params setted\n");

    snd_pcm_hw_params_free(hw_params);
    hw_params = NULL;
    fprintf(stdout, "hw_params freed\n");

    rc = snd_pcm_prepare(capture);
    if (rc < 0) {
        fprintf(stderr, "cannot prepare audio interface for use (%s)\n", snd_strerror(rc));
        goto fail;
    }
    fprintf(stdout, "audio interface prepared\n");
    return capture;

fail:
    if (hw_params) snd_pcm_hw_params_free(hw_params);
    snd_pcm_close(capture);
    fprintf(stdout, "audio interface closed\n");
    return NULL;
}

int main(int argc, char *argv[]) {
    LTCDecoder *decoder = NULL;
    short *audiobuf = NULL;
    int sock = -1;
    char *device = NULL;
    int device_allocated = 0;

    if (argc != 4) {
        fprintf(stderr, "Usage:  %s <alsa-device> <OSC destination ip> <OSC port>\n", argv[0]);
        fprintf(stderr, "Use - for device for automatic detection on raspberry pi\n");
        return 1;
    }

    const char *osc_ip = argv[2];
    int osc_port = atoi(argv[3]);

    signal(SIGINT, sighandler);
    signal(SIGTERM, sighandler);

    /* Device selection */
    if (strcmp(argv[1], "-") == 0) {
        device = detect_device();
        if (device == NULL) {
            fprintf(stderr, "No suitable capture device found\n");
            return 1;
        }
        device_allocated = 1;
    } else {
        device = argv[1];
    }

    /* Allocate audio buffer */
    audiobuf = malloc(BUF_SIZE * CHANNELS * sizeof(short));
    if (audiobuf == NULL) {
        fprintf(stderr, "cannot allocate audio buffer\n");
        goto cleanup;
    }
    fprintf(stdout, "buffer allocated\n");

    /* Initialize LTC decoder */
    decoder = ltc_decoder_create(SAMPLE_RATE * FPS / BUF_SIZE, 32);
    if (decoder == NULL) {
        fprintf(stderr, "cannot create LTC decoder\n");
        goto cleanup;
    }
    fprintf(stdout, "LTC decoder initialized: sample rate: %d, fps: %d / %d\n",
            SAMPLE_RATE, FPS, BUF_SIZE);

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

    /* Outer retry loop — reopens the device after any setup or persistent read failure.
     * The process stays alive so systemd doesn't thrash the USB device with rapid restarts.
     * A 30-second delay between reopens matches the old RestartSec=30 behaviour. */
    int open_attempt = 0;
    while (running) {
        /* Delay between retries to let the USB device recover */
        if (open_attempt > 0) {
            fprintf(stderr, "waiting 30s before reopening (attempt %d)\n", open_attempt + 1);
            for (int i = 0; i < 30 && running; i++)
                sleep(1);
        }
        if (!running) break;

        snd_pcm_t *capture = setup_capture(device);
        if (capture == NULL) {
            open_attempt++;
            continue;
        }
        open_attempt = 0;

        /* Inner capture loop */
        char prev_tc[16] = "";
        ltc_off_t total_samples = 0;
        int consecutive_errors = 0;
        int need_reopen = 0;

        while (running && !need_reopen) {
            snd_pcm_sframes_t frames = snd_pcm_readi(capture, audiobuf, BUF_SIZE);
            if (frames < 0) {
                consecutive_errors++;
                fprintf(stderr, "read from audio interface failed (%s) [%d]\n",
                        snd_strerror(frames), consecutive_errors);
                sleep(1);
                /* Try a soft recover — don't close the device on prepare failure as
                 * closing a USB device triggers a kernel reset that causes ETIMEDOUT
                 * on the next open. Only reopen after 10 consecutive read failures. */
                snd_pcm_prepare(capture);
                if (consecutive_errors >= 10) {
                    fprintf(stderr, "reopening audio device after %d consecutive errors\n",
                            consecutive_errors);
                    need_reopen = 1;
                }
                continue;
            }
            consecutive_errors = 0;

            ltc_decoder_write_s16(decoder, audiobuf, frames, total_samples);
            total_samples += frames;

            LTCFrameExt frame;
            while (ltc_decoder_read(decoder, &frame)) {
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

                    fflush(stdout);
                }
            }
        }

        snd_pcm_close(capture);
        fprintf(stdout, "audio interface closed\n");
        if (need_reopen)
            open_attempt = 1; /* will sleep before next attempt */
    }

cleanup:
    fprintf(stdout, "\n");
    if (audiobuf) {
        free(audiobuf);
        fprintf(stdout, "buffer freed\n");
    }
    if (decoder)
        ltc_decoder_free(decoder);
    if (sock >= 0)
        close(sock);
    if (device_allocated && device)
        free(device);

    return 0;
}
