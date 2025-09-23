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

#include <stdlib.h>
#include <fileref.h>
#include <flacfile.h>
#include <flacpicture.h>
#include <mp4file.h>
#include <id3v2tag.h>
#include <tbytevector.h>
#include <tbytevectorstream.h>
#include <tfile.h>
#include <tlist.h>
#include <tpropertymap.h>
#include <attachedpictureframe.h>
#include <string.h>
#include <typeinfo>
#include <apefile.h>
#include <apetag.h>
#include <id3v1tag.h>
#include <xiphcomment.h>
#include <mpegfile.h>

#include "audiotags.h"

static bool unicodeStrings = true;

class ByteVectorStreamWithName : public TagLib::ByteVectorStream
{
public:
  ByteVectorStreamWithName(const char *name, const TagLib::ByteVector &data) : TagLib::ByteVectorStream(data)
  {
    this->fileName = TagLib::FileName(name);
  }
  TagLib::FileName name() const
  {
    return this->fileName;
  }

private:
  TagLib::FileName fileName;
};

TagLib_FileRefRef *audiotags_file_new(const char *filename)
{
  TagLib::FileRef *fr = new TagLib::FileRef(filename);
  if (fr == NULL || fr->isNull() || !fr->file()->isValid() || fr->tag() == NULL)
  {
    if (fr)
    {
      delete fr;
      fr = NULL;
    }
    return NULL;
  }

  TagLib_FileRefRef *holder = new TagLib_FileRefRef();
  holder->fileRef = reinterpret_cast<void *>(fr);
  holder->ioStream = NULL;
  return holder;
}

TagLib_FileRefRef *audiotags_file_memory(const char *data, unsigned int length)
{
  TagLib::ByteVectorStream *ioStream = new TagLib::ByteVectorStream(TagLib::ByteVector(data, length));
  TagLib::FileRef *fr = new TagLib::FileRef(ioStream);
  if (fr == NULL || fr->isNull() || !fr->file()->isValid() || fr->tag() == NULL)
  {
    if (fr)
    {
      delete fr;
      fr = NULL;
    }
    if (ioStream)
    {
      delete ioStream;
      ioStream = NULL;
    }
    return NULL;
  }

  TagLib_FileRefRef *holder = new TagLib_FileRefRef();
  holder->fileRef = reinterpret_cast<void *>(fr);
  holder->ioStream = reinterpret_cast<void *>(ioStream);
  return holder;
}

TagLib_FileRefRef *audiotags_file_memory_with_name(const char *fileName, const char *data, unsigned int length)
{
  TagLib::ByteVectorStream *ioStream = new ByteVectorStreamWithName(fileName, TagLib::ByteVector(data, length));
  TagLib::FileRef *fr = new TagLib::FileRef(ioStream);
  if (fr == NULL || fr->isNull() || !fr->file()->isValid() || fr->tag() == NULL)
  {
    if (fr)
    {
      delete fr;
      fr = NULL;
    }
    if (ioStream)
    {
      delete ioStream;
      ioStream = NULL;
    }
    return NULL;
  }

  TagLib_FileRefRef *holder = new TagLib_FileRefRef();
  holder->fileRef = reinterpret_cast<void *>(fr);
  holder->ioStream = reinterpret_cast<void *>(ioStream);
  return holder;
}

void audiotags_file_close(TagLib_FileRefRef *fileRefRef)
{
  delete reinterpret_cast<TagLib::FileRef *>(fileRefRef->fileRef);
  if (fileRefRef->ioStream)
  {
    delete reinterpret_cast<TagLib::IOStream *>(fileRefRef->ioStream);
  }
  delete fileRefRef;
}

void process_tags(const TagLib::PropertyMap &tags, int id)
{
  for (TagLib::PropertyMap::ConstIterator i = tags.begin(); i != tags.end(); ++i)
  {
    for (TagLib::StringList::ConstIterator j = i->second.begin(); j != i->second.end(); ++j)
    {
      char *key = ::strdup(i->first.toCString(unicodeStrings));
      char *val = ::strdup((*j).toCString(unicodeStrings));
      goTagPut(id, key, val);
      free(key);
      free(val);
    }
  }
}

void audiotags_file_properties(const TagLib_FileRefRef *fileRefRef, int id)
{
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);

  if (TagLib::MPEG::File *mpeg = dynamic_cast<TagLib::MPEG::File *>(fileRef->file()))
  {
    if (auto id3v2Tag = mpeg->ID3v2Tag(false))
    {
      process_tags(id3v2Tag->properties(), id);
    }
    else if (auto id3v1Tag = mpeg->ID3v1Tag(false))
    {
      process_tags(id3v1Tag->properties(), id);
    }
  }
  else
  {
    process_tags(fileRef->file()->properties(), id);
  }
}

