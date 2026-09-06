#include "litert_lm_c_api.h"
#if __has_include("csrc/engine.h")
#include "csrc/engine.h"
#elif __has_include("engine.h")
#include "engine.h"
#else
#include "../../../include/engine.h"
#endif

#include <cstdlib>
#include <cstring>
#include <string>

struct LiteRTLMEngine {
    LiteRtLmEngine* engine;
};

extern "C" {

LiteRTLMEngine* litert_lm_create_engine(const LiteRTLMConfig* config, char** err_msg) {
    if (!config || !config->model_path) {
        if (err_msg) *err_msg = strdup("Invalid model configuration or path");
        return nullptr;
    }

    const char* backend_str = (config->backend == 1) ? "gpu" : "cpu";
    LiteRtLmEngineSettings* settings = litert_lm_engine_settings_create(
        config->model_path, backend_str, nullptr, nullptr);
    if (!settings) {
        if (err_msg) *err_msg = strdup("Failed to create LiteRT-LM engine settings");
        return nullptr;
    }

    if (config->max_seq_len > 0) {
        litert_lm_engine_settings_set_max_num_tokens(settings, config->max_seq_len);
    }

    LiteRtLmEngine* engine = litert_lm_engine_create(settings);
    litert_lm_engine_settings_delete(settings);

    if (!engine) {
        if (err_msg) *err_msg = strdup("Failed to create LiteRT-LM engine instance");
        return nullptr;
    }

    LiteRTLMEngine* wrapper = new LiteRTLMEngine();
    wrapper->engine = engine;
    return wrapper;
}

void litert_lm_free_engine(LiteRTLMEngine* engine) {
    if (engine) {
        if (engine->engine) {
            litert_lm_engine_delete(engine->engine);
        }
        delete engine;
    }
}

struct StreamAdapterCtx {
    LiteRTLMTokenCallback callback;
    void* user_data;
    bool should_continue;
};

static void stream_callback_gateway(void* callback_data, const LiteRtLmStreamChunk* chunk) {
    if (!callback_data || !chunk) return;
    StreamAdapterCtx* ctx = static_cast<StreamAdapterCtx*>(callback_data);
    if (!ctx->should_continue) return;

    const char* text = litert_lm_stream_chunk_get_text(chunk);
    bool is_final = litert_lm_stream_chunk_is_final(chunk);
    
    if (ctx->callback) {
        bool cont = ctx->callback(text ? text : "", is_final, ctx->user_data);
        if (!cont) {
            ctx->should_continue = false;
        }
    }
}

bool litert_lm_generate(
    LiteRTLMEngine* engine,
    const char* prompt,
    LiteRTLMTokenCallback callback,
    void* user_data,
    char** err_msg
) {
    if (!engine || !engine->engine || !prompt) {
        if (err_msg) *err_msg = strdup("Engine or prompt is null");
        return false;
    }

    LiteRtLmSessionConfig* session_cfg = litert_lm_session_config_create();
    if (!session_cfg) {
        if (err_msg) *err_msg = strdup("Failed to create LiteRT-LM session config");
        return false;
    }

    litert_lm_session_config_set_apply_prompt_template(session_cfg, false);

    LiteRtLmSession* session = litert_lm_engine_create_session(engine->engine, session_cfg);
    litert_lm_session_config_delete(session_cfg);

    if (!session) {
        if (err_msg) *err_msg = strdup("Failed to create LiteRT-LM session");
        return false;
    }

    LiteRtLmInputData* input = litert_lm_input_data_create(
        kLiteRtLmInputDataTypeText, prompt, strlen(prompt));
    if (!input) {
        litert_lm_session_delete(session);
        if (err_msg) *err_msg = strdup("Failed to create LiteRT-LM text input data");
        return false;
    }

    StreamAdapterCtx ctx;
    ctx.callback = callback;
    ctx.user_data = user_data;
    ctx.should_continue = true;

    const LiteRtLmInputData* inputs[1] = { input };
    int ret = litert_lm_session_generate_content_stream(
        session, inputs, 1, stream_callback_gateway, &ctx);

    litert_lm_input_data_delete(input);
    litert_lm_session_delete(session);

    if (ret != 0) {
        if (err_msg) *err_msg = strdup("Failed during LiteRT-LM content generation");
        return false;
    }

    return true;
}

} // extern "C"
