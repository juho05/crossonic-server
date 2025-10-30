#ifndef AUDIOTAGS_H
#define AUDIOTAGS_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdbool.h>
#include <stdio.h>

typedef struct { void *fileRef; void *ioStream; } TagLib_FileRefRef;

typedef struct {
    char* key;
    char* value;
} KeyValue;

typedef struct {
    KeyValue* tags;
    int size;
} TagMap;

typedef struct {
    TagMap* tags;

    int lengthMs;
    int bitRate;
    int sampleRate;
    int channels;

    int hasImage;
} Metadata;

extern void goPutImage(int id, char *data, int size);

Metadata* read(const char *filename, int checkHasImage);
void free_metadata(Metadata* metadata);
void read_picture(const char* filename, int id);
int write_tag(const char* filename, const char* key, const char* value);
int remove_crossonic_id(const char* filename, const char* instanceId);

TagLib_FileRefRef *audiotags_file_new(const char *filename);
void audiotags_file_close(TagLib_FileRefRef *file);
TagMap* audiotags_file_properties(const TagLib_FileRefRef *file);

bool audiotags_has_picture(TagLib_FileRefRef *fileRefRef);

#ifdef __cplusplus
}
#endif

#endif