bool audiotags_clear_properties(TagLib_FileRefRef *fileRefRef)
{
  TagLib::FileRef *f = reinterpret_cast<TagLib::FileRef *>(fileRefRef->fileRef);
  TagLib::Tag *tag = f->tag();

  TagLib::PropertyMap properties = f->file()->properties();
  properties.clear();
  f->file()->setProperties(properties);

  f->file()->save();

  return true;
}

bool audiotags_write_properties(TagLib_FileRefRef *fileRefRef, unsigned int len, const char *fields_c[], const char *values_c[])
{
  TagLib::FileRef *f = reinterpret_cast<TagLib::FileRef *>(fileRefRef->fileRef);
  TagLib::Tag *tag = f->tag();

  TagLib::PropertyMap properties = f->file()->properties();
  properties.clear();
  f->file()->setProperties(properties);

  for (uint i = 0; i < len; i++)
  {
    TagLib::String field(fields_c[i], TagLib::String::Type::UTF8);
    TagLib::String value(values_c[i], TagLib::String::Type::UTF8);

    TagLib::PropertyMap properties = f->file()->properties();
    TagLib::StringList values = value.split('\003');
    for (const auto &v : values)
      properties.insert(field, v);
    f->file()->setProperties(properties);
  }

  f->file()->save();

  return true;
}

const TagLib_AudioProperties *audiotags_file_audioproperties(const TagLib_FileRefRef *fileRefRef)
{
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);
  return reinterpret_cast<const TagLib_AudioProperties *>(fileRef->file()->audioProperties());
}

const TagLib::AudioProperties *props(const TagLib_AudioProperties *audioProperties)
{
  return reinterpret_cast<const TagLib::AudioProperties *>(audioProperties);
}

int audiotags_audioproperties_length(const TagLib_AudioProperties *audioProperties)
{
  return props(audioProperties)->lengthInSeconds();
}

int audiotags_audioproperties_length_ms(const TagLib_AudioProperties *audioProperties)
{
  return props(audioProperties)->lengthInMilliseconds();
}

int audiotags_audioproperties_bitrate(const TagLib_AudioProperties *audioProperties)
{
  return props(audioProperties)->bitrate();
}

int audiotags_audioproperties_samplerate(const TagLib_AudioProperties *audioProperties)
{
  return props(audioProperties)->sampleRate();
}

int audiotags_audioproperties_channels(const TagLib_AudioProperties *audioProperties)
{
  return props(audioProperties)->channels();
}

enum img_type
{
  JPEG = 0,
  PNG = 1,
  // to be continued...
};

bool audiotags_has_picture(TagLib_FileRefRef *fileRefRef)
{
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);

  TagLib::ByteVector imageData;
  if (TagLib::FLAC::File *flac = dynamic_cast<TagLib::FLAC::File *>(fileRef->file()))
  {
    auto pictures = flac->pictureList();
    for (auto it = pictures.begin(); it != pictures.end(); ++it)
    {
      if ((*it)->type() == TagLib::FLAC::Picture::Type::FrontCover)
      {
        return true;
      }
    }
  }
  else if (TagLib::APE::File *ape = dynamic_cast<TagLib::APE::File *>(fileRef->file()))
  {
    if (auto apeTag = ape->APETag(false))
    {
      printf("\nape tag !!\n");
    }
  }
  else if (TagLib::MPEG::File *mpeg = dynamic_cast<TagLib::MPEG::File *>(fileRef->file()))
  {
    if (auto id3v2Tag = mpeg->ID3v2Tag(false))
    {
      auto frames = id3v2Tag->frameList();
      for (auto it = frames.begin(); it != frames.end(); ++it)
      {
        if (auto *pFrame = dynamic_cast<TagLib::ID3v2::AttachedPictureFrame *>(*it))
        {
          return true;
        }
      }
    }
  }
  else
  {
    auto tags = fileRef->file()->tag();
    if (auto mp4Tag = dynamic_cast<TagLib::MP4::Tag *>(tags))
    {
      TagLib::MP4::CoverArtList covers = mp4Tag->item("covr").toCoverArtList();
      if (!covers.isEmpty())
      {
        return true;
      }
    }
    else if (auto oggTag = dynamic_cast<TagLib::Ogg::XiphComment *>(tags))
    {
      auto pictures = oggTag->pictureList();
      for (auto it = pictures.begin(); it != pictures.end(); ++it)
      {
        if ((*it)->type() == TagLib::FLAC::Picture::Type::FrontCover)
        {
          return true;
        }
      }
    }
    else if (auto id3Tag = dynamic_cast<TagLib::ID3v2::Tag *>(tags))
    {
      auto frames = id3Tag->frameList();
      for (auto it = frames.begin(); it != frames.end(); ++it)
      {
        if (auto *pFrame = dynamic_cast<TagLib::ID3v2::AttachedPictureFrame *>(*it))
        {
          return true;
        }
      }
    }
  }
  return false;
}

