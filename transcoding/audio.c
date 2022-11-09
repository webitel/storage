#include <stdio.h>

#include <libavutil/opt.h>
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libswresample/swresample.h>

#define RAW_OUT_ON_PLANAR false
#define C_CAST(type, variable) ((type)variable)
#define REINTERPRET_CAST(type, variable) C_CAST(type, variable)
#define STATIC_CAST(type, variable) C_CAST(type, variable)

typedef struct {
  int rate;
  const char *path;
} transcodingConfig;


void goFrameHandler(uintptr_t ctx, uint8_t *samples, int len, int channels);


static void initTranscoding() {
//    av_register_all();
//    av_log_set_level(AV_LOG_TRACE);
    avformat_network_init();
}


/**
 * Print an error string describing the errorCode to stderr.
 */
static int printError(const char* prefix, int errorCode) {
    if(errorCode == 0) {
        return 0;
    } else {
        const size_t bufsize = 64;
        char buf[bufsize];

        if(av_strerror(errorCode, buf, bufsize) != 0) {
            strcpy(buf, "UNKNOWN_ERROR");
        }
        fprintf(stderr, "%s (%d: %s)\n", prefix, errorCode, buf);

        return errorCode;
    }
}


/**
 * Extract a single sample and convert to float.
 */
static float getSample(const AVCodecContext* codecCtx, uint8_t* buffer, int sampleIndex) {
    int64_t val = 0;
    float ret = 0;
    int sampleSize = av_get_bytes_per_sample(codecCtx->sample_fmt);
    switch(sampleSize) {
        case 1:
            // 8bit samples are always unsigned
            val = REINTERPRET_CAST(uint8_t*, buffer)[sampleIndex];
            // make signed
            val -= 127;
            break;

        case 2:
            val = REINTERPRET_CAST(int16_t*, buffer)[sampleIndex];
            break;

        case 4:
            val = REINTERPRET_CAST(int32_t*, buffer)[sampleIndex];
            break;

        case 8:
            val = REINTERPRET_CAST(int64_t*, buffer)[sampleIndex];
            break;

        default:
            fprintf(stderr, "Invalid sample size %d.\n", sampleSize);
            return 0;
    }

    // Check which data type is in the sample.
    switch(codecCtx->sample_fmt) {
        case AV_SAMPLE_FMT_U8:
        case AV_SAMPLE_FMT_S16:
        case AV_SAMPLE_FMT_S32:
        case AV_SAMPLE_FMT_U8P:
        case AV_SAMPLE_FMT_S16P:
        case AV_SAMPLE_FMT_S32P:
            // integer => Scale to [-1, 1] and convert to float.
            ret = val / STATIC_CAST(float, ((1 << (sampleSize*8-1))-1));
            break;

        case AV_SAMPLE_FMT_FLT:
        case AV_SAMPLE_FMT_FLTP:
            // float => reinterpret
            ret = *REINTERPRET_CAST(float*, &val);
            break;

        case AV_SAMPLE_FMT_DBL:
        case AV_SAMPLE_FMT_DBLP:
            // double => reinterpret and then static cast down
            ret = STATIC_CAST(float, *REINTERPRET_CAST(double*, &val));
            break;

        default:
            return 0;
    }

    return ret;
}

/**
 * Write the frame to an output file.
 */
