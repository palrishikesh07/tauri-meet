#include "sphinx_bridge.h"

#include <pocketsphinx/pocketsphinx.h>

ps_decoder_t *meet_ps_open(const char *hmm, const char *lm, const char *dict) {
    cmd_ln_t *config = cmd_ln_init(NULL, ps_args(), TRUE,
                                   "-hmm", hmm,
                                   "-lm", lm,
                                   "-dict", dict,
                                   "-logfn", "/dev/null",
                                   "-samprate", "16000",
                                   NULL);
    if (config == NULL) {
        return NULL;
    }
    ps_decoder_t *ps = ps_init(config);
    if (ps == NULL) {
        cmd_ln_free_r(config);
    }
    return ps;
}

int meet_ps_start(ps_decoder_t *ps) {
    return ps_start_utt(ps);
}

int meet_ps_feed(ps_decoder_t *ps, const int16_t *data, size_t n_samples) {
    return ps_process_raw(ps, data, n_samples, FALSE, FALSE);
}

int meet_ps_in_speech(ps_decoder_t *ps) {
    return ps_get_in_speech(ps) ? 1 : 0;
}

const char *meet_ps_hyp(ps_decoder_t *ps) {
    return ps_get_hyp(ps, NULL);
}

int meet_ps_end(ps_decoder_t *ps) {
    return ps_end_utt(ps);
}

void meet_ps_close(ps_decoder_t *ps) {
    if (ps != NULL) {
        ps_free(ps);
    }
}