void audiotags_read_picture(TagLib_FileRefRef *fileRefRef, int id)
{
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);

  TagLib::ByteVector imageData;
  if (TagLib::FLAC::File *flac = dynamic_cast<TagLib::FLAC::File *>(fileRef->file()))
  {
    auto pictures = flac->pictureList();
    for (auto it = pictures.begin(); it != pictures.end(); ++it)
    {
      if ((*it)->type() == TagLib::FLAC::Picture::Type::FrontCover)
      {
        imageData = (*it)->data();
        break;
      }
    }
  }
  else if (TagLib::APE::File *ape = dynamic_cast<TagLib::APE::File *>(fileRef->file()))
  {
    if (auto apeTag = ape->APETag(false))
    {
      printf("\nape tag !!\n");
    }
  }
  else if (TagLib::MPEG::File *mpeg = dynamic_cast<TagLib::MPEG::File *>(fileRef->file()))
  {
    if (auto id3v2Tag = mpeg->ID3v2Tag(false))
    {
      auto frames = id3v2Tag->frameList();
      for (auto it = frames.begin(); it != frames.end(); ++it)
      {
        if (auto *pFrame = dynamic_cast<TagLib::ID3v2::AttachedPictureFrame *>(*it))
        {
          imageData = pFrame->picture();
          break;
        }
      }
    }
  }
  else
  {
    auto tags = fileRef->file()->tag();
    if (auto mp4Tag = dynamic_cast<TagLib::MP4::Tag *>(tags))
    {
      TagLib::MP4::CoverArtList covers = mp4Tag->item("covr").toCoverArtList();
      if (!covers.isEmpty())
      {
        imageData = covers.front().data();
      }
    }
    else if (auto oggTag = dynamic_cast<TagLib::Ogg::XiphComment *>(tags))
    {
      auto pictures = oggTag->pictureList();
      for (auto it = pictures.begin(); it != pictures.end(); ++it)
      {
        if ((*it)->type() == TagLib::FLAC::Picture::Type::FrontCover)
        {
          imageData = (*it)->data();
          break;
        }
      }
    }
    else if (auto id3Tag = dynamic_cast<TagLib::ID3v2::Tag *>(tags))
    {
      auto frames = id3Tag->frameList();
      for (auto it = frames.begin(); it != frames.end(); ++it)
      {
        if (auto *pFrame = dynamic_cast<TagLib::ID3v2::AttachedPictureFrame *>(*it))
        {
          imageData = pFrame->picture();
          break;
        }
      }
    }
  }
  if (!imageData.isEmpty())
  {
    goPutImage(id, imageData.data(), imageData.size());
  }
}

bool audiotags_write_picture(TagLib_FileRefRef *fileRefRef, const char *data, unsigned int length, int w, int h, int type)
{
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);
  const TagLib::ByteVector byte_vec = TagLib::ByteVector(data, length);

  // check which type the file is (flac, mp4, etc)
  if (TagLib::FLAC::File *flac = dynamic_cast<TagLib::FLAC::File *>(fileRef->file()))
  {
    TagLib::FLAC::Picture *pic = new TagLib::FLAC::Picture;
    // only front cover type supported for now
    pic->setType(TagLib::FLAC::Picture::Type::FrontCover);
    if (type == PNG)
    {
      pic->setMimeType("image/png");
    }
    else if (type == JPEG)
    {
      pic->setMimeType("image/jpeg");
    }
    else
    {
      return false;
    }

    pic->setData(byte_vec);
    pic->setWidth(w);
    pic->setHeight(h);
    pic->setColorDepth(24);
    pic->setNumColors(16777216);
    flac->addPicture(pic);
    flac->save();
  }
  else if (TagLib::MP4::File *mp4 = dynamic_cast<TagLib::MP4::File *>(fileRef->file()))
  {
    TagLib::MP4::CoverArt::Format fmt = TagLib::MP4::CoverArt::Format::Unknown;
    if (type == PNG)
    {
      fmt = TagLib::MP4::CoverArt::Format::PNG;
    }
    else if (type == JPEG)
    {
      fmt = TagLib::MP4::CoverArt::Format::JPEG;
    }
    else
    {
      return false;
    }

    TagLib::MP4::CoverArtList l = mp4->tag()->item("covr").toCoverArtList();
    l.prepend(TagLib::MP4::CoverArt(fmt, byte_vec));
    mp4->tag()->setItem("covr", l);
    mp4->save();
  }
  else
  {
    return false;
  }
  return true;
}

bool audiotags_remove_pictures(TagLib_FileRefRef *fileRefRef)
{
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);

  // check which type the file is (flac, mp4, etc)
  if (TagLib::FLAC::File *flac = dynamic_cast<TagLib::FLAC::File *>(fileRef->file()))
  {
    flac->removePictures();
    flac->save();
  }
  else if (TagLib::MP4::File *mp4 = dynamic_cast<TagLib::MP4::File *>(fileRef->file()))
  {
    mp4->tag()->removeItem("covr");
    mp4->save();
  }
  else
  {
    return false;
  }
  return true;
}
