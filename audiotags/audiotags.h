/***************************************************************************
    copyright            : (C) 2003 by Scott Wheeler
    email                : wheeler@kde.org
 ***************************************************************************/

/***************************************************************************
    copyright            : (C) 2014 by Nick Sellen
    email                : code@nicksellen.co.uk
 ***************************************************************************/

/***************************************************************************
 *   This library is free software; you can redistribute it and/or modify  *
 *   it  under the terms of the GNU Lesser General Public License version  *
 *   2.1 as published by the Free Software Foundation.                     *
 *                                                                         *
 *   This library is distributed in the hope that it will be useful, but   *
 *   WITHOUT ANY WARRANTY; without even the implied warranty of            *
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU     *
 *   Lesser General Public License for more details.                       *
 *                                                                         *
 *   You should have received a copy of the GNU Lesser General Public      *
 *   License along with this library; if not, write to the Free Software   *
 *   Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  *
 *   USA                                                                   *
 ***************************************************************************/
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct { void *fileRef; void *ioStream; } TagLib_FileRefRef;
typedef struct { int dummy; } TagLib_AudioProperties;

extern void goTagPut(int id, char *key, char *val);
extern void goPutImage(int id, char *data, int size);

TagLib_FileRefRef *audiotags_file_new(const char *filename);
TagLib_FileRefRef *audiotags_file_memory(const char *data, unsigned int length);
TagLib_FileRefRef *audiotags_file_memory_with_name(const char *fileName, const char *data, unsigned int length);
void audiotags_file_close(TagLib_FileRefRef *file);
void audiotags_file_properties(const TagLib_FileRefRef *file, int id);
const TagLib_AudioProperties *audiotags_file_audioproperties(const TagLib_FileRefRef *file);
bool audiotags_write_property(TagLib_FileRefRef *file, const char *field_c, const char *value_c);
bool audiotags_write_properties(TagLib_FileRefRef *file, unsigned int len, const char *fields_c[], const char *values_c[]);
bool audiotags_clear_properties(TagLib_FileRefRef *file);

int audiotags_audioproperties_length(const TagLib_AudioProperties *audioProperties);
int audiotags_audioproperties_length_ms(const TagLib_AudioProperties *audioProperties);
int audiotags_audioproperties_bitrate(const TagLib_AudioProperties *audioProperties);
int audiotags_audioproperties_samplerate(const TagLib_AudioProperties *audioProperties);
int audiotags_audioproperties_channels(const TagLib_AudioProperties *audioProperties);

void audiotags_read_picture(TagLib_FileRefRef *fileRefRef, int id);
bool audiotags_write_picture(TagLib_FileRefRef *file, const char *data, unsigned int length, int w, int h, int type);
bool audiotags_remove_pictures(TagLib_FileRefRef *file);

#ifdef __cplusplus
}
#endif