static void handleFrame(uintptr_t ctx, transcodingConfig *conf, const AVCodecContext* codecCtx, const AVFrame* frame, SwrContext *swr ) {
        if(av_sample_fmt_is_planar(codecCtx->sample_fmt) == 1) {
            // This means that the data of each channel is in its own buffer.
            // => frame->extended_data[i] contains data for the i-th channel.
            for(int s = 0; s < frame->nb_samples; ++s) {
                for(int c = 0; c < codecCtx->channels; ++c) {
                    float sample = getSample(codecCtx, frame->extended_data[c], s);
    //                fwrite(&sample, sizeof(float), 1, outFile); // TODO CALLBACK
                }
            }
        } else {
            //Externally supplied data
            const uint8_t* in_samples = frame->extended_data[0];
            int in_num_samples = frame->nb_samples;
            int in_samplerate = frame->sample_rate;
            int out_samplerate = conf->rate;
            int out_num_channels = frame->channels;

            //Perform the resample
            uint8_t* out_samples;
            int out_num_samples = av_rescale_rnd(swr_get_delay(swr, in_samplerate) + in_num_samples, out_samplerate, in_samplerate, AV_ROUND_UP);
            av_samples_alloc(&out_samples, NULL, out_num_channels, out_num_samples, AV_SAMPLE_FMT_FLT, 0);
            out_num_samples = swr_convert(swr, &out_samples, out_num_samples, &in_samples, in_num_samples);


            float *result = malloc(sizeof(float) * out_num_channels * out_num_samples);

            for(int s = 0; s < out_num_samples; ++s) {
                for(int c = 0; c < codecCtx->channels; ++c) {
                    float sample = getSample(codecCtx, &out_samples[0], s*codecCtx->channels+c); //frame->extended_data[0]
                    result[s*codecCtx->channels+c] = sample;
                }
            }
//            buf->data = (float *) realloc(buf->data, (buf->len + (out_num_channels * out_num_samples)) * sizeof(float));
//            memcpy(buf->data + buf->len, result, ((out_num_channels * out_num_samples)) * sizeof(float));
//            buf->len += out_num_channels * out_num_samples;
    //        ChunkCallback(buf, result, out_num_channels * out_num_samples, out_num_channels);
            free(result);
        }
}

/**
 * Find the first audio stream and returns its index. If there is no audio stream returns -1.
 */
static int findAudioStream(const AVFormatContext* formatCtx) {
    int audioStreamIndex = -1;
    for(size_t i = 0; i < formatCtx->nb_streams; ++i) {
        // Use the first audio stream we can find.
        // NOTE: There may be more than one, depending on the file.
        if(formatCtx->streams[i]->codecpar->codec_type == AVMEDIA_TYPE_AUDIO) {
            audioStreamIndex = i;
            break;
        }
    }
    return audioStreamIndex;
}

/**
 * Receive as many frames as available and handle them.
 */
static int receiveAndHandle(uintptr_t ctx, transcodingConfig *conf, AVCodecContext* codecCtx, AVFrame* frame, SwrContext *swr) {
    int err = 0;
    // Read the packets from the decoder.
    // NOTE: Each packet may generate more than one frame, depending on the codec.
    while((err = avcodec_receive_frame(codecCtx, frame)) == 0) {
        // Let's handle the frame in a function.
        handleFrame(ctx, conf, codecCtx, frame, swr);
        // Free any buffers and reset the fields to default values.
        av_frame_unref(frame);
    }
    return err;
}


/*
 * Drain any buffered frames.
 */
static void drainDecoder(uintptr_t ctx, transcodingConfig *conf, AVCodecContext* codecCtx, AVFrame* frame, SwrContext *swr ) {
    int err = 0;
    // Some codecs may buffer frames. Sending NULL activates drain-mode.
    if((err = avcodec_send_packet(codecCtx, NULL)) == 0) {
        // Read the remaining packets from the decoder.
        err = receiveAndHandle(ctx, conf, codecCtx, frame, swr);
        if(err != AVERROR(EAGAIN) && err != AVERROR_EOF) {
            // Neither EAGAIN nor EOF => Something went wrong.
            printError("Receive error.", err);
        }
    } else {
        // Something went wrong.
        printError("Send error.", err);
    }
}

