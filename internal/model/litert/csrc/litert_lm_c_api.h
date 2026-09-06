#ifndef LITERT_LM_C_API_H_
#define LITERT_LM_C_API_H_

#include <stdbool.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct LiteRTLMEngine LiteRTLMEngine;

typedef struct {
    const char* model_path;
    int backend;              // 0: CPU, 1: GPU
    int max_seq_len;
    float temperature;
    float top_p;
    int top_k;
} LiteRTLMConfig;

// Token callback for streaming responses
// Returns true to continue generating, false to cancel/interrupt
typedef bool (*LiteRTLMTokenCallback)(const char* token, bool is_done, void* user_data);

// Create and load model into memory
LiteRTLMEngine* litert_lm_create_engine(const LiteRTLMConfig* config, char** err_msg);

// Free engine and unload memory
void litert_lm_free_engine(LiteRTLMEngine* engine);

// Generate tokens given a prompt
bool litert_lm_generate(
    LiteRTLMEngine* engine,
    const char* prompt,
    LiteRTLMTokenCallback callback,
    void* user_data,
    char** err_msg
);

#ifdef __cplusplus
}
#endif

#endif  // LITERT_LM_C_API_H_
