#ifndef MEET_SPHINX_BRIDGE_H
#define MEET_SPHINX_BRIDGE_H

#include <stddef.h>
#include <stdint.h>

typedef struct ps_decoder_s ps_decoder_t;

ps_decoder_t *meet_ps_open(const char *hmm, const char *lm, const char *dict);
int meet_ps_start(ps_decoder_t *ps);
int meet_ps_feed(ps_decoder_t *ps, const int16_t *data, size_t n_samples);
int meet_ps_in_speech(ps_decoder_t *ps);
const char *meet_ps_hyp(ps_decoder_t *ps);
int meet_ps_end(ps_decoder_t *ps);
void meet_ps_close(ps_decoder_t *ps);

#endif