static int transcode(uintptr_t context, const char *path, int rate) {
    int err = 0;
    transcodingConfig conf;
    conf.path = path;
    conf.rate = rate;
    AVFormatContext *formatCtx = NULL;

    if ((err = avformat_open_input(&formatCtx, conf.path, NULL, 0)) != 0) {
        return printError("Error opening file.", err);
    }

    // In case the file had no header, read some frames and find out which format and codecs are used.
    // This does not consume any data. Any read packets are buffered for later use.
    avformat_find_stream_info(formatCtx, NULL);


    // Try to find an audio stream.
    int audioStreamIndex = findAudioStream(formatCtx);
    if(audioStreamIndex == -1) {
        // No audio stream was found.
        avformat_close_input(&formatCtx);
        return -1;
    }

    // Find the correct decoder for the codec.
    AVCodec* codec = avcodec_find_decoder(formatCtx->streams[audioStreamIndex]->codecpar->codec_id);
    if (codec == NULL) {
        // Decoder not found.
        avformat_close_input(&formatCtx);
        return -1;
    }

    // Initialize codec context for the decoder.
    AVCodecContext* codecCtx = avcodec_alloc_context3(codec);
    if (codecCtx == NULL) {
        // Something went wrong. Cleaning up...
        avformat_close_input(&formatCtx);
        return -1;
    }

    // Fill the codecCtx with the parameters of the codec used in the read file.
    if ((err = avcodec_parameters_to_context(codecCtx, formatCtx->streams[audioStreamIndex]->codecpar)) != 0) {
        // Something went wrong. Cleaning up...
        avcodec_close(codecCtx);
        avcodec_free_context(&codecCtx);
        avformat_close_input(&formatCtx);
        return printError("Error setting codec context parameters.", err);
    }

    // Explicitly request non planar data.
    codecCtx->request_sample_fmt = av_get_alt_sample_fmt(codecCtx->sample_fmt, 0);

    // Initialize the decoder.
    if ((err = avcodec_open2(codecCtx, codec, NULL)) != 0) {
        avcodec_close(codecCtx);
        avcodec_free_context(&codecCtx);
        avformat_close_input(&formatCtx);
        return -1;
    }

    AVFrame* frame = NULL;
    if ((frame = av_frame_alloc()) == NULL) {
        avcodec_close(codecCtx);
        avcodec_free_context(&codecCtx);
        avformat_close_input(&formatCtx);
        return -1;
    }

    // Prepare the packet.
    AVPacket packet;
    // Set default values.
    av_init_packet(&packet);

    // prepare resampler
    SwrContext *swr = swr_alloc();
    av_opt_set_int(swr, "in_channel_count", codecCtx->channels, 0);
    av_opt_set_int(swr, "out_channel_count", codecCtx->channels, 0);
    av_opt_set_int(swr, "in_channel_layout", codecCtx->channel_layout, 0);
    av_opt_set_int(swr, "out_channel_layout", codecCtx->channel_layout, 0);
    av_opt_set_int(swr, "in_sample_rate", codecCtx->sample_rate, 0);
    av_opt_set_int(swr, "out_sample_rate", conf.rate, 0);
    av_opt_set_sample_fmt(swr, "in_sample_fmt", codecCtx->sample_fmt, 0);
    av_opt_set_sample_fmt(swr, "out_sample_fmt", AV_SAMPLE_FMT_FLT, 0);
    swr_init(swr);
    if (!swr_is_initialized(swr)) {
        fprintf(stderr, "Resampler has not been properly initialized\n");
        avcodec_close(codecCtx);
        avcodec_free_context(&codecCtx);
        avformat_close_input(&formatCtx);
        return -1;
    }

    while ((err = av_read_frame(formatCtx, &packet)) != AVERROR_EOF) {
        if(err != 0) {
            // Something went wrong.
            printError("Read error.", err);
            break; // Don't return, so we can clean up nicely.
        }
        // Does the packet belong to the correct stream?
        if(packet.stream_index != audioStreamIndex) {
            // Free the buffers used by the frame and reset all fields.
            av_packet_unref(&packet);
            continue;
        }
        // We have a valid packet => send it to the decoder.
        if((err = avcodec_send_packet(codecCtx, &packet)) == 0) {
            // The packet was sent successfully. We don't need it anymore.
            // => Free the buffers used by the frame and reset all fields.
            av_packet_unref(&packet);
        } else {
            // Something went wrong.
            // EAGAIN is technically no error here but if it occurs we would need to buffer
            // the packet and send it again after receiving more frames. Thus we handle it as an error here.
            printError("Send error.", err);
            break; // Don't return, so we can clean up nicely.
        }

        // Receive and handle frames.
        // EAGAIN means we need to send before receiving again. So thats not an error.
        if((err = receiveAndHandle(context, &conf, codecCtx, frame, swr)) != AVERROR(EAGAIN)) {
            // Not EAGAIN => Something went wrong.
            printError("Receive error.", err);
            break; // Don't return, so we can clean up nicely.
        }
    }

    return 0;
